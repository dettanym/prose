package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/proto-gen/api_v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"privacy-profile-composer/pkg/composer"
	proto "privacy-profile-composer/pkg/proto"
)

// ─── Profile types imported from composer package ─────────────────────────────
// All rich profile struct types are defined in composer.go so they can be
// used by both main.go and composer.go without circular imports.
// We alias them here for brevity.

type PIIType = proto.PIIType
type ObservedPIITypes = composer.ObservedPIITypes
type IncomingEntry = composer.IncomingEntry
type IndirectProcessingInfo = composer.IndirectProcessingInfo
type IndirectEntry = composer.IndirectEntry
type SharedProcessingInfo = composer.SharedProcessingInfo
type SharedEntry = composer.SharedEntry
type OutgoingEntries = composer.OutgoingEntries
type EndpointProfileData = composer.EndpointProfileData
type EndpointEntry = composer.EndpointEntry
type EndpointsList = composer.EndpointsList
type SvcObservedProfileLocal = composer.SvcObservedProfileLocal

// piiStringToEnum converts a Presidio string like "PERSON" to proto.PIIType.
func piiStringToEnum(piiStr string) proto.PIIType {
	if val, ok := proto.PIIType_value[piiStr]; ok {
		return proto.PIIType(val)
	}
	return -1
}

// piiStringsToEnums converts a slice of PII strings to []proto.PIIType.
func piiStringsToEnums(strs []string) []proto.PIIType {
	var result []proto.PIIType
	for _, s := range strs {
		t := piiStringToEnum(s)
		if t >= 0 {
			result = append(result, t)
		}
	}
	return result
}

// profileStore accumulates SvcObservedProfileLocal per service FQDN.
var profileStore = map[string]*SvcObservedProfileLocal{}

// getOrCreateProfile returns or creates a SvcObservedProfileLocal for a service.
func getOrCreateProfile(serviceFQDN, purposeOfUse string) *SvcObservedProfileLocal {
	if p, ok := profileStore[serviceFQDN]; ok {
		return p
	}
	p := &SvcObservedProfileLocal{
		SvcInternalFQDN:  serviceFQDN,
		TargetPolicyHash: "%POLICY_FILE_HASH%",
		ServiceHash:      "%SERVICE_IMAGE_HASH%",
	}
	profileStore[serviceFQDN] = p
	return p
}

// endpointNameFromOperationName extracts the endpoint name from an Envoy SP
// operationName e.g. "authentication Login" → "Login"
func endpointNameFromOperationName(operationName string) string {
	parts := strings.Fields(operationName)
	if len(parts) == 0 {
		return operationName
	}
	return parts[len(parts)-1]
}

// findOrCreateEndpoint returns or appends an endpoint entry by name.
func findOrCreateEndpoint(profile *SvcObservedProfileLocal, endpointName string) *EndpointEntry {
	for i := range profile.Endpoints.Endpoint {
		if profile.Endpoints.Endpoint[i].EndpointName == endpointName {
			return &profile.Endpoints.Endpoint[i]
		}
	}
	profile.Endpoints.Endpoint = append(profile.Endpoints.Endpoint,
		EndpointEntry{
			EndpointName: endpointName,
			EndpointHash: "%OBJECT_HASH%",
		},
	)
	return &profile.Endpoints.Endpoint[len(profile.Endpoints.Endpoint)-1]
}

// populateServiceProfile maps one PROSE span into the correct bucket of the
// correct service's SvcObservedProfileLocal using local named struct types.
func populateServiceProfile(
	span model.Span,
	spanStore map[model.SpanID]model.Span,
) error {
	tags := model.KeyValues(span.GetTags())

	directionTag, hasDir := tags.FindByKey("prose_sidecar_direction")
	if !hasDir {
		return nil
	}

	dataFlowTag, hasFlow := tags.FindByKey("prose_data_flow")
	var dataFlow string
	if hasFlow {
		dataFlow = dataFlowTag.VStr
	} else {
		opName := strings.ToLower(span.OperationName)
		switch {
		case strings.Contains(opName, "encodedata"):
			dataFlow = "ENCODE_DATA"
		case strings.Contains(opName, "decodedata"):
			dataFlow = "DECODE_DATA"
		default:
			return nil
		}
	}
	if dataFlow != "DECODE_DATA" && dataFlow != "ENCODE_DATA" {
		return nil
	}

	_, hasPii := tags.FindByKey("prose_pii_types")
	_, hasPiiCompliant := tags.FindByKey("prose_pii_compliant")
	_, hasPiiViolation := tags.FindByKey("prose_pii_violation")
	_, hasOpa := tags.FindByKey("prose_opa_decision")
	if !hasPii && !hasPiiCompliant && !hasPiiViolation && !hasOpa {
		return nil
	}

	isInbound := strings.Contains(directionTag.VStr, "INBOUND")
	isDecode := dataFlow == "DECODE_DATA"

	piiCompliantTag, hasCompliant := tags.FindByKey("prose_pii_compliant")
	piiViolationTag, hasViolation := tags.FindByKey("prose_pii_violation")

	var compliantStrings, violatingStrings []string
	if hasCompliant || hasViolation {
		if hasCompliant && piiCompliantTag.VStr != "" {
			compliantStrings = strings.Split(piiCompliantTag.VStr, ",")
		}
		if hasViolation && piiViolationTag.VStr != "" {
			violatingStrings = strings.Split(piiViolationTag.VStr, ",")
		}
	} else {
		piiTypesTag, _ := tags.FindByKey("prose_pii_types")
		opaDecisionTag, _ := tags.FindByKey("prose_opa_decision")
		var allTypes []string
		if piiTypesTag.VStr != "" {
			allTypes = strings.Split(piiTypesTag.VStr, ",")
		}
		if opaDecisionTag.VStr == "deny" {
			violatingStrings = allTypes
		} else {
			compliantStrings = allTypes
		}
	}

	compliantPIIs := piiStringsToEnums(compliantStrings)
	violatingPIIs := piiStringsToEnums(violatingStrings)

	parentSpan, ok := spanStore[span.ParentSpanID()]
	if !ok {
		return fmt.Errorf("could not find parent span %s\n", span.ParentSpanID())
	}

	traceID := fmt.Sprintf("0x%s", span.TraceID)
	spanIDStr := fmt.Sprintf("0x%s", span.SpanID)

	// ── Case 1: INBOUND + DECODE_DATA → callee's Incoming[] ──────────────────
	if isInbound && isDecode {
		calleeFQDN := parentSpan.GetProcess().GetServiceName()
		endpointName := endpointNameFromOperationName(parentSpan.GetOperationName())

		profile := getOrCreateProfile(calleeFQDN, calleeFQDN)
		ep := findOrCreateEndpoint(profile, endpointName)

		ep.EndpointProfile.Incoming = append(ep.EndpointProfile.Incoming,
			IncomingEntry{
				TraceID:                           traceID,
				SpanIDOfIncomingRequestToEndpoint: spanIDStr,
				ObservedPIITypes: ObservedPIITypes{
					CompliantPIIs: compliantPIIs,
					ViolatingPIIs: violatingPIIs,
				},
			},
		)
		fmt.Printf("[PROFILE] %s.%s → incoming: compliant=%v violation=%v\n",
			calleeFQDN, endpointName, compliantStrings, violatingStrings)
		return nil
	}

	// ── Case 2: OUTBOUND + ENCODE_DATA → caller's Outgoing.Indirect[] ────────
	if !isInbound && !isDecode {
		callerFQDN := parentSpan.GetProcess().GetServiceName()
		callerAppSpan, err := findAnAncestorCaller(parentSpan.ParentSpanID(), spanStore)
		if err != nil {
			return fmt.Errorf("could not find caller app span: %v\n", err)
		}
		endpointName := endpointNameFromOperationName(callerAppSpan.GetOperationName())

		calleeHost := ""
		calleePath := ""
		calleeTags := model.KeyValues(parentSpan.GetTags())
		if upstreamCluster, ok := calleeTags.FindByKey("upstream_cluster"); ok {
			parts := strings.Split(upstreamCluster.VStr, "||")
			if len(parts) >= 2 && parts[len(parts)-1] != "" {
				calleeHost = parts[len(parts)-1]
			}
		}
		if calleeHost == "" {
			if peerAddr, ok := calleeTags.FindByKey("peer.address"); ok {
				calleeHost = peerAddr.VStr
			}
		}
		if urlTag, ok := calleeTags.FindByKey("http.url"); ok {
			parts := strings.SplitN(urlTag.VStr, ":9080", 2)
			if len(parts) == 2 {
				calleePath = parts[1]
			}
		}

		profile := getOrCreateProfile(callerFQDN, callerFQDN)
		ep := findOrCreateEndpoint(profile, endpointName)

		ep.EndpointProfile.Outgoing.Indirect = append(ep.EndpointProfile.Outgoing.Indirect,
			IndirectEntry{
				ProcessingInfo: IndirectProcessingInfo{
					TraceID:                             traceID,
					SpanIDOfIncomingRequestToEndpoint:   spanIDStr,
					SpanIDOfOutgoingRequestFromEndpoint: spanIDStr,
					ObservedPIITypes: ObservedPIITypes{
						CompliantPIIs: compliantPIIs,
						ViolatingPIIs: violatingPIIs,
					},
				},
				CalleeHost: calleeHost,
				CalleePath: calleePath,
			},
		)
		fmt.Printf("[PROFILE] %s.%s → indirect via %s: compliant=%v violation=%v\n",
			callerFQDN, endpointName, calleeHost, compliantStrings, violatingStrings)
		return nil
	}

	// ── Case 3: OUTBOUND + DECODE_DATA → caller's Outgoing.Shared[] ──────────
	if !isInbound && isDecode {
		callerFQDN := parentSpan.GetProcess().GetServiceName()
		callerAppSpan, err := findAnAncestorCaller(parentSpan.ParentSpanID(), spanStore)
		if err != nil {
			return fmt.Errorf("could not find caller app span: %v\n", err)
		}
		endpointName := endpointNameFromOperationName(callerAppSpan.GetOperationName())

		externalDomainTag, _ := tags.FindByKey("prose_external_domain")
		externalDomain := externalDomainTag.VStr
		if externalDomain == "" {
			parentTags := model.KeyValues(parentSpan.GetTags())
			if urlTag, ok := parentTags.FindByKey("http.url"); ok {
				externalDomain = urlTag.VStr
			}
			if externalDomain == "" {
				if peerTag, ok := parentTags.FindByKey("peer.address"); ok {
					externalDomain = peerTag.VStr
				}
			}
		}

		profile := getOrCreateProfile(callerFQDN, callerFQDN)
		ep := findOrCreateEndpoint(profile, endpointName)

		ep.EndpointProfile.Outgoing.Shared = append(ep.EndpointProfile.Outgoing.Shared,
			SharedEntry{
				ProcessingInfo: SharedProcessingInfo{
					TraceID:                             traceID,
					SpanIDOfIncomingRequestToEndpoint:   spanIDStr,
					SpanIDOfOutgoingRequestFromEndpoint: spanIDStr,
					ObservedPIITypes: ObservedPIITypes{
						CompliantPIIs: compliantPIIs,
						ViolatingPIIs: violatingPIIs,
					},
				},
				ExternalDomain: externalDomain,
			},
		)
		fmt.Printf("[PROFILE] %s.%s → shared with %s: compliant=%v violation=%v\n",
			callerFQDN, endpointName, externalDomain, compliantStrings, violatingStrings)
		return nil
	}

	return nil
}

// Run `docker compose up -d` to start services before this program
func main() {
	// setup grpc client and query jaeger
	grpcCC, err := grpc.Dial(
		"localhost:16685",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		fmt.Printf("error dialing grpc server:\n%v\n", err)
		return
	}

	defer func() {
		err := grpcCC.Close()
		if err != nil {
			fmt.Printf("error closing grpc client connection:\n%v\n", err)
		}
	}()

	// for possible operations see:
	// https://github.com/jaegertracing/jaeger-idl/blob/main/proto/api_v2/query.proto
	// https://github.com/jaegertracing/jaeger/blob/main/proto-gen/api_v2/query.pb.go
	jaegerQueryClient := api_v2.NewQueryServiceClient(grpcCC)

	services, err := jaegerQueryClient.GetServices(
		context.Background(),
		&api_v2.GetServicesRequest{},
	)

	if err != nil {
		fmt.Printf("error loading services:\n%v\n", err)
		return
	} else {
		fmt.Printf("loaded services:\n%v\n", services.Services)
	}

	findTracesClient, err := jaegerQueryClient.FindTraces(
		context.Background(),
		&api_v2.FindTracesRequest{
			Query: &api_v2.TraceQueryParameters{
				// ServiceName seems to be required
				ServiceName: "golang-filter",
				//OperationName: "SQL SELECT",
				//Tags: map[string]string{
				//	"span.kind": "client",
				//},
				// Treated as num_traces, the number of traces to return in the response
				// SearchDepth: 3,
				StartTimeMin: time.Now().Add(-time.Hour),
				StartTimeMax: time.Now(),
			},
		},
	)
	// TODO: Convert map value to []Spans
	//  and then to map[SpanID]Span
	traceIDToSpansMap := map[model.TraceID]api_v2.SpansResponseChunk{}
	numberOfResponseChunksFound := 0
	numberOfSpansFound := 0
	for {
		spansResponse, err := findTracesClient.Recv()
		if err != nil { // probably got an EOF
			if numberOfResponseChunksFound == 0 {
				fmt.Printf("error finding traces:\n%v\n", err)
			}
			break
		}
		// found a spans response chunk
		numberOfResponseChunksFound += 1

		for _, span := range spansResponse.GetSpans() {
			numberOfSpansFound += 1
			spansResponseChunkForTrace, sameTraceExistsBefore := traceIDToSpansMap[span.TraceID]
			if sameTraceExistsBefore {
				otherSpansInSameTrace := spansResponseChunkForTrace.GetSpans()
				otherSpansInSameTrace = append(otherSpansInSameTrace, span)
				traceIDToSpansMap[span.TraceID] = api_v2.SpansResponseChunk{
					Spans: otherSpansInSameTrace,
				}
			} else {
				traceIDToSpansMap[span.TraceID] = *spansResponse
			}

			fmt.Printf("--> Span Operation name: %s\n", span.OperationName)
			fmt.Printf("Span details: trace id %s, span id: %s, parent span id: %s,\n", span.TraceID, span.SpanID, span.ParentSpanID())
			fmt.Printf("Span start time: %s, duration: %s, references:%s\n", span.GetStartTime(), span.GetDuration(), span.GetReferences())

			printTags(span)
		}
	}
	fmt.Printf("found %d spans across %d traces\n", numberOfSpansFound, len(traceIDToSpansMap))

	// Process each trace independently to avoid span ID collisions across traces.
	// Span IDs are only unique within a single trace — building one global spanStore
	// across all traces causes later spans to overwrite earlier ones with the same ID,
	// breaking parent lookups in populateServiceProfile.
	totalProseSpansProcessed := 0
	totalErrors := 0

	for traceID, chunk := range traceIDToSpansMap {
		// Build a per-trace spanID → Span map
		traceSpanStore := map[model.SpanID]model.Span{}
		for _, s := range chunk.GetSpans() {
			traceSpanStore[s.SpanID] = s
		}

		// Find and process all PROSE spans in this trace
		for _, s := range traceSpanStore {
			// Skip orphan spans — parent ID is all zeros means parent not in this trace
			if s.ParentSpanID().String() == "0000000000000000" {
				continue
			}

			tags := s.GetTags()
			isProseSpan := false
			for _, kv := range tags {
				if strings.HasPrefix(kv.GetKey(), "prose_") {
					isProseSpan = true
					break
				}
			}
			if isProseSpan {
				totalProseSpansProcessed++
				if err := populateServiceProfile(s, traceSpanStore); err != nil {
					totalErrors++
					fmt.Printf("  [trace %s] error: %v", traceID, err)
				}
			}
		}
	}

	fmt.Printf("\nProcessed %d PROSE spans, %d errors\n", totalProseSpansProcessed, totalErrors)
	// Print all populated service profiles as JSON
	fmt.Printf("\n═══ OBSERVED SERVICE PROFILES ═══\n")
	if len(profileStore) == 0 {
		fmt.Printf("(no profiles populated — check errors above)\n")
	}
	for fqdn, profile := range profileStore {
		profileJSON, _ := json.MarshalIndent(profile, "", "  ")
		fmt.Printf("\nService: %s\n%s\n", fqdn, string(profileJSON))
	}

	// ── Composition operator (paper Section 7.2) ──────────────────────────────
	systemWideProfile := proto.SystemwideObservedProfile{}
	for _, svcProfile := range profileStore {
		systemWideProfile = *composer.ComposeWithSvcProfile(&systemWideProfile, svcProfile)
	}

	fmt.Printf("\n═══ SYSTEM-WIDE OBSERVED PROFILE ═══\n")
	systemProfileJSON, _ := json.MarshalIndent(&systemWideProfile, "", "  ")
	fmt.Printf("%s\n", string(systemProfileJSON))
}

func queryHotrod() {
	var err error

	// run some workloads so traces are created
	_, err = http.Get("http://localhost:8080/dispatch?customer=123")
	if err != nil {
		fmt.Printf("error querying hotrod app:\n%v\n", err)
		return
	}

	_, err = http.Get("http://localhost:8080/dispatch?customer=392")
	if err != nil {
		fmt.Printf("error querying hotrod app:\n%v\n", err)
		return
	}

	_, err = http.Get("http://localhost:8080/dispatch?customer=731")
	if err != nil {
		fmt.Printf("error querying hotrod app:\n%v\n", err)
		return
	}

	_, err = http.Get("http://localhost:8080/dispatch?customer=567")
	if err != nil {
		fmt.Printf("error querying hotrod app:\n%v\n", err)
		return
	}
}

func printTags(s model.Span) {
	var tagsForProcess model.KeyValues
	tagsForProcess = s.GetProcess().GetTags()
	tagsForProcess.Sort()
	for _, kv := range tagsForProcess {
		if strings.Contains(kv.GetKey(), "ip") {
			// TODO: This value is parsed as a negative integer. Need to get an ip address instead.
			fmt.Printf("Process tag. %s: (%s) %s\n", kv.GetKey(), kv.VType, kv.VInt64)
		}
	}

	var tags model.KeyValues
	tags = s.GetTags()
	tags.Sort()

	// Identify Prose spans by prose_ prefix — better than operationName check
	isProseSpan := false
	for _, kv := range tags {
		if strings.HasPrefix(kv.GetKey(), "prose_") {
			isProseSpan = true
			break
		}
	}

	if isProseSpan {
		fmt.Printf("Golang filter's span: %s\n", s.OperationName)
		// Print every prose_* tag name and value
		for _, kv := range tags {
			if strings.HasPrefix(kv.GetKey(), "prose_") {
				fmt.Printf("  prose tag: %s = %s\n", kv.GetKey(), kv.VStr)
			}
		}
	}

	// Keep Envoy proxy tag printing unchanged
	for _, kv := range tags {
		if strings.Contains(kv.GetKey(), "component") &&
			strings.Contains(kv.Value().(string), "proxy") {
			fmt.Printf("Envoy proxy\n")
		} else if strings.Contains(kv.GetKey(), "x-request-id") ||
			strings.Contains(kv.GetKey(), "address") ||
			strings.Contains(kv.GetKey(), "url") ||
			strings.Contains(kv.GetKey(), "cluster") {
			fmt.Printf("Envoy proxy with key %s: %s\n", kv.GetKey(), kv.VStr)
		}
	}
	// fmt.Printf("found an attribute with key %s value %s\n", kv.GetKey(), kv.Value.GetStringValue())

}

func parseSpan(spanID model.SpanID, spanStore map[model.SpanID]model.Span) error {
	span, ok := spanStore[spanID]

	if !ok {
		return fmt.Errorf("could not find span with id %s\n", spanID)
	}

	// get whether the golang filter is inbound / outbound and encode / decode
	var tags model.KeyValues
	tags = span.GetTags()

	// Use prose_sidecar_direction and prose_data_flow tags for classification.
	// Fall back to operationName if prose_data_flow is absent (old filter without the tag).
	sidecarDirTag, hasSidecarDir := tags.FindByKey("prose_sidecar_direction")
	if !hasSidecarDir {
		return fmt.Errorf("could not find prose_sidecar_direction tag — not a Prose data span %s\n", spanID)
	}
	var dataFlow string
	dataFlowTag, hasDataFlow := tags.FindByKey("prose_data_flow")
	if hasDataFlow {
		dataFlow = dataFlowTag.VStr
	} else {
		opName := strings.ToLower(span.OperationName)
		switch {
		case strings.Contains(opName, "encodedata"):
			dataFlow = "ENCODE_DATA"
		case strings.Contains(opName, "decodedata"):
			dataFlow = "DECODE_DATA"
		default:
			return nil
		}
	}
	isInbound := strings.Contains(sidecarDirTag.VStr, "INBOUND")
	isDecode := dataFlow == "DECODE_DATA"
	isEncode := dataFlow == "ENCODE_DATA"

	// Only DECODE_DATA and ENCODE_DATA carry PII results — skip header-only spans
	if !isDecode && !isEncode {
		return nil
	}

	// get the parent of the golang-filter span
	parentSpanID := span.ParentSpanID()
	parentSpan, ok := spanStore[parentSpanID]
	if !ok { // this is unusual since a golang-filter service always has a parent span: namely, the Envoy SP
		return fmt.Errorf("could not find parent span with id %s\n", parentSpan)
	}

	// validate that the parent is an envoy SP by checking for the component:proxy span tag
	var parentTags model.KeyValues
	parentTags = parentSpan.GetTags()
	value, ok := parentTags.FindByKey("component")
	parentIsEnvoyProxy := ok && value.VStr == "proxy"
	if !parentIsEnvoyProxy {
		return fmt.Errorf("parent of a Golangfilter span does not have a tag component:proxy and so is probably not an Envoy Proxy")
	}
	parentServiceName := parentSpan.GetProcess().GetServiceName()
	parentOperationName := parentSpan.GetOperationName()
	fmt.Printf("%s %s\n", parentServiceName, parentOperationName)

	// In all cases add PII types in tags in *own* span to the profile of the parent (which can be the caller or the callee)
	if isInbound {
		// parent is the callee SP
		// get the caller by traversing up the call stack
		callerSpanID, err := findAnAncestorCaller(parentSpan.SpanID, spanStore)
		if err != nil {
			return fmt.Errorf("could not find the caller of this span %s\n", parentSpanID)
		}
		callerServiceName := callerSpanID.GetProcess().GetServiceName()
		callerOperationName := callerSpanID.GetOperationName()
		fmt.Printf("%s %s\n", callerServiceName, callerOperationName)

		if isDecode {
			// PII types are sent in a direct request to the parent (callee)
			// Could lead to a direct purpose of use violation
		} else { // Encode case
			// Ignore for now. Handled in outbound encode case
			// PII types are returned in a response by the parent (callee)
		}
	} else { // OUTBOUND sidecar
		// parent is the caller SP
		// get the callee by traversing down the call stack
		// afaict the Span object doesn't store references to own children so can't go down the call stack
		// maybe whenever you find an ancestor caller using the inbound SP, mark it as such?
		calleeSpanID, err := findAnAncestorCaller(parentSpan.SpanID, spanStore)
		if err != nil {
			return fmt.Errorf("could not find the callee of this span %s\n", parentSpanID)
		}
		calleeServiceName := calleeSpanID.GetProcess().GetServiceName()
		calleeOperationName := calleeSpanID.GetOperationName()
		fmt.Printf("%s %s\n", calleeServiceName, calleeOperationName)
		if isDecode {
			// PII types are sent in a request to a third party
			// can cause a data sharing violation
		} else { // Encode case
			// PII types are sent in a response from either a third party or another service
			// we ignore the first case for now
			// in the second case, can cause an indirect purpose of use violation
		}
	}
	return nil
}

func findAnAncestorCaller(spanID model.SpanID, spansStore map[model.SpanID]model.Span) (model.Span, error) {
	// go access the parent using the span.ParentSpanID(). Look it up in the spansStore. If the span's operation name includes the words "router * egress" then go to its parent
	// e.g. for service2's checkstock span
	// the parent would be the service1's "router service2 egress" span, which we know has been inserted by envoy
	// so we skip it and get to its parent, ie service1's checkStock and return it
	parentSpan, ok := spansStore[spanID]
	if ok {
		extraSpanInsertedByEnvoy := strings.Contains(parentSpan.GetOperationName(), "router") && strings.Contains(parentSpan.GetOperationName(), "egress")
		if extraSpanInsertedByEnvoy {
			return findAnAncestorCaller(parentSpan.ParentSpanID(), spansStore)
		}
		return spansStore[spanID], nil
	}
	return model.Span{}, fmt.Errorf("could not find a span with id %s\n", spanID)
}
