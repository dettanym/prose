package envoyfilter

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"privacy-profile-composer/pkg/envoyfilter/internal/common"
	pb "privacy-profile-composer/pkg/proto"
)

func NewInboundFilter(callbacks api.FilterCallbackHandler, config *config) api.StreamFilter {
	return &inboundFilter{callbacks: callbacks, config: config}
}

type inboundFilter struct {
	api.PassThroughStreamFilter

	callbacks api.FilterCallbackHandler
	config    *config

	path                string
	method              string
	contentType         string
	contentLength       string
	host                string
	istioHeader         common.XEnvoyPeerMetadataHeader
	canAnalyzePIIOnBody bool
	piiTypes            string
}

func (f *inboundFilter) sendLocalReplyInternal() api.StatusType {
	body := fmt.Sprintf("%s, path: %s\r\n", f.config.echoBody, f.path)
	f.callbacks.SendLocalReply(200, body, nil, 0, "")
	return api.LocalReply
}

// Callbacks which are called in request path
func (f *inboundFilter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.StatusType {
	log.Println(">>> DECODE HEADERS")

	f.path = header.Path() //Get(":path")
	f.method = header.Method()
	f.host = header.Host()

	contentType, exists := header.Get("content-type")
	if exists {
		f.contentType = contentType
	}

	contentLength, exists := header.Get("content-length")
	if exists {
		f.contentLength = contentLength
	}

	xEnvoyPeerMetadata, exists := header.Get("x-envoy-peer-metadata")
	if exists {
		parsedHeader, err := common.DecodeXEnvoyPeerMetadataHeader(xEnvoyPeerMetadata)
		if err != nil {
			log.Printf("Error decoding x-envoy-peer-metadata header: %s", err)
			return api.Continue
		}
		f.istioHeader = parsedHeader
	}

	var purpose, svcName string
	if &f.istioHeader != nil {
		svcName = f.istioHeader.Name
		labels := f.istioHeader.Labels
		var purposeExists bool
		purpose, purposeExists = labels["purpose"]
		if purposeExists == false {
			// The pod hasn't been labelled with a purpose
			// Initialize the purpose to the svcName
			// Infer it from the svcName using presidio
			purpose = svcName
		}
	} else {
		purpose = "ANY"
		svcName = "UNKNOWN SVC"
	}
	// TODO: Insert it into OpenTelemetry baggage for tracing?
	header.Add("x-prose-purpose", purpose) // For OPA

	log.Printf("%v (%v) %v://%v%v\n", header.Method(), header.Protocol(), header.Scheme(), header.Host(), header.Path())
	header.Range(func(key, value string) bool {
		log.Printf("  \"%v\": %v\n", key, value)
		return true
	})

	return api.Continue
}

func sendComposedProfile(fqdn string, purpose string, piiTypes []string, thirdParties []string) api.StatusType {
	var (
		composerSvcAddr = flag.String("addr", "http://prose-server.prose-system.svc.cluster.local:50051", "the address to connect to")
	)

	flag.Parse()
	// Set up a connection to the server.
	conn, err := grpc.Dial(*composerSvcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("can not connect to Composer SVC at addr %v. ERROR: %v", composerSvcAddr, err)
		return api.Continue
	}
	defer func(conn *grpc.ClientConn) {
		err = conn.Close()
		if err != nil {
			log.Printf("could not close connection to Composer server %s", err)
		}
	}(conn)
	c := pb.NewPrivacyProfileComposerClient(conn)

	// Contact the server and print out its response.
	ctx := context.Background()

	processingEntries := make(map[string]*pb.DataItemAndThirdParties, len(piiTypes))
	for _, pii := range piiTypes {
		dataItemThirdParties := map[string]*pb.ThirdParties{
			pii: {
				ThirdParty: thirdParties,
			},
		}
		processingEntries[purpose] = &pb.DataItemAndThirdParties{Entry: dataItemThirdParties}
	}
	_, err = c.PostObservedProfile(
		ctx,
		&pb.SvcObservedProfile{
			SvcInternalFQDN: fqdn,
			ObservedProcessingEntries: &pb.PurposeBasedProcessing{
				ProcessingEntries: processingEntries},
		},
	)

	if err != nil {
		log.Printf("got this error when posting observed profile: %v", err)
	}
	return api.Continue
}

func piiAnalysis(svcName string, bufferBytes []byte) (string, error) {
	var jsonBody = `{
			"key_F": {
				"key_a1": "My phone number is 212-121-1424"
			},
			"URL": "www.abc.com",
			"key_c": 3,
			"names": ["James Bond", "Clark Kent", "Hakeem Olajuwon", "No name here!"],
			"address": "123 Alpha Beta, Waterloo ON N2L3G1, Canada",
			"DOB": "01-01-1989",
			"gender": "Female",
			"race": "Asian",
			"language": "English"
		}`

	svcNameBuf, err := json.Marshal(svcName)
	if err != nil {
		return "", fmt.Errorf("could not marshal service name string into a valid JSON string: %w", err)
	}

	// TODO replace jsonBody with bufferBytes input arg
	msgString := `{"json_to_analyze":` + jsonBody + `,"derive_purpose":` + string(svcNameBuf) + `}`

	resp, err := http.Post("http://presidio.prose-system.svc.cluster.local:3000/batchanalyze", "application/json", bytes.NewBufferString(msgString))
	if err != nil {
		return "", fmt.Errorf("presidio post error: %w", err)
	}

	log.Printf("presidio responded '%v', content-length is %v bytes\n", resp.Status, resp.ContentLength)

	jsonResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not read Presidio response, %w", err)
	}

	err = resp.Body.Close()
	if err != nil {
		return "", fmt.Errorf("could not close presidio response body, %w", err)
	}

	log.Println("presidio response headers:")
	for key, value := range resp.Header {
		log.Printf("  \"%v\": %v\n", key, value)
	}
	log.Println("presidio response body:")
	log.Printf("%v\n", string(jsonResp))

	return string(jsonResp), nil
}

func (f *inboundFilter) DecodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	log.Println(">>> DECODE DATA")
	log.Println("  <<About to forward", buffer.Len(), "bytes of data to service>>")

	var jsonBody []byte
	if f.contentType == "application/x-www-form-urlencoded" {
		query, err := url.ParseQuery(buffer.String())
		if err != nil {
			log.Printf("Failed to start decoding JSON data")
			return api.Continue
		}
		log.Println("  <<decoded x-www-form-urlencoded data: ", query)
		jsonBody, err = json.Marshal(query)
		if err != nil {
			log.Printf("Could not transform URL encoded data to JSON to pass to Presidio")
			return api.Continue
		}
		f.canAnalyzePIIOnBody = true
	}

	if f.contentType == "application/json" {
		jsonBody = buffer.Bytes()
		f.canAnalyzePIIOnBody = true
	}

	var err error
	if f.canAnalyzePIIOnBody {
		if f.piiTypes, err = piiAnalysis(f.istioHeader.Name, jsonBody); err != nil {
			log.Println(err)
			return api.Continue
		}
	}

	return api.Continue
}

func (f *inboundFilter) DecodeTrailers(trailers api.RequestTrailerMap) api.StatusType {
	log.Println(">>> DECODE TRAILERS")
	log.Printf("%+v", trailers)
	if f.piiTypes != "" {
		trailers.Add("x-prose-pii-types", f.piiTypes) // For OPA
	}
	return api.Continue
}

func (f *inboundFilter) EncodeHeaders(header api.ResponseHeaderMap, endStream bool) api.StatusType {
	//if f.path == "/update_upstream_response" {
	//	header.Set("Content-Length", strconv.Itoa(len(UpdateUpstreamBody)))
	//}
	//header.Set("Rsp-Header-From-Go", "bar-test")

	log.Println("<<< ENCODE HEADERS")

	status, statusWasSet := header.Status()
	log.Printf("Status was set (%v) to %v with response headers:\n", statusWasSet, status)
	header.Range(func(key, value string) bool {
		log.Printf("  \"%v\": %v\n", key, value)
		return true
	})

	return api.Continue
}

// Callbacks which are called in response path
func (f *inboundFilter) EncodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	//if f.path == "/update_upstream_response" {
	//	if endStream {
	//		buffer.SetString(UpdateUpstreamBody)
	//	} else {
	//		// TODO implement buffer->Drain, buffer.SetString means buffer->Drain(buffer.Len())
	//		buffer.SetString("")
	//	}
	//}
	log.Println("<<< ENCODE DATA")
	log.Println("  <<About to forward", buffer.Len(), "bytes of data to client>>")
	return api.Continue
}

func (f *inboundFilter) EncodeTrailers(trailers api.ResponseTrailerMap) api.StatusType {
	log.Println("<<< ENCODE TRAILERS")
	log.Printf("%+v", trailers)
	return api.Continue
}

func (f *inboundFilter) OnDestroy(reason api.DestroyReason) {
}
