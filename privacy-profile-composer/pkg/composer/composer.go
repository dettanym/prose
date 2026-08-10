package composer

import (
	"privacy-profile-composer/pkg/proto"
	"slices"
)

// Adapted from fp-ts
// https://github.com/gcanti/fp-ts/blob/01b8661f2fa594d6f2010573f010d358e6808d13/src/ReadonlyRecord.ts#L1232
func union[V any](
	first map[string]V,
	second map[string]V,
	combine func(firstVal V, secondVal V) V,
) map[string]V {
	// Performance optimizations
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

func combineStringLists(
	strList1 []string,
	strList2 []string,
) []string {
	var strListWithDuplicates []string
	if len(strList1) == 0 {
		strListWithDuplicates = strList2
	} else if len(strList2) == 0 {
		strListWithDuplicates = strList1
	} else {
		strListWithDuplicates = append(strList1, strList2...)
	}
	strListUnique := uniqueNonEmptyElementsOf(strListWithDuplicates)
	return strListUnique
}

func uniqueNonEmptyElementsOf(s []string) []string {
	unique := make(map[string]bool, len(s))
	us := make([]string, len(s))
	for _, elem := range s {
		if len(elem) != 0 && !unique[elem] {
			us = append(us, elem)
			unique[elem] = true
		}
	}
	return us
}

func combinerInnerMost(
	party1 *proto.DataItemAndThirdParties,
	party2 *proto.DataItemAndThirdParties,
) *proto.DataItemAndThirdParties {
	if party1 == nil {
		return party2
	}
	if party2 == nil {
		return party1
	}

	f := func(
		thirdParties1 *proto.ThirdParties,
		thirdParties2 *proto.ThirdParties,
	) *proto.ThirdParties {
		if thirdParties1 == nil {
			return thirdParties2
		}
		if thirdParties2 == nil {
			return thirdParties1
		}
		out := combineStringLists(thirdParties1.ThirdParty, thirdParties2.ThirdParty)
		return &proto.ThirdParties{
			ThirdParty: out,
		}
	}

	var partyOut = proto.DataItemAndThirdParties{
		Entry: union(party1.Entry, party2.Entry, f),
	}

	return &partyOut
}

func combinerMiddle(
	processing1 *proto.PurposeBasedProcessing,
	processing2 *proto.PurposeBasedProcessing,
) *proto.PurposeBasedProcessing {
	if processing1 == nil {
		return processing2
	}
	if processing2 == nil {
		return processing1
	}
	var processingOut = proto.PurposeBasedProcessing{
		ProcessingEntries: union(
			processing1.ProcessingEntries,
			processing2.ProcessingEntries,
			combinerInnerMost,
		),
	}

	return &processingOut
}

func combineSvcInternalFQDNs(systemWideFQDNs []string, svcFQDN string) []string {
	indexOfSvcFQDN := slices.IndexFunc(systemWideFQDNs, func(givenSvcFQDN string) bool {
		return givenSvcFQDN == svcFQDN
	})
	if indexOfSvcFQDN == -1 {
		systemWideFQDNs = append(systemWideFQDNs, svcFQDN)
	}
	return systemWideFQDNs
}

func Composer(
	systemProfile *proto.SystemwideObservedProfile,
	svcProfile *proto.SvcObservedProfile,
) *proto.SystemwideObservedProfile {
	composedProfile := &proto.SystemwideObservedProfile{
		SystemwideProcessingEntries: combinerMiddle(
			systemProfile.SystemwideProcessingEntries,
			svcProfile.ObservedProcessingEntries,
		),
		ComposedServicesInternalFQDNs: combineSvcInternalFQDNs(
			systemProfile.ComposedServicesInternalFQDNs,
			svcProfile.SvcInternalFQDN),
	}

	return composedProfile
}
// SummarisedCall is the result of flattening one EndpointCall's direct +
// indirect buckets into a single pii_compliant / pii_violation pair.
// This is the intermediate form used by Steps 1-3 before the final
// proto.PurposeBasedProcessing is assembled for Step 4.
type SummarisedCall struct {
	PiiCompliant []string // union of direct.pii_compliant + all indirect[*].pii_compliant
	PiiViolation []string // union of direct.pii_violation + all indirect[*].pii_violation
	ThirdParties []string // external domains from shared outgoing entries
}

// Step 1:
// "Summarise an endpoint profile by unioning the pii_compliant and pii_violation
// sets across the direct and indirect buckets."
//
// Input:  one EndpointCall (direct + outgoing[] from main.go's profile builder)
// Output: one SummarisedCall with flat compliant/violation sets
//
// The policy doesn't distinguish between direct and indirect processing —
// it only cares what PII a service touched, not how it arrived.
// So we union direct + indirect into one flat set here.
func summariseEndpointCall(call EndpointCall) SummarisedCall {
	var compliant []string
	var violation []string
	var thirdParties []string
 
	// Direct processing bucket
	compliant = combineStringLists(compliant, call.EndpointProfile.Direct.PiiCompliant)
	violation = combineStringLists(violation, call.EndpointProfile.Direct.PiiViolation)
 
	// Indirect + shared outgoing entries
	for _, outgoing := range call.EndpointProfile.Outgoing {
		switch outgoing.Type {
		case "indirect":
			// Indirect: PII received in a response from another service
			compliant = combineStringLists(compliant, outgoing.PiiCompliant)
			violation = combineStringLists(violation, outgoing.PiiViolation)
		case "shared":
			// Shared: PII sent to an external domain
			// For sharing violations, PiiViolation = what was shared against policy
			violation = combineStringLists(violation, outgoing.PiiViolation)
			if outgoing.ExternalDomain != "" {
				thirdParties = combineStringLists(thirdParties, []string{outgoing.ExternalDomain})
			}
		}
	}
 
	return SummarisedCall{
		PiiCompliant: compliant,
		PiiViolation: violation,
		ThirdParties: thirdParties,
	}
}

// Step 2:
// "Compose two endpoint profiles for the same endpoint by unioning their
// summarised compliant and violation sets."
//
// Input:  map[hash]EndpointCall — all observed calls to one endpoint
// Output: one SummarisedCall representing all observed behaviour at that endpoint
//
// Why: an endpoint may be called many times across many traces with different
// request/response bodies each time. Each call may expose different PII types.
// We union them all so the endpoint's profile reflects everything ever observed.
func composeEndpointCalls(calls map[string]EndpointCall) SummarisedCall {
	result := SummarisedCall{}
	for _, call := range calls {
		summarised := summariseEndpointCall(call)
		result.PiiCompliant = combineStringLists(result.PiiCompliant, summarised.PiiCompliant)
		result.PiiViolation = combineStringLists(result.PiiViolation, summarised.PiiViolation)
		result.ThirdParties = combineStringLists(result.ThirdParties, summarised.ThirdParties)
	}
	return result
}

// Step 3:
// "Compose across all endpoints of the same microservice."
//
// Input:  ServiceObservedProfile (from main.go's profile builder) with
//         map[endpoint]map[hash]EndpointCall
// Output: proto.PurposeBasedProcessing ready for Step 4's Composer()
//
// Why: the privacy policy applies per purpose-of-use, not per endpoint.
// A policy says "the authentication service can process EMAIL_ADDRESS" —
// it doesn't say "only the Login endpoint can, not SetPasswd".
// So we union across all endpoints into one entry keyed by purpose.
//
// The purpose key in the proto map comes from ServiceObservedProfile.PurposeOfUse
// which is set from the service's FQDN (its Kubernetes service name).
func composeSvcProfile(svcProfile ServiceObservedProfile) *proto.PurposeBasedProcessing {
	// Compose across all endpoints of this service into one SummarisedCall
	serviceSummary := SummarisedCall{}
	for _, endpointCalls := range svcProfile.Endpoints {
		// Step 2: compose all calls to this endpoint
		endpointSummary := composeEndpointCalls(endpointCalls)
		// Step 3: union this endpoint's summary into the service summary
		serviceSummary.PiiCompliant = combineStringLists(
			serviceSummary.PiiCompliant,
			endpointSummary.PiiCompliant,
		)
		serviceSummary.PiiViolation = combineStringLists(
			serviceSummary.PiiViolation,
			endpointSummary.PiiViolation,
		)
		serviceSummary.ThirdParties = combineStringLists(
			serviceSummary.ThirdParties,
			endpointSummary.ThirdParties,
		)
	}
 
	// Convert SummarisedCall → proto.PurposeBasedProcessing
	// The outer map key is the purpose of use (e.g. "authentication")
	// The inner map key is the PII type (e.g. "EMAIL_ADDRESS")
	// The value is the list of third parties (empty string = no external domain)
	processingEntries := make(map[string]*proto.DataItemAndThirdParties)
 
	// Build the DataItemAndThirdParties entry for this purpose
	piiEntry := make(map[string]*proto.ThirdParties)
 
	for _, piiType := range serviceSummary.PiiCompliant {
		piiEntry[piiType] = &proto.ThirdParties{
			ThirdParty: []string{""}, // internal only — no external domain
		}
	}
 
	for _, piiType := range serviceSummary.PiiViolation {
		// If it's a violation, record the third parties involved
		// (empty string for purpose-of-use violations, actual domain for sharing)
		existing, exists := piiEntry[piiType]
		if exists {
			// Already in compliant — seen in both compliant and violation contexts
			// Add third parties from violation context
			existing.ThirdParty = combineStringLists(
				existing.ThirdParty,
				serviceSummary.ThirdParties,
			)
		} else {
			piiEntry[piiType] = &proto.ThirdParties{
				ThirdParty: serviceSummary.ThirdParties,
			}
		}
	}
 
	processingEntries[svcProfile.PurposeOfUse] = &proto.DataItemAndThirdParties{
		Entry: piiEntry,
	}
 
	return &proto.PurposeBasedProcessing{
		ProcessingEntries: processingEntries,
	}
}

// Step 4:
// "Compose across all microservices into the system-wide observed profile."
//
// This function now accepts a ServiceObservedProfile (from main.go's profile
// builder) instead of a proto.SvcObservedProfile, runs Steps 1-3 first,
// then does the cross-service union with the existing system-wide profile.
//
// The original Composer() that takes proto.SvcObservedProfile is kept below
// for backward compatibility with grpc_server.go's PostObservedProfile.
func ComposeWithSvcProfile(
	systemProfile *proto.SystemwideObservedProfile,
	svcProfile ServiceObservedProfile,
) *proto.SystemwideObservedProfile {
	// Steps 1-3: summarise and flatten the per-service observed profile
	purposeBasedProcessing := composeSvcProfile(svcProfile)
 
	// Step 4: union with the existing system-wide profile (same as before)
	return &proto.SystemwideObservedProfile{
		SystemwideProcessingEntries: combinerMiddle(
			systemProfile.SystemwideProcessingEntries,
			purposeBasedProcessing,
		),
		ComposedServicesInternalFQDNs: combineSvcInternalFQDNs(
			systemProfile.ComposedServicesInternalFQDNs,
			svcProfile.SvcFQDN,
		),
	}
}
 
// Composer is kept unchanged for grpc_server.go backward compatibility.
// It accepts the flat proto.SvcObservedProfile directly (skips Steps 1-3).
// Use ComposeWithSvcProfile() when calling from main.go's pipeline.
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
 
