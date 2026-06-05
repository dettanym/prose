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
    SvcInternalFQDN string `json:"SvcInternalFQDN"` //TODO: still need?
    PurposeOfUse PurposeOfUse `json:"PurposeOfUse"`
    ObservedProcessingEntries purposeBasedProcessing `json:"ObservedProcessingEntries"` // TODO: still need?
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
    EndpointName []endpoint `json:EndpointName`
}

type endpoint struct {
    EndpointHash string `json:EndpointHash`
    EndpointProfile endpointProfile `json:EndpointProfile`
}

type endpointProfile struct {
    Incoming incomingRequest `json:Incoming`
    Outgoing outgoingRequests `json:Outgoing`
}


type incomingRequest struct {
    TraceID string `json:TraceID`
    SpanIDOfIncomingRequestToEndpoint string `json:SpanIDOfIncomingRequestToEndpoint`
    ObservedPIITypes observedPIITypes `json:ObservedPIITypes`
}

type outgoingRequests struct {
    Indirect []OutgoingRequestToInternalEndpoint  `json:Indirect`
    Shared []OutgoingRequestToExternalEndpoint  `json:Shared`
}

type OutgoingRequestToExternalEndpoint struct {
       ProcessingInfo processingInfo  `json:ProcessingInfo`
       ExternalDomain string  `json:ExternalDomain`
}
 
type OutgoingRequestToInternalEndpoint struct {
      ProcessingInfo processingInfo `json:ProcessingInfo`
      CalleePath string `json:CalleePath`
      CalleeHost string  `json:CalleeHost`
}

type processingInfo struct {
    TraceID string `json:TraceID`
    SpanIDOfIncomingRequestToEndpoint string `json:SpanIDOfIncomingRequestToEndpoint`
    SpanIDOfOutgoingRequestFromEndpoint string `json:SpanIDOfOutgoingRequestFromEndpoint`
    ObservedPIITypes observedPIITypes `json:ObservedPIITypes`
}

type observedPIITypes struct { 
    ObservedPIIsClassified []PIIType `json:ObservedPIIsClassified`
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



//validation here
func infer(printNicely bool) {
    schema, _ := jsonschema.For[SvcObservedProfile](nil)
    var schemaPrint []byte
    if printNicely {
        schemaPrint, _ = json.MarshalIndent(schema,"","    ")
}   else {
    schemaPrint, _ = json.Marshal(schema)
}
    fmt.Println(string(schemaPrint))
}

func encode(){
    var encoding, _ = json.MarshalIndent (`{"target_policy_hash": "%POLICY_FILE_HASH%", "service_hash": "%SERVICE_IMAGE_HASH%", "purpose_of_use": "authentication", "endpoints": { "Login": { "%OBJECT_HASH%": { "traceID": "0x00000000074ace1d", "spanID_of_call": "0x000000000059aebc", "endpoint_profile": {"direct": {"pii_compliant": ["EMAIL_ADDRESS"]}, "outgoing": [{ "type": "indirect", "spanID": "0x00000000004789bb", "callee_host": "users.default.svc.cluster.local", "callee_path": "/GetUserByUsername/", "pii_compliant": ["EMAIL_ADDRESS", "PHONE_NUMBER"], "pii_violation": ["PERSON", "LOCATION", "DATE", "GENDER"], "violation_reason": "Detected PII type PERSON, LOCATION, DATE, GENDER cannot be processed for purpose of use AUTHENTICATION" },{ "type": "shared","spanID": "0x0000000000621def","pii_violation": ["PHONE_NUMBER"],"external_domain": "twilio.com" }]}}},"SetPasswd": {"%OBJECT_HASH%": {"traceID": "0x00000000831bbcce","spanID_of_call": "0x0000000093aaddee","endpoint_profile": {"direct": {"pii_compliant": ["EMAIL_ADDRESS"]}}}}}}`,"","    ")
    //var validation, _ = (*jsonschema.Resolved).Validate(string(encoding))
    fmt.Println(string(encoding))
}

func validate(){

}


//funcs here
func main(){
    infer(false)
    encode()
//var encoding, _ = json.Marshal (`{"target_policy_hash": "%POLICY_FILE_HASH%", "service_hash": "%SERVICE_IMAGE_HASH%", "purpose_of_use": "authentication", "endpoints": { "Login": { "%OBJECT_HASH%": { "traceID": "0x00000000074ace1d", "spanID_of_call": "0x000000000059aebc", "endpoint_profile": {"direct": {"pii_compliant": ["EMAIL_ADDRESS"]}, "outgoing": [{ "type": "indirect", "spanID": "0x00000000004789bb", "callee_host": "users.default.svc.cluster.local", "callee_path": "/GetUserByUsername/", "pii_compliant": ["EMAIL_ADDRESS", "PHONE_NUMBER"], "pii_violation": ["PERSON", "LOCATION", "DATE", "GENDER"], "violation_reason": "Detected PII type PERSON, LOCATION, DATE, GENDER cannot be processed for purpose of use AUTHENTICATION" },{ "type": "shared","spanID": "0x0000000000621def","pii_violation": ["PHONE_NUMBER"],"external_domain": "twilio.com" }]}}},"SetPasswd": {"%OBJECT_HASH%": {"traceID": "0x00000000831bbcce","spanID_of_call": "0x0000000093aaddee","endpoint_profile": {"direct": {"pii_compliant": ["EMAIL_ADDRESS"]}}}}}}`)
//var validation, _ = (*jsonschema.Resolved).Validate(string(encoding))
//fmt.Println(validation)

}
