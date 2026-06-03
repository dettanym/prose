package proto;
import (
        "encoding/json"
        "github.com/google/jsonschema-go/jsonschema"
        "fmt"
)


type SystemwideObservedProfile struct {
    purpose PurposeOfUse
    systemwideProcessingEntries purposeBasedProcessing
    ComposedServicesInternalFQDNs []string
}

type SvcObservedProfile struct {
    targetPolicyHash string
    serviceHash string
    svcInternalFQDN string
    purposeOfUse PurposeOfUse
    observedProcessingEntries purposeBasedProcessing
}

type purposeBasedProcessing struct {
    processingEntries map[string]dataItemAndThirdParties
}

type dataItemAndThirdParties struct {
    entry map[string]thirdParties
}

type thirdParties struct {
    thirdParty string
}

type endpoints struct {
    endpointName []endpoint
}

type endpoint struct {
    endpointHash string
    endpointProfile endpointProfile
}

type endpointProfile struct {
    incoming incomingRequest
    outgoing outgoingRequests
}


type incomingRequest struct {
    traceID string
    spanIDOfIncomingRequestToEndpoint string
    observedPIITypes observedPIITypes
}

type outgoingRequests struct {
    indirect []OutgoingRequestToInternalEndpoint 
    shared []OutgoingRequestToExternalEndpoint 
}

type OutgoingRequestToExternalEndpoint struct {
       processingInfo processingInfo 
       externalDomain string 
}
 
type OutgoingRequestToInternalEndpoint struct {
      processingInfo processingInfo
      calleePath string
      calleeHost string 
}

type processingInfo struct {
    traceID string
    spanIDOfIncomingRequestToEndpoint string
    spanIDOfOutgoingRequestFromEndpoint string
    observedPIITypes observedPIITypes
}

type observedPIITypes struct { 
    observedPIIsClassified map[PIIType]boolean
}

var types = map[string]int{
    "CREDIT_CARD": 0,
    "NRP": 1,
    "US_ITIN": 2,
    "PERSON": 3,
    "US_BANK_NUMBER": 4,
    "US_PASSPORT": 5,
    "IP_ADDRESS": 6,
    "US_DRIVER_LICENSE": 7,
    "CRYPTO": 8,
    "URL": 9,
    "PHONE_NUMBER": 10,
    "IBAN_CODE": 11,
    "DATE_TIME": 12,
    "LOCATION": 13,
    "EMAIL_ADDRESS": 14,
    "US_SSN": 15,
}

type PIIType struct {
    PII int
} 

var purposes = map[string]int{
    "advertising": 0,
    "authentication": 1,
    "shipping": 2,
    "payment": 3,
    "marketing": 4,
}

type PurposeOfUse struct {
    purpose int
}

//validation here
func validate(){

}


//funcs here
func encode(structure struct){
json.Marshal ()
}
