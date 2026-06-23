package main;
import (
        "encoding/json"
        "github.com/google/jsonschema-go/jsonschema"
        "fmt"
        "slices"
)


type SystemwideObservedProfile struct {
    Purpose PurposeOfUse `json:"purpose"`
    SystemwideProcessingEntries purposeBasedProcessing `json:"systemwideProcessingEntries"`
    ComposedServicesInternalFQDNs []string `json:"ComposedServicesInternalFQDNs"`
}

type SvcObservedProfile struct {
    TargetPolicyHash string `json:"TargetPolicyHash"`
    ServiceHash string `json:"ServiceHash"`
    SvcInternalFQDN string `json:"SvcInternalFQDN"`
    PurposeOfUse PurposeOfUse `json:"PurposeOfUse"`
    ObservedProcessingEntries purposeBasedProcessing `json:"ObservedProcessingEntries"`
    Endpoints endpoints `json:"Endpoints"`
}

type purposeBasedProcessing struct {
    ProcessingEntries map[string]dataItemAndThirdParties `json:"ProcessingEntries"`
}

type dataItemAndThirdParties struct {
    Entry map[string]ThirdParties `json:"Entry"`
}

type ThirdParties struct {
    ThirdParty string `json:"ThirdParty"`
}

type endpoints struct {
    Endpoint []endpoint `json:"Endpoint"`
}

type endpoint struct {
    EndpointName string `json:"EndpointName"`
    EndpointHash string `json:"EndpointHash"`
    EndpointProfile endpointProfile `json:"EndpointProfile"`
}

type endpointProfile struct {
    Incoming []incomingRequest `json:"Incoming"`
    Outgoing outgoingRequests `json:"Outgoing"`
}


type incomingRequest struct {
    TraceID string `json:"TraceID"`
    SpanIDOfIncomingRequestToEndpoint string `json:"SpanIDOfIncomingRequestToEndpoint"`
    ObservedPIITypes observedPIITypes `json:"ObservedPIITypes"`
}

type outgoingRequests struct {
    Indirect []OutgoingRequestToInternalEndpoint  `json:"Indirect"`
    Shared []OutgoingRequestToExternalEndpoint  `json:"Shared"`
}

type OutgoingRequestToExternalEndpoint struct {
    ProcessingInfo processingInfo  `json:"ProcessingInfo"`
    ExternalDomain string  `json:"ExternalDomain"`
}
 
type OutgoingRequestToInternalEndpoint struct {
    ProcessingInfo processingInfo `json:"ProcessingInfo"`
    CalleePath string `json:"CalleePath"`
    CalleeHost string  `json:"CalleeHost"`
}

type processingInfo struct {
    TraceID string `json:"TraceID"`
    SpanIDOfIncomingRequestToEndpoint string `json:"SpanIDOfIncomingRequestToEndpoint"`
    SpanIDOfOutgoingRequestFromEndpoint string `json:"SpanIDOfOutgoingRequestFromEndpoint"`
    ObservedPIITypes observedPIITypes `json:"ObservedPIITypes"`
}

type observedPIITypes struct { 
    CompliantPIIs []PIIType `json:CompliantPIIs`
    ViolatingPIIs []PIIType `json:ViolatingPIIs`
}

type PIIType int

const (
    CREDIT_CARD PIIType = iota
    NRP
    US_ITIN
    PERSON
    US_BANK_NUMBER
    US_PASSPORT
    IP_ADDRESS
    US_DRIVER_LICENSE
    CRYPTO
    URL
    PHONE_NUMBER
    IBAN_CODE
    DATE_TIME
    LOCATION
    EMAIL_ADDRESS
    US_SSN
)

var types = map[PIIType]string{
    CREDIT_CARD:         "CREDIT_CARD",
    NRP:                 "NRP",
    US_ITIN:             "US_ITIN",
    PERSON:              "PERSON",
    US_BANK_NUMBER:      "US_BANK_NUMBER",
    US_PASSPORT:         "US_PASSPORT",
    IP_ADDRESS:          "IP_ADDRESS",
    US_DRIVER_LICENSE:   "US_DRIVER_LICENSE",
    CRYPTO:              "CRYPTO",
    URL:                 "URL",
    PHONE_NUMBER:        "PHONE_NUMBER",
    IBAN_CODE:           "IBAN_CODE",
    DATE_TIME:           "DATE_TIME",
    LOCATION:            "LOCATION",
    EMAIL_ADDRESS:       "EMAIL_ADDRESS",
    US_SSN:              "US_SSN",
}

type PurposeOfUse int

const (
    advertising PurposeOfUse = iota
    authentication
    shipping
    payment
    marketing
)

var purposes = map[PurposeOfUse]string{
    advertising:    "advertising",
    authentication: "authentication",
    shipping:       "shipping",
    payment:        "payment",
    marketing:      "marketing",
}


// A function to JSON schema based on structs above
// Parameter printNicely determines whether to print inferred schema in one line or in more human-readable format
func infer(printNicely bool) *jsonschema.Schema{
    schema, _ := jsonschema.For[SvcObservedProfile](nil)
    var schemaPrint []byte
    if printNicely {
        schemaPrint, _ = json.MarshalIndent(schema,"","    ")
}   else {
    schemaPrint, _ = json.Marshal(schema)
}
    fmt.Println("Inferred schema: \n\n", string(schemaPrint))
    return schema
}

//unneeded, since paper already provides a json, but may be useful in the future
//func encode(validatedJson string) []byte {        
//    var encoding, _ = json.Marshal (validatedJson)
//    fmt.Println("encoded JSON: \n\n", string(encoding))
//    return encoding
//}

// A function to validate provided validatedJson against a JSON schema provided by infer()
func validate(validatedJson []byte){
    sch, _ := (infer(false)).Resolve(nil)
    var v interface{}
    json.Unmarshal(validatedJson, &v)
    fmt.Println("\ndecoded JSON: \n\n", v)
    validation := sch.Validate(v)
    fmt.Println("\n\nValidation results:\n\n",validation, "\n\n")
}


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

// This function initializes 2 sample profiles as JSONs, validates that both of them match the above schema,
// and checks to ensure that the consistency criteria are met
func main(){
    JSON1 := []byte(`{"TargetPolicyHash": "%POLICY_FILE_HASH%", "ServiceHash": "%SERVICE_IMAGE_HASH%", "SvcInternalFQDN":"/svc/endpoint/login", "PurposeOfUse": 1, "ObservedProcessingEntries":{"ProcessingEntries":{"analytics":{"Entry":{"advertising":{"ThirdParty":"adware.xyz"}}}}},"Endpoints": {"Endpoint":[{"EndpointName": "Login", "EndpointHash":"OBJECT_HASH", "EndpointProfile":{"Incoming":[{"TraceID":"0x00000000074ace1d", "SpanIDOfIncomingRequestToEndpoint": "0x000000000059aebc", "ObservedPIITypes": {"CompliantPIIs": [1,2], "ViolatingPIIs": [14,10,3,13,12]}}],"Outgoing":{"Indirect":[{"ProcessingInfo": {"TraceID": "0x00000000081bca3f","SpanIDOfIncomingRequestToEndpoint":"0x000000000059aebe","SpanIDOfOutgoingRequestFromEndpoint": "0x001234567059aebc","ObservedPIITypes": {"CompliantPIIs": [3,5,10], "ViolatingPIIs": [3,5,10]}},"CalleePath": "/GetUserByUsername/","CalleeHost":"users.default.svc.cluster.local"}],"Shared":[{"ProcessingInfo": {"TraceID": "0x00000000085cad64","SpanIDOfIncomingRequestToEndpoint":"0x000000000059aebc","SpanIDOfOutgoingRequestFromEndpoint": "0x0012345670593c7a","ObservedPIITypes": {"CompliantPIIs": [1,2,5,7,9], "ViolatingPIIs": [1,2,5,7,9]}},"ExternalDomain": "twilio2notascam.xyz"}]}}}, {"EndpointName": "SetPasswd", "EndpointHash":"OBJECT_HASH2", "EndpointProfile":{"Incoming":[{"TraceID":"0x000000000747921d", "SpanIDOfIncomingRequestToEndpoint": "0x000000000057d209", "ObservedPIITypes": {"CompliantPIIs": [3], "ViolatingPIIs": []}}],"Outgoing":{"Indirect":[{"ProcessingInfo": {"TraceID":"","SpanIDOfIncomingRequestToEndpoint":"","SpanIDOfOutgoingRequestFromEndpoint": "","ObservedPIITypes": {"CompliantPIIs": [], "ViolatingPIIs": []}},"CalleePath":"","CalleeHost":""}],"Shared":[{"ProcessingInfo": {"TraceID": "","SpanIDOfIncomingRequestToEndpoint":"","SpanIDOfOutgoingRequestFromEndpoint": "","ObservedPIITypes": {"CompliantPIIs": [], "ViolatingPIIs": []}},"ExternalDomain": ""}]}}}]}}`)
    JSON2 := []byte(`{"TargetPolicyHash": "%POLICY_FILE_HASH%", "ServiceHash": "%SERVICE_IMAGE_HASH%", "SvcInternalFQDN":"/svc/endpoint/login", "PurposeOfUse": 1, "ObservedProcessingEntries":{"ProcessingEntries":{"analytics":{"Entry":{"advertising":{"ThirdParty":"adware.xyz"}}}}},"Endpoints": {"Endpoint":[{"EndpointName": "Login", "EndpointHash":"OBJECT_HASH", "EndpointProfile":{"Incoming":[{"TraceID":"0x00000000074ace1d", "SpanIDOfIncomingRequestToEndpoint": "0x000000000059aebc", "ObservedPIITypes": {"CompliantPIIs": [1,2,4,5], "ViolatingPIIs": []}}],"Outgoing":{"Indirect":[{"ProcessingInfo": {"TraceID": "0x00000000081bca3f","SpanIDOfIncomingRequestToEndpoint":"0x000000000059aebe","SpanIDOfOutgoingRequestFromEndpoint": "0x001234567059aebc","ObservedPIITypes": {"CompliantPIIs": [3,5,10], "ViolatingPIIs": [3,5,10]}},"CalleePath": "/GetUserByUsername/","CalleeHost":"users.default.svc.cluster.local"}],"Shared":[{"ProcessingInfo": {"TraceID": "0x00000000085cad64","SpanIDOfIncomingRequestToEndpoint":"0x000000000059aebc","SpanIDOfOutgoingRequestFromEndpoint": "0x0012345670593c7a","ObservedPIITypes": {"CompliantPIIs": [1,2,5,7,9], "ViolatingPIIs": [1,2,5,7,9]}},"ExternalDomain": "twilio2notascam.xyz"}]}}}, {"EndpointName": "SetPasswd", "EndpointHash":"OBJECT_HASH2", "EndpointProfile":{"Incoming":[{"TraceID":"0x000000000747921d", "SpanIDOfIncomingRequestToEndpoint": "0x000000000057d209", "ObservedPIITypes": {"CompliantPIIs": [3], "ViolatingPIIs": []}}],"Outgoing":{"Indirect":[{"ProcessingInfo": {"TraceID":"","SpanIDOfIncomingRequestToEndpoint":"","SpanIDOfOutgoingRequestFromEndpoint": "","ObservedPIITypes": {"CompliantPIIs": [], "ViolatingPIIs": []}},"CalleePath":"","CalleeHost":""}],"Shared":[{"ProcessingInfo": {"TraceID": "","SpanIDOfIncomingRequestToEndpoint":"","SpanIDOfOutgoingRequestFromEndpoint": "","ObservedPIITypes": {"CompliantPIIs": [], "ViolatingPIIs": []}},"ExternalDomain": ""}]}}}]}}`)
    //infer(true)
    validate(JSON1)
    validate(JSON2)
    checkConsistency(JSON1, JSON2)
}

// TODO: put jsons in a test file