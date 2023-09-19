package composer

import "privacy-profile-composer/pkg/proto"

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

func Composer(
	systemProfile proto.SystemwideObservedProfile,
	svcProfile proto.SvcObservedProfile,
) proto.SystemwideObservedProfile {
	composedProfile := proto.SystemwideObservedProfile{
		SystemwideProcessingEntries: combinerMiddle(
			systemProfile.SystemwideProcessingEntries,
			svcProfile.ObservedProcessingEntries,
		),
		ComposedServicesInternalFQDNs: systemProfile.ComposedServicesInternalFQDNs,
	}

	return composedProfile
}
