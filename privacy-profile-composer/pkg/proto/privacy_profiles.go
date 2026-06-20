package main;
import (
        "encoding/json"
        "github.com/google/jsonschema-go/jsonschema"
        "fmt"
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
    Incoming incomingRequest `json:"Incoming"`
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
    ObservedPIIsClassified []PIIType `json:"ObservedPIIsClassified"`
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
    fmt.Println("decoded JSON: \n\n", v)
    validation := sch.Validate(v)
    fmt.Println("\n\nValidation results:\n\n",validation)
}

func checkConsistency(json1 []byte, json2 []byte){
    var obj1 SvcObservedProfile
    var obj2 SvcObservedProfile
    json.Unmarshal(json1, &obj1)
    json.Unmarshal(json2, &obj2)

}

// This function initializes 2 sample profiles as JSONs, validates that both of them match the above schema,
// and checks to ensure that the consistency criteria are met
func main(){
    JSON1 := []byte(`{"TargetPolicyHash": "%POLICY_FILE_HASH%", "ServiceHash": "%SERVICE_IMAGE_HASH%", "SvcInternalFQDN":"/svc/endpoint/login", "PurposeOfUse": 1, "ObservedProcessingEntries":{"ProcessingEntries":{"analytics":{"Entry":{"advertising":{"ThirdParty":"adware.xyz"}}}}},"Endpoints": {"Endpoint":[{"EndpointName": "Login", "EndpointHash":"OBJECT_HASH", "EndpointProfile":{"Incoming":{"TraceID":"0x00000000074ace1d", "SpanIDOfIncomingRequestToEndpoint": "0x000000000059aebc", "ObservedPIITypes": {"ObservedPIIsClassified": [14,10,3,13,12]}},"Outgoing":{"Indirect":[{"ProcessingInfo": {"TraceID": "0x00000000081bca3f","SpanIDOfIncomingRequestToEndpoint":"0x000000000059aebe","SpanIDOfOutgoingRequestFromEndpoint": "0x001234567059aebc","ObservedPIITypes": {"ObservedPIIsClassified": [3,5,10]}},"CalleePath": "/GetUserByUsername/","CalleeHost":"users.default.svc.cluster.local"}],"Shared":[{"ProcessingInfo": {"TraceID": "0x00000000085cad64","SpanIDOfIncomingRequestToEndpoint":"0x000000000059aebc","SpanIDOfOutgoingRequestFromEndpoint": "0x0012345670593c7a","ObservedPIITypes": {"ObservedPIIsClassified": [5,1,9,7,2]}},"ExternalDomain": "twilio2notascam.xyz"}]}}}, {"EndpointName": "SetPasswd", "EndpointHash":"OBJECT_HASH2", "EndpointProfile":{"Incoming":{"TraceID":"0x000000000747921d", "SpanIDOfIncomingRequestToEndpoint": "0x000000000057d209", "ObservedPIITypes": {"ObservedPIIsClassified": [14]}},"Outgoing":{"Indirect":[{"ProcessingInfo": {"TraceID":"","SpanIDOfIncomingRequestToEndpoint":"","SpanIDOfOutgoingRequestFromEndpoint": "","ObservedPIITypes": {"ObservedPIIsClassified": []}},"CalleePath":"","CalleeHost":""}],"Shared":[{"ProcessingInfo": {"TraceID": "","SpanIDOfIncomingRequestToEndpoint":"","SpanIDOfOutgoingRequestFromEndpoint": "","ObservedPIITypes": {"ObservedPIIsClassified": []}},"ExternalDomain": ""}]}}}]}}`)
    JSON2 := []byte(`{"TargetPolicyHash": "%POLICY_FILE_HASH%", "ServiceHash": "%SERVICE_IMAGE_HASH%", "SvcInternalFQDN":"/svc/endpoint/login", "PurposeOfUse": 1, "ObservedProcessingEntries":{"ProcessingEntries":{"analytics":{"Entry":{"advertising":{"ThirdParty":"adware.xyz"}}}}},"Endpoints": {"Endpoint":[{"EndpointName": "Login", "EndpointHash":"OBJECT_HASH", "EndpointProfile":{"Incoming":{"TraceID":"0x00000000074ace1d", "SpanIDOfIncomingRequestToEndpoint": "0x000000000059aebc", "ObservedPIITypes": {"ObservedPIIsClassified": [14,10,3,13,12]}},"Outgoing":{"Indirect":[{"ProcessingInfo": {"TraceID": "0x00000000081bca3f","SpanIDOfIncomingRequestToEndpoint":"0x000000000059aebe","SpanIDOfOutgoingRequestFromEndpoint": "0x001234567059aebc","ObservedPIITypes": {"ObservedPIIsClassified": [3,5,10]}},"CalleePath": "/GetUserByUsername/","CalleeHost":"users.default.svc.cluster.local"}],"Shared":[{"ProcessingInfo": {"TraceID": "0x00000000085cad64","SpanIDOfIncomingRequestToEndpoint":"0x000000000059aebc","SpanIDOfOutgoingRequestFromEndpoint": "0x0012345670593c7a","ObservedPIITypes": {"ObservedPIIsClassified": [5,1,9,7,2]}},"ExternalDomain": "twilio2notascam.xyz"}]}}}, {"EndpointName": "SetPasswd", "EndpointHash":"OBJECT_HASH2", "EndpointProfile":{"Incoming":{"TraceID":"0x000000000747921d", "SpanIDOfIncomingRequestToEndpoint": "0x000000000057d209", "ObservedPIITypes": {"ObservedPIIsClassified": [14]}},"Outgoing":{"Indirect":[{"ProcessingInfo": {"TraceID":"","SpanIDOfIncomingRequestToEndpoint":"","SpanIDOfOutgoingRequestFromEndpoint": "","ObservedPIITypes": {"ObservedPIIsClassified": []}},"CalleePath":"","CalleeHost":""}],"Shared":[{"ProcessingInfo": {"TraceID": "","SpanIDOfIncomingRequestToEndpoint":"","SpanIDOfOutgoingRequestFromEndpoint": "","ObservedPIITypes": {"ObservedPIIsClassified": []}},"ExternalDomain": ""}]}}}]}}`)
    //infer(true)
    validate(JSON1)
    validate(JSON2)
   // checkConsistency(JSON1, JSON2)
}

