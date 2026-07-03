package main;
import (
        "encoding/json"
        "fmt"
        "slices"
)
// This function is used to check whether there are matching elements in 2 given slices 
// (which, if true, means that there is a consistency violation related to the data flow described by the provided slices)
func checkMatchingElementsInSlices (slice1, slice2 []int) (violating []int, isViolating bool) {
    var violatingElements = []int{}
    var isViolationDetected = false
    for _, element1 := range slice1 {
        if slices.Contains(slice2, element1) {
            isViolationDetected = true
            violatingElements = append (violatingElements, int(element1))
        }
    }
    return violatingElements, isViolationDetected
}

func checkConsistency(json1 []byte, json2 []byte){
    //necessary checks: 
    //1) Each Incoming request must be consistent with other incoming requests to the same endpoint (CC1)
    //2) Each Outgoing.Indirect request must be consistent with other Outgoing.Indirect requests to the same endpoint (CC2)
    //3) Same as 2, but with Outgoing.Shared requests (CC3)
    //4) No single request may have the same PII in both, although this will be caught with the current implementation anyway
    //5) All 4 above must also be consistent between different endpoints of the same service
    //6) The two provided profiles must also be consistent with each other in Incoming, Outgoing.Indirect, and Outgoing.Shared parts.
    violationsDetected := false
    var obj1 SvcObservedProfile
    var obj2 SvcObservedProfile
    json.Unmarshal(json1, &obj1)
    json.Unmarshal(json2, &obj2)

    //getting first profile's compliances/violations
    var profile1IncomingCompliant = []int{}
    var profile1IndirectCompliant = []int{}
    var profile1SharedCompliant = []int{}
    var profile1IncomingViolating = []int{}
    var profile1IndirectViolating = []int{}
    var profile1SharedViolating = []int{}
    
    //scanning each of the service's endpoints in profile 1
    for _, iEndpoint := range obj1.Endpoints.Endpoint {
        //scanning Incoming requests
        for _, jIncoming := range iEndpoint.EndpointProfile.Incoming {
            for _, kCompliant := range jIncoming.ObservedPIITypes.CompliantPIIs {
                if !(slices.Contains(profile1IncomingCompliant, int(kCompliant))) {
                    profile1IncomingCompliant = append (profile1IncomingCompliant, int(kCompliant))
                }
            }
            for _, kViolating := range jIncoming.ObservedPIITypes.ViolatingPIIs {
                if !(slices.Contains(profile1IncomingViolating, int(kViolating))) {
                    profile1IncomingViolating = append (profile1IncomingViolating, int(kViolating))
                }
            }
        }
        //scanning Indirect requests
        for _, jIndirect := range iEndpoint.EndpointProfile.Outgoing.Indirect {
            for _, kCompliant := range jIndirect.ProcessingInfo.ObservedPIITypes.CompliantPIIs {
                if !(slices.Contains(profile1IndirectCompliant, int(kCompliant))) {
                    profile1IndirectCompliant = append (profile1IndirectCompliant, int(kCompliant))
                }
            }
            for _, kViolating := range jIndirect.ProcessingInfo.ObservedPIITypes.ViolatingPIIs {
                if !(slices.Contains(profile1IndirectViolating, int(kViolating))) {
                    profile1IndirectViolating = append (profile1IndirectViolating, int(kViolating))
                }
            }
        }

        //scanning Shared requests
        for _, jShared := range iEndpoint.EndpointProfile.Outgoing.Shared {
            for _, kCompliant := range jShared.ProcessingInfo.ObservedPIITypes.CompliantPIIs {
                if !(slices.Contains(profile1SharedCompliant, int(kCompliant))) {
                    profile1SharedCompliant = append (profile1SharedCompliant, int(kCompliant))
                }
            }
            for _, kViolating := range jShared.ProcessingInfo.ObservedPIITypes.ViolatingPIIs {
                if !(slices.Contains(profile1SharedViolating, int(kViolating))) {
                    profile1SharedViolating = append (profile1SharedViolating, int(kViolating))
                }
            }
        }

    }


    var profile2IncomingCompliant = []int{}
    var profile2IndirectCompliant = []int{}
    var profile2SharedCompliant = []int{}
    var profile2IncomingViolating = []int{}
    var profile2IndirectViolating = []int{}
    var profile2SharedViolating = []int{}
    
    //scanning each of the service's endpoints in profile 2
    for _, iEndpoint := range obj2.Endpoints.Endpoint {
        //scanning Incoming requests
        for _, jIncoming := range iEndpoint.EndpointProfile.Incoming {
            for _, kCompliant := range jIncoming.ObservedPIITypes.CompliantPIIs {
                if !(slices.Contains(profile2IncomingCompliant, int(kCompliant))) {
                    profile2IncomingCompliant = append (profile2IncomingCompliant, int(kCompliant))
                }
            }
            for _, kViolating := range jIncoming.ObservedPIITypes.ViolatingPIIs {
                if !(slices.Contains(profile2IncomingViolating, int(kViolating))) {
                    profile2IncomingViolating = append (profile2IncomingViolating, int(kViolating))
                }
            }
        }
        //scanning Indirect requests
        for _, jIndirect := range iEndpoint.EndpointProfile.Outgoing.Indirect {
            for _, kCompliant := range jIndirect.ProcessingInfo.ObservedPIITypes.CompliantPIIs {
                if !(slices.Contains(profile2IndirectCompliant, int(kCompliant))) {
                    profile2IndirectCompliant = append (profile2IndirectCompliant, int(kCompliant))
                }
            }
            for _, kViolating := range jIndirect.ProcessingInfo.ObservedPIITypes.ViolatingPIIs {
                if !(slices.Contains(profile2IndirectViolating, int(kViolating))) {
                    profile2IndirectViolating = append (profile2IndirectViolating, int(kViolating))
                }
            }
        }
        //scanning Shared requests
        for _, jShared := range iEndpoint.EndpointProfile.Outgoing.Shared {
            for _, kCompliant := range jShared.ProcessingInfo.ObservedPIITypes.CompliantPIIs {
                if !(slices.Contains(profile2SharedCompliant, int(kCompliant))) {
                    profile2SharedCompliant = append (profile2SharedCompliant, int(kCompliant))
                }
            }
            for _, kViolating := range jShared.ProcessingInfo.ObservedPIITypes.ViolatingPIIs {
                if !(slices.Contains(profile2SharedViolating, int(kViolating))) {
                    profile2SharedViolating = append (profile2SharedViolating, int(kViolating))
                }
            }
        }
    }
    var violatingElements = []int{}
    var isViolating = false
    //checking that there are no violations within the same profile's dataflow
    //profile1
    violatingElements, isViolating = checkMatchingElementsInSlices(profile1IncomingCompliant, profile1IncomingViolating)
    if isViolating {
        violationsDetected = true
        fmt.Println("Violation detected at profile1, Incoming dataflow")
        fmt.Println("Inconsistent PIIs: ",violatingElements)
    }
    violatingElements, isViolating = checkMatchingElementsInSlices(profile1IndirectCompliant, profile1IndirectViolating)
    if isViolating {
        violationsDetected = true
        fmt.Println("Violation detected at profile1, Indirect dataflow")
        fmt.Println("Inconsistent PIIs: ",violatingElements)
    }
    violatingElements, isViolating = checkMatchingElementsInSlices(profile1SharedCompliant, profile1SharedViolating)
    if isViolating {
        violationsDetected = true
        fmt.Println("Violation detected at profile1, Shared dataflow")
        fmt.Println("Inconsistent PIIs: ",violatingElements)
    }
    //profile2
    violatingElements, isViolating = checkMatchingElementsInSlices(profile2IncomingCompliant, profile2IncomingViolating)
    if isViolating {
        violationsDetected = true
        fmt.Println("Violation detected at profile2, Incoming dataflow")
        fmt.Println("Inconsistent PIIs: ",violatingElements)
    }
    violatingElements, isViolating = checkMatchingElementsInSlices(profile2IndirectCompliant, profile2IndirectViolating)
    if isViolating {
        violationsDetected = true
        fmt.Println("Violation detected at profile2, Indirect dataflow")
        fmt.Println("Inconsistent PIIs: ",violatingElements)
    }
    violatingElements, isViolating = checkMatchingElementsInSlices(profile2SharedCompliant, profile2SharedViolating)
    if isViolating {
        violationsDetected = true
        fmt.Println("Violation detected at profile2, Shared dataflow")
        fmt.Println("Inconsistent PIIs: ",violatingElements)
    }

    //Checking that the two profiles do not have violations between each other
    //Here we need to compare compliant1 and violating2, as well as vice versa (compliant2 and violating1)
    violatingElements, isViolating = checkMatchingElementsInSlices(profile1IncomingCompliant, profile2IncomingViolating)
    if isViolating {
        violationsDetected = true
        fmt.Println("Violation detected between profile1 compliant and profile2 violating, Incoming dataflow")
        fmt.Println("Inconsistent PIIs: ",violatingElements)
    }
    violatingElements, isViolating = checkMatchingElementsInSlices(profile2IncomingCompliant, profile1IncomingViolating)
    if isViolating {
        violationsDetected = true
        fmt.Println("Violation detected between profile1 violating and profile2 compliant, Incoming dataflow")
        fmt.Println("Inconsistent PIIs: ",violatingElements)
    }
    violatingElements, isViolating = checkMatchingElementsInSlices(profile1IndirectCompliant, profile2IndirectViolating)
    if isViolating {
        violationsDetected = true
        fmt.Println("Violation detected between profile1 compliant and profile2 violating, Indirect dataflow")
        fmt.Println("Inconsistent PIIs: ",violatingElements)
    }
    violatingElements, isViolating = checkMatchingElementsInSlices(profile2IndirectCompliant, profile1IndirectViolating)
    if isViolating {
        violationsDetected = true
        fmt.Println("Violation detected between profile1 violating and profile2 compliant, Indirect dataflow")
        fmt.Println("Inconsistent PIIs: ",violatingElements)
    }
    violatingElements, isViolating = checkMatchingElementsInSlices(profile1SharedCompliant, profile1SharedViolating)
    if isViolating {
        violationsDetected = true
        fmt.Println("Violation detected between profile1 compliant and profile2 violating, Shared dataflow")
        fmt.Println("Inconsistent PIIs: ",violatingElements)
    }
    violatingElements, isViolating = checkMatchingElementsInSlices(profile2SharedCompliant, profile2SharedViolating)
    if isViolating {
        violationsDetected = true
        fmt.Println("Violation detected between profile1 violating and profile2 compliant, Shared dataflow")
        fmt.Println("Inconsistent PIIs: ",violatingElements)
    }

    if !violationsDetected {
        fmt.Println("No consistency violations detected!")
    }
}
