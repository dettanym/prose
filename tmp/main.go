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
)

// ─── Observed profile data structures (matching Listing 3 in paper) ──────────

// DirectProcessing holds PII types seen directly in a request to this endpoint.
// Maps to "direct" key in Listing 3 line 10.
type DirectProcessing struct {
	PiiCompliant []string `json:"pii_compliant,omitempty"`
	PiiViolation []string `json:"pii_violation,omitempty"`
}

// OutgoingEntry represents one outgoing call from an endpoint — either an
// indirect purpose-of-use (call to another service) or a data-sharing entry
// (call to an external domain). Maps to Listing 3 lines 13-29.
type OutgoingEntry struct {
	Type             string   `json:"type"`                        // "indirect" or "shared"
	SpanID           string   `json:"spanID"`                      // evidence
	CalleeHost       string   `json:"callee_host,omitempty"`       // for indirect
	CalleePath       string   `json:"callee_path,omitempty"`       // for indirect
	PiiCompliant     []string `json:"pii_compliant,omitempty"`     // for indirect
	PiiViolation     []string `json:"pii_violation,omitempty"`     // both types
	ViolationReason  string   `json:"violation_reason,omitempty"`  // both types
	ExternalDomain   string   `json:"external_domain,omitempty"`   // for shared
}

// EndpointProfile holds the full privacy profile for one call to an endpoint.
// Maps to Listing 3 lines 9-29 (the "endpoint_profile" object).
type EndpointProfile struct {
	Direct   DirectProcessing `json:"direct,omitempty"`
	Outgoing []OutgoingEntry  `json:"outgoing,omitempty"`
}

// EndpointCall holds one observed call to an endpoint, indexed by a hash of the
// profile (to deduplicate identical calls). Maps to Listing 3 lines 6-29.
type EndpointCall struct {
	TraceID         string          `json:"traceID"`
	SpanIDOfCall    string          `json:"spanID_of_call"`
	EndpointProfile EndpointProfile `json:"endpoint_profile"`
}

// ServiceObservedProfile is the top-level observed profile for one service.
// Maps to Listing 3 lines 1-36.
// Endpoints maps endpoint name → (profile hash → EndpointCall).
type ServiceObservedProfile struct {
	TargetPolicyHash string                               `json:"target_policy_hash"`
	ServiceHash      string                               `json:"service_hash"`
	PurposeOfUse     string                               `json:"purpose_of_use"`
	Endpoints        map[string]map[string]EndpointCall   `json:"endpoints"`
}

// profileStore accumulates profiles keyed by service FQDN during a pipeline run.
// In a real system this would be persisted-- here it lives in memory per run.
var profileStore = map[string]*ServiceObservedProfile{}

// getOrCreateProfile returns the existing profile for a service or creates a new one.
func getOrCreateProfile(serviceFQDN, purposeOfUse string) *ServiceObservedProfile {
	if p, ok := profileStore[serviceFQDN]; ok {
		return p
	}
	p := &ServiceObservedProfile{
		TargetPolicyHash: "%POLICY_FILE_HASH%",  // TODO: read from OPA bundle hash
		ServiceHash:      "%SERVICE_IMAGE_HASH%", // TODO: read from x-envoy-peer-metadata
		PurposeOfUse:     purposeOfUse,
		Endpoints:        map[string]map[string]EndpointCall{},
	}
	profileStore[serviceFQDN] = p
	return p
}

// endpointNameFromOperationName extracts the endpoint name from an Envoy SP
// operationName. Istio formats these as "<service> <endpoint>" for inbound spans
// (e.g. "authentication Login") or just the endpoint name for app spans.
// We take the last whitespace-separated token.
func endpointNameFromOperationName(operationName string) string {
	parts := strings.Fields(operationName)
	if len(parts) == 0 {
		return operationName
	}
	return parts[len(parts)-1]
}

// profileHashKey produces a simple deduplication key for an EndpointProfile.
// The paper uses a cryptographic hash; here we use a string concatenation of
// the PII types for simplicity. Replace with sha256 in production.
func profileHashKey(profile EndpointProfile) string {
	key := strings.Join(profile.Direct.PiiCompliant, ",") +
		"|" + strings.Join(profile.Direct.PiiViolation, ",")
	for _, o := range profile.Outgoing {
		key += "|" + o.Type + ":" + strings.Join(o.PiiViolation, ",")
	}
	if key == "|" {
		key = "empty"
	}
	return key
}

// addToProfile inserts or updates an EndpointCall in the profile for a given
// service and endpoint name. If an identical profile already exists (same hash),
// only the traceID/spanID evidence is updated to the most recent call.
func addToProfile(profile *ServiceObservedProfile, endpointName string, call EndpointCall) {
	if _, ok := profile.Endpoints[endpointName]; !ok {
		profile.Endpoints[endpointName] = map[string]EndpointCall{}
	}
	hashKey := profileHashKey(call.EndpointProfile)
	profile.Endpoints[endpointName][hashKey] = call
}

// populateServiceProfile is the core function that maps one parsed PROSE span
// into the correct bucket of the correct service's observed profile.
//
// The three cases mirror paper Section 6.1 / Figure 4:
//   INBOUND  + DECODE_DATA → callee's direct processing bucket
//   OUTBOUND + ENCODE_DATA → caller's indirect processing bucket
//   OUTBOUND + DECODE_DATA → caller's data sharing bucket
func populateServiceProfile(
	span model.Span,
	spanStore map[model.SpanID]model.Span,
) error {
	tags := model.KeyValues(span.GetTags())

	// Read the direction tag — must be present
	directionTag, hasDir := tags.FindByKey("prose_sidecar_direction")
	if !hasDir {
		return nil // not a prose span
	}

	// Read the data flow tag — may be absent if running the old filter
	// without the PROSE_DATA_FLOW tag. Fall back to inferring from operation name.
	dataFlowTag, hasFlow := tags.FindByKey("prose_data_flow")
	var dataFlow string
	if hasFlow {
		dataFlow = dataFlowTag.VStr
	} else {
		// Infer from operation name — old filter uses "encodedata" / "decodedata"
		opName := strings.ToLower(span.OperationName)
		switch {
		case strings.Contains(opName, "encodedata") || opName == "encodedata":
			dataFlow = "ENCODE_DATA"
		case strings.Contains(opName, "decodedata") || opName == "decodedata":
			dataFlow = "DECODE_DATA"
		default:
			return nil // not a data span
		}
	}

	if dataFlow != "DECODE_DATA" && dataFlow != "ENCODE_DATA" {
		return nil // header-only spans carry no PII results
	}

	// Also require prose_pii_types or prose_opa_decision to confirm
	// this span actually ran processBody (not just a header span)
	_, hasPii := tags.FindByKey("prose_pii_types")
	_, hasOpa := tags.FindByKey("prose_opa_decision")
	if !hasPii && !hasOpa {
		return nil // span didn't run processBody, nothing to profile
	}

	isInbound := strings.Contains(directionTag.VStr, "INBOUND")
	isDecode := dataFlow == "DECODE_DATA"

	// Read PII result tags
	piiTypesTag, _ := tags.FindByKey("prose_pii_types")
	opaDecisionTag, _ := tags.FindByKey("prose_opa_decision")
	violationTypeTag, _ := tags.FindByKey("prose_violation_type")

	var piiTypes []string
	if piiTypesTag.VStr != "" {
		piiTypes = strings.Split(piiTypesTag.VStr, ",")
	}
	isViolation := opaDecisionTag.VStr == "deny"

	// Split PII types into compliant and violation lists
	var piiCompliant, piiViolation []string
	if isViolation {
		piiViolation = piiTypes
	} else {
		piiCompliant = piiTypes
	}

	// Navigate to the parent Envoy SP span
	parentSpanID := span.ParentSpanID()
	parentSpan, ok := spanStore[parentSpanID]
	if !ok {
		return fmt.Errorf("could not find parent span %s\n", parentSpanID)
	}

	traceID := fmt.Sprintf("0x%s", span.TraceID)
	spanID := fmt.Sprintf("0x%s", span.SpanID)

	// ── Case 1: INBOUND + DECODE_DATA ─────────────────────────────────────────
	// The PROSE span's parent is the callee's inbound Envoy SP.
	// The parent's operationName is "<service> <endpoint>" e.g. "authentication Login".
	// This gives us both the service name and the endpoint name.
	// Profile update: callee's direct processing bucket.
	if isInbound && isDecode {
		calleeFQDN := parentSpan.GetProcess().GetServiceName()
		endpointName := endpointNameFromOperationName(parentSpan.GetOperationName())

		// purposeOfUse comes from the service name (same as the pod label)
		// In production this would come from x-envoy-peer-metadata
		purposeOfUse := calleeFQDN

		profile := getOrCreateProfile(calleeFQDN, purposeOfUse)

		call := EndpointCall{
			TraceID:      traceID,
			SpanIDOfCall: spanID,
			EndpointProfile: EndpointProfile{
				Direct: DirectProcessing{
					PiiCompliant: piiCompliant,
					PiiViolation: piiViolation,
				},
			},
		}
		addToProfile(profile, endpointName, call)

		fmt.Printf("[PROFILE] %s.%s → direct: compliant=%v violation=%v\n",
			calleeFQDN, endpointName, piiCompliant, piiViolation)
		return nil
	}

	// ── Case 2: OUTBOUND + ENCODE_DATA ────────────────────────────────────────
	// The PROSE span's parent is the caller's outbound Envoy SP.
	// We need the caller's app span (grandparent, skipping the Envoy SP).
	// The caller's app span operationName is the endpoint name e.g. "Login".
	// Profile update: caller's indirect processing bucket (outgoing entry).
	if !isInbound && !isDecode {
		// Service name: from parent (outbound Envoy SP) process name = caller service
		callerFQDN := parentSpan.GetProcess().GetServiceName()

		// Walk up past the outbound Envoy SP to the caller's app span
		// to get the endpoint name (the operation the caller was executing)
		callerAppSpan, err := findAnAncestorCaller(parentSpan.ParentSpanID(), spanStore)
		if err != nil {
			return fmt.Errorf("could not find caller app span: %v\n", err)
		}
		// Endpoint name: last path segment of the caller app span's operation name
		// e.g. "/productpage" from "productpage.svc.cluster.local:9080/productpage"
		// or just use the HTTP path from the url tag on the parent Envoy SP
		endpointName := endpointNameFromOperationName(callerAppSpan.GetOperationName())
		purposeOfUse := callerFQDN

		// Callee host: upstream_cluster tag on the parent Envoy SP is the most
		// reliable source — Istio sets it to the full FQDN in format:
		// "outbound|9080||reviews.bookinfo-with-prose-filter.svc.cluster.local"
		// Extract the FQDN from the last segment after "||"
		calleeHost := ""
		calleePath := ""
		calleeTags := model.KeyValues(parentSpan.GetTags())
		if upstreamCluster, ok := calleeTags.FindByKey("upstream_cluster"); ok {
			parts := strings.Split(upstreamCluster.VStr, "||")
			if len(parts) >= 2 && parts[len(parts)-1] != "" {
				calleeHost = parts[len(parts)-1] // e.g. "reviews.bookinfo-with-prose-filter.svc.cluster.local"
			}
		}
		// fallback: peer.address (IP only, less useful)
		if calleeHost == "" {
			if peerAddr, ok := calleeTags.FindByKey("peer.address"); ok {
				calleeHost = peerAddr.VStr
			}
		}
		// Also extract the HTTP path as callee_path from the url tag
		if urlTag, ok := calleeTags.FindByKey("http.url"); ok {
			// e.g. "http://reviews:9080/reviews/0" → "/reviews/0"
			parts := strings.SplitN(urlTag.VStr, ":9080", 2)
			if len(parts) == 2 {
				calleePath = parts[1]
			}
		}
		_ = calleePath

		profile := getOrCreateProfile(callerFQDN, purposeOfUse)

		outgoingEntry := OutgoingEntry{
			Type:            "indirect",
			SpanID:          spanID,
			CalleeHost:      calleeHost,
			CalleePath:      calleePath,
			PiiCompliant:    piiCompliant,
			PiiViolation:    piiViolation,
			ViolationReason: violationTypeTag.VStr,
		}

		// Find or create the EndpointCall for this endpoint and append outgoing entry
		if _, ok := profile.Endpoints[endpointName]; !ok {
			profile.Endpoints[endpointName] = map[string]EndpointCall{}
		}
		// Deduplication: use pii types + callee as the hash key.
		// Spans with identical PII behaviour collapse into one entry;
		// only the evidence (traceID/spanID) is updated to the most recent call.
		hashKey := profileHashKey(EndpointProfile{Outgoing: []OutgoingEntry{outgoingEntry}}) + calleeHost
		existingCall, exists := profile.Endpoints[endpointName][hashKey]
		if exists {
			// Same behaviour seen before — update evidence to most recent span
			existingCall.TraceID = traceID
			existingCall.SpanIDOfCall = spanID
			profile.Endpoints[endpointName][hashKey] = existingCall
		} else {
			profile.Endpoints[endpointName][hashKey] = EndpointCall{
				TraceID:      traceID,
				SpanIDOfCall: spanID,
				EndpointProfile: EndpointProfile{
					Outgoing: []OutgoingEntry{outgoingEntry},
				},
			}
		}

		fmt.Printf("[PROFILE] %s.%s → indirect via %s: compliant=%v violation=%v\n",
			callerFQDN, endpointName, calleeHost, piiCompliant, piiViolation)
		return nil
	}

	// ── Case 3: OUTBOUND + DECODE_DATA ────────────────────────────────────────
	// The PROSE span's parent is the caller's outbound Envoy SP.
	// Caller is sending PII to a third-party external domain.
	// Profile update: caller's data sharing (shared) outgoing entry.
	if !isInbound && isDecode {
		callerFQDN := parentSpan.GetProcess().GetServiceName()

		callerAppSpan, err := findAnAncestorCaller(parentSpan.ParentSpanID(), spanStore)
		if err != nil {
			return fmt.Errorf("could not find caller app span: %v\n", err)
		}
		endpointName := endpointNameFromOperationName(callerAppSpan.GetOperationName())
		purposeOfUse := callerFQDN

		externalDomainTag, _ := tags.FindByKey("prose_external_domain")
		externalDomain := externalDomainTag.VStr
		// Only fall back to Envoy tags if PROSE_EXTERNAL_DOMAIN is empty
		// (old filter without the tag)
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

		profile := getOrCreateProfile(callerFQDN, purposeOfUse)

		outgoingEntry := OutgoingEntry{
			Type:            "shared",
			SpanID:          spanID,
			ExternalDomain:  externalDomain,
			PiiViolation:    piiViolation,
			PiiCompliant:    piiCompliant,
			ViolationReason: violationTypeTag.VStr,
		}

		if _, ok := profile.Endpoints[endpointName]; !ok {
			profile.Endpoints[endpointName] = map[string]EndpointCall{}
		}
		hashKey := "shared:" + externalDomain + ":" + strings.Join(piiViolation, ",")
		profile.Endpoints[endpointName][hashKey] = EndpointCall{
			TraceID:      traceID,
			SpanIDOfCall: spanID,
			EndpointProfile: EndpointProfile{
				Outgoing: []OutgoingEntry{outgoingEntry},
			},
		}

		fmt.Printf("[PROFILE] %s.%s → shared with %s: compliant=%v violation=%v\n",
			callerFQDN, endpointName, externalDomain, piiCompliant, piiViolation)
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
