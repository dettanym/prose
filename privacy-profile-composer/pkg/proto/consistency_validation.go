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
func checkCompliance(complianceType []PIIType, complianceSlice *[]int) {
    for _, compliance := range complianceType {
        if !(slices.Contains(*complianceSlice, int(compliance))) {
            *complianceSlice = append (*complianceSlice, int(compliance))
        }
}}

func checkEndpoints(profile SvcObservedProfile, incomingCompliant *[]int, indirectCompliant *[]int, sharedCompliant *[]int, incomingViolating *[]int, indirectViolating *[]int, sharedViolating *[]int) {
    for _, endpoint := range profile.Endpoints.Endpoint {
        //scanning Incoming, Indirect, and Shared requests
        for _, eachType := range endpoint.EndpointProfile.Incoming {
                checkCompliance(eachType.ObservedPIITypes.CompliantPIIs, incomingCompliant)
                checkCompliance(eachType.ObservedPIITypes.ViolatingPIIs, incomingViolating)
        }
        for _, eachType := range endpoint.EndpointProfile.Outgoing.Indirect {
                checkCompliance(eachType.ProcessingInfo.ObservedPIITypes.CompliantPIIs, indirectCompliant)
                checkCompliance(eachType.ProcessingInfo.ObservedPIITypes.ViolatingPIIs, indirectViolating)
        }
        for _, eachType := range endpoint.EndpointProfile.Outgoing.Shared {
                checkCompliance(eachType.ProcessingInfo.ObservedPIITypes.CompliantPIIs, sharedCompliant)
                checkCompliance(eachType.ProcessingInfo.ObservedPIITypes.ViolatingPIIs, sharedViolating)
        }
    }
}

func checkConsistency(json1 []byte, json2 []byte) {
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
    checkEndpoints(obj1, &profile1IncomingCompliant,&profile1IndirectCompliant,&profile1SharedCompliant,&profile1IncomingViolating,&profile1IndirectViolating,&profile1SharedViolating)
    

    var profile2IncomingCompliant = []int{}
    var profile2IndirectCompliant = []int{}
    var profile2SharedCompliant = []int{}
    var profile2IncomingViolating = []int{}
    var profile2IndirectViolating = []int{}
    var profile2SharedViolating = []int{}
    
    //scanning each of the service's endpoints in profile 2
    checkEndpoints(obj2, &profile2IncomingCompliant,&profile2IndirectCompliant,&profile2SharedCompliant,&profile2IncomingViolating,&profile2IndirectViolating,&profile2SharedViolating)

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
