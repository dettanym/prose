package proto;
import (
        "encoding/json"
        "github.com/google/jsonschema-go/jsonschema"
        "fmt"
)


type SystemwideObservedProfile struct {
    traceID string
    spanID string
    purpose PurposeOfUse
    systemwideProcessingEntries purposeBasedProcessing
    ComposedServicesInternalFQDNs []string
}

type SvcObservedProfile struct {
    svcInternalFQDN string
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
    endpointName []string
}

type endpoint struct {
    OBJECT_HASH_NAME string
    OBJECT_HASH objectHash
}

type objectHash struct{
    traceID string
    spanID_of_call string
    endpoint_profile endpointProfile
}

type endpointProfile struct {
    direct directMessage
    outgoing outgoingMessage
}


type directMessage struct {
    pii_compliant pii_compliant
    pii_violation pii_violation
}

type outgoingMessage struct {
    indirect []indirectMessage
    shared []sharedMessage
}

type indirectMessage struct {
    spanID string
    callee_path string
    callee_host string
    pii_compliant pii_compliant
    pii_violation pii_violation
}

type sharedMessage struct {
    spanID string
    pii_compliant pii_compliant
    pii_violation pii_violation
    external_domain string
}

type pii_violation struct {
    pii []PII_type
//  violation_reason string  //not used yet
}

type pii_compliant struct {
    pii []PII_type
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

type PII_type struct {
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



//funcs here

