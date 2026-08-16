package main;
import (
        "encoding/json"
        "github.com/google/jsonschema-go/jsonschema"
        "fmt"
        "os"
)

type SystemwideObservedProfile struct {
    Purpose PurposeOfUse `json:"purpose"`
    SystemwideProcessingEntries struct {
		ProcessingEntries map[string]struct {
			Entry map[string]struct {
				ThirdParty string `json:"ThirdParty"`
			} `json:"Entry"`
		} `json:"ProcessingEntries"`
	} `json:"systemwideProcessingEntries"`
    ComposedServicesInternalFQDNs []string `json:"ComposedServicesInternalFQDNs"`
}

type SvcObservedProfile struct {
	TargetPolicyHash string         `json:"TargetPolicyHash"`
	ServiceHash      string         `json:"ServiceHash"`
	SvcInternalFQDN  string         `json:"SvcInternalFQDN"`
	PurposeOfUse     PurposeOfUse `json:"PurposeOfUse"`

	ObservedProcessingEntries struct {
		ProcessingEntries map[string]struct {
			Entry map[string]struct {
				ThirdParty string `json:"ThirdParty"`
			} `json:"Entry"`
		} `json:"ProcessingEntries"`
	} `json:"ObservedProcessingEntries"`

	Endpoints struct {
		Endpoint []struct {
			EndpointName  string `json:"EndpointName"`
			EndpointHash  string `json:"EndpointHash"`
			EndpointProfile struct {
				Incoming []struct {
					TraceID  string `json:"TraceID"`
					SpanIDOfIncomingRequestToEndpoint string `json:"SpanIDOfIncomingRequestToEndpoint"`
					ObservedPIITypes struct {
						CompliantPIIs []PIIType `json:"CompliantPIIs"`
						ViolatingPIIs []PIIType `json:"ViolatingPIIs"`
					} `json:"ObservedPIITypes"`
				} `json:"Incoming"`

				Outgoing struct {
					Indirect []struct {
						ProcessingInfo struct {
							TraceID  string `json:"TraceID"`
							SpanIDOfIncomingRequestToEndpoint        string `json:"SpanIDOfIncomingRequestToEndpoint"`
							SpanIDOfOutgoingRequestFromEndpoint     string `json:"SpanIDOfOutgoingRequestFromEndpoint"`
							ObservedPIITypes struct {
								CompliantPIIs []PIIType `json:"CompliantPIIs"`
								ViolatingPIIs []PIIType `json:"ViolatingPIIs"`
							} `json:"ObservedPIITypes"`
						} `json:"ProcessingInfo"`
						CalleePath string `json:"CalleePath"`
						CalleeHost string `json:"CalleeHost"`
					} `json:"Indirect"`

					Shared []struct {
						ProcessingInfo struct {
							TraceID  string `json:"TraceID"`
							SpanIDOfIncomingRequestToEndpoint        string `json:"SpanIDOfIncomingRequestToEndpoint"`
							SpanIDOfOutgoingRequestFromEndpoint     string `json:"SpanIDOfOutgoingRequestFromEndpoint"`
							ObservedPIITypes struct {
								CompliantPIIs []PIIType `json:"CompliantPIIs"`
								ViolatingPIIs []PIIType `json:"ViolatingPIIs"`
							} `json:"ObservedPIITypes"`
						} `json:"ProcessingInfo"`
						ExternalDomain string `json:"ExternalDomain"`
					} `json:"Shared"`
				} `json:"Outgoing"`
			} `json:"EndpointProfile"`
		} `json:"Endpoint"`
	} `json:"Endpoints"`
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

// This function initializes 2 sample profiles as JSONs, validates that both of them match the above schema,
// and checks to ensure that the consistency criteria are met
func main(){
    JSON1, err1 := os.ReadFile("test_json1.json")
    if err1!= nil {
        panic(err1)
    }
    JSON2, err2 := os.ReadFile("test_json2.json")
    if err2!= nil {
        panic(err2)
    }

    validate(JSON1)
    validate(JSON2)
    checkConsistency(JSON1, JSON2)
}