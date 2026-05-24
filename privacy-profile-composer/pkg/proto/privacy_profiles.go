//TODO: integrate this file into the rest of the system properly
package privacy_profiles;
import (
        "encoding/json"
        "github.com/google/jsonschema-go/jsonschema"
        "fmt"
)


type SystemwideObservedProfile struct {
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



//TODO: check that this actually makes any sense. Don't want to head blindly in the wrong direction
//FIXME: also, why is names' casing is so inconsistent? can this be fixed, or will a whole bunch of stuff break? 
type PII_type struct {
    PII int
    allowedTypes := map[string]int{
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
    "US_SSN: 15",
    }
}

type PurposeOfUse struct {
    purpose int
    allowedPurposes := map[string]int{
    "advertising": 0,
    "authentication": 1,
    "shipping": 2,
    "payment": 3,
    "marketing": 4,
    }
}

//validation here



//funcs here


service PrivacyProfileComposer {
  // Sends a greeting
  rpc PostObservedProfile (SvcObservedProfile) returns (google.protobuf.Empty) {}
  // Sends another greeting
  rpc GetSystemWideProfile (google.protobuf.Empty) returns (SystemwideObservedProfile) {}
}



func main() {
fmt.Println("test of Go")
human1 := Human{"abc", 4}
human1_enc, err := json.Marshal(human1) 
if err == nil {
fmt.Println(string(human1_enc))
} 
var human2 Human 
json.Unmarshal(human1_enc, &human2)

fmt.Println(human2)
}



type Human struct{
    A string
    B int
}

