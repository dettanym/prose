package composer

import (
	"privacy-profile-composer/pkg/proto"
	"slices"
)

// ─── Input types for the composition pipeline ─────────────────────────────────
// These types are defined here in the composer package so main.go can import
// and use them directly. They mirror the anonymous struct types in
// privacy_profiles.go but are named and importable.

type ObservedPIITypes struct {
	CompliantPIIs []proto.PIIType
	ViolatingPIIs []proto.PIIType
}

type IncomingEntry struct {
	TraceID                           string
	SpanIDOfIncomingRequestToEndpoint string
	ObservedPIITypes                  ObservedPIITypes
}

type IndirectProcessingInfo struct {
	TraceID                             string
	SpanIDOfIncomingRequestToEndpoint   string
	SpanIDOfOutgoingRequestFromEndpoint string
	ObservedPIITypes                    ObservedPIITypes
}

type IndirectEntry struct {
	ProcessingInfo IndirectProcessingInfo
	CalleeHost     string
	CalleePath     string
}

type SharedProcessingInfo struct {
	TraceID                             string
	SpanIDOfIncomingRequestToEndpoint   string
	SpanIDOfOutgoingRequestFromEndpoint string
	ObservedPIITypes                    ObservedPIITypes
}

type SharedEntry struct {
	ProcessingInfo SharedProcessingInfo
	ExternalDomain string
}

type OutgoingEntries struct {
	Indirect []IndirectEntry
	Shared   []SharedEntry
}

type EndpointProfileData struct {
	Incoming []IncomingEntry
	Outgoing OutgoingEntries
}

type EndpointEntry struct {
	EndpointName    string
	EndpointHash    string
	EndpointProfile EndpointProfileData
}

type EndpointsList struct {
	Endpoint []EndpointEntry
}

// SvcObservedProfileLocal is the rich in-memory profile type used by main.go.
// It mirrors the SvcObservedProfile struct from privacy_profiles.go.
type SvcObservedProfileLocal struct {
	TargetPolicyHash string
	ServiceHash      string
	SvcInternalFQDN  string
	Endpoints        EndpointsList
}

// ─── Existing primitives (unchanged) ─────────────────────────────────────────

func union[V any](first, second map[string]V, combine func(V, V) V) map[string]V {
	if len(first) == 0 {
		return second
	}
	if len(second) == 0 {
		return first
	}
	out := make(map[string]V)
	for k, v1 := range first {
		if v2, ok := second[k]; ok {
			out[k] = combine(v1, v2)
		} else {
			out[k] = v1
		}
	}
	for k, v2 := range second {
		if _, ok := out[k]; !ok {
			out[k] = v2
		}
	}
	return out
}

func combineStringLists(strList1 []string, strList2 []string) []string {
	return uniqueNonEmptyElementsOf(append(strList1, strList2...))
}

func uniqueNonEmptyElementsOf(s []string) []string {
	unique := make(map[string]bool, len(s))
	us := make([]string, 0, len(s))
	for _, elem := range s {
		if len(elem) != 0 && !unique[elem] {
			us = append(us, elem)
			unique[elem] = true
		}
	}
	return us
}

func combinerInnerMost(party1, party2 *proto.DataItemAndThirdParties) *proto.DataItemAndThirdParties {
	if party1 == nil {
		return party2
	}
	if party2 == nil {
		return party1
	}
	f := func(t1, t2 *proto.ThirdParties) *proto.ThirdParties {
		if t1 == nil {
			return t2
		}
		if t2 == nil {
			return t1
		}
		return &proto.ThirdParties{
			ThirdParty: combineStringLists(t1.ThirdParty, t2.ThirdParty),
		}
	}
	return &proto.DataItemAndThirdParties{Entry: union(party1.Entry, party2.Entry, f)}
}

func combinerMiddle(p1, p2 *proto.PurposeBasedProcessing) *proto.PurposeBasedProcessing {
	if p1 == nil {
		return p2
	}
	if p2 == nil {
		return p1
	}
	return &proto.PurposeBasedProcessing{
		ProcessingEntries: union(p1.ProcessingEntries, p2.ProcessingEntries, combinerInnerMost),
	}
}

func combineSvcInternalFQDNs(fqdns []string, fqdn string) []string {
	idx := slices.IndexFunc(fqdns, func(s string) bool { return s == fqdn })
	if idx == -1 {
		fqdns = append(fqdns, fqdn)
	}
	return fqdns
}

// ─── PIIType helpers ──────────────────────────────────────────────────────────

func uniquePIITypes(s []proto.PIIType) []proto.PIIType {
	seen := make(map[proto.PIIType]bool)
	var out []proto.PIIType
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func unionPIITypes(a, b []proto.PIIType) []proto.PIIType {
	return uniquePIITypes(append(a, b...))
}

// ─── Intermediate composition type ───────────────────────────────────────────

type SummarisedCall struct {
	PiiCompliant []proto.PIIType
	PiiViolation []proto.PIIType
	ThirdParties []string
}

// ─── Step 1: summariseEndpointProfile ────────────────────────────────────────
// Union Incoming + Outgoing.Indirect + Outgoing.Shared into one flat set.
func summariseEndpointProfile(ep EndpointEntry) SummarisedCall {
	result := SummarisedCall{}
	for _, inc := range ep.EndpointProfile.Incoming {
		result.PiiCompliant = unionPIITypes(result.PiiCompliant, inc.ObservedPIITypes.CompliantPIIs)
		result.PiiViolation = unionPIITypes(result.PiiViolation, inc.ObservedPIITypes.ViolatingPIIs)
	}
	for _, ind := range ep.EndpointProfile.Outgoing.Indirect {
		result.PiiCompliant = unionPIITypes(result.PiiCompliant, ind.ProcessingInfo.ObservedPIITypes.CompliantPIIs)
		result.PiiViolation = unionPIITypes(result.PiiViolation, ind.ProcessingInfo.ObservedPIITypes.ViolatingPIIs)
	}
	for _, sh := range ep.EndpointProfile.Outgoing.Shared {
		result.PiiViolation = unionPIITypes(result.PiiViolation, sh.ProcessingInfo.ObservedPIITypes.ViolatingPIIs)
		if sh.ExternalDomain != "" {
			result.ThirdParties = combineStringLists(result.ThirdParties, []string{sh.ExternalDomain})
		}
	}
	return result
}

// ─── Step 2: composeEndpoints ─────────────────────────────────────────────────
// Union across all observed entries for the same endpoint.
func composeEndpoints(endpoints []EndpointEntry) SummarisedCall {
	result := SummarisedCall{}
	for _, ep := range endpoints {
		s := summariseEndpointProfile(ep)
		result.PiiCompliant = unionPIITypes(result.PiiCompliant, s.PiiCompliant)
		result.PiiViolation = unionPIITypes(result.PiiViolation, s.PiiViolation)
		result.ThirdParties = combineStringLists(result.ThirdParties, s.ThirdParties)
	}
	return result
}

// ─── Step 3: composeSvcProfile ────────────────────────────────────────────────
// Union across all endpoints of the same service → proto.PurposeBasedProcessing.
func composeSvcProfile(svcProfile *SvcObservedProfileLocal) *proto.PurposeBasedProcessing {
	// Group by endpoint name then compose each group (Step 2)
	byName := make(map[string][]EndpointEntry)
	for _, ep := range svcProfile.Endpoints.Endpoint {
		byName[ep.EndpointName] = append(byName[ep.EndpointName], ep)
	}

	serviceSummary := SummarisedCall{}
	for _, eps := range byName {
		s := composeEndpoints(eps)
		serviceSummary.PiiCompliant = unionPIITypes(serviceSummary.PiiCompliant, s.PiiCompliant)
		serviceSummary.PiiViolation = unionPIITypes(serviceSummary.PiiViolation, s.PiiViolation)
		serviceSummary.ThirdParties = combineStringLists(serviceSummary.ThirdParties, s.ThirdParties)
	}

	// Convert to proto.PurposeBasedProcessing
	purposeKey := svcProfile.SvcInternalFQDN
	piiEntry := make(map[string]*proto.ThirdParties)

	for _, t := range serviceSummary.PiiCompliant {
		key := proto.PIIType_name[int32(t)]
		piiEntry[key] = &proto.ThirdParties{ThirdParty: []string{""}}
	}
	for _, t := range serviceSummary.PiiViolation {
		key := proto.PIIType_name[int32(t)]
		if _, exists := piiEntry[key]; !exists {
			domains := serviceSummary.ThirdParties
			if len(domains) == 0 {
				domains = []string{""}
			}
			piiEntry[key] = &proto.ThirdParties{ThirdParty: domains}
		}
	}

	return &proto.PurposeBasedProcessing{
		ProcessingEntries: map[string]*proto.DataItemAndThirdParties{
			purposeKey: {Entry: piiEntry},
		},
	}
}

// ─── Step 4: ComposeWithSvcProfile ────────────────────────────────────────────
// Runs Steps 1-3 then unions into system-wide profile.
func ComposeWithSvcProfile(
	systemProfile *proto.SystemwideObservedProfile,
	svcProfile *SvcObservedProfileLocal,
) *proto.SystemwideObservedProfile {
	purposeBasedProcessing := composeSvcProfile(svcProfile)
	return &proto.SystemwideObservedProfile{
		SystemwideProcessingEntries: combinerMiddle(
			systemProfile.SystemwideProcessingEntries,
			purposeBasedProcessing,
		),
		ComposedServicesInternalFQDNs: combineSvcInternalFQDNs(
			systemProfile.ComposedServicesInternalFQDNs,
			svcProfile.SvcInternalFQDN,
		),
	}
}

// Composer is kept for grpc_server.go backward compatibility.
func Composer(
	systemProfile *proto.SystemwideObservedProfile,
	svcProfile *proto.SvcObservedProfile,
) *proto.SystemwideObservedProfile {
	return &proto.SystemwideObservedProfile{
		SystemwideProcessingEntries: combinerMiddle(
			systemProfile.SystemwideProcessingEntries,
			svcProfile.ObservedProcessingEntries,
		),
		ComposedServicesInternalFQDNs: combineSvcInternalFQDNs(
			systemProfile.ComposedServicesInternalFQDNs,
			svcProfile.SvcInternalFQDN,
		),
	}
}
