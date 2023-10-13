package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"net/http"
	pb "privacy-profile-composer/pkg/proto"
)

var UpdateUpstreamBody = "upstream response body updated by the simple plugin"

type filter struct {
	api.PassThroughStreamFilter

	callbacks     api.FilterCallbackHandler
	path          string
	method        string
	contentType   string
	contentLength string
	host          string
	istioHeader   XEnvoyPeerMetadataHeader
	config        *config
}

func (f *filter) sendLocalReplyInternal() api.StatusType {
	body := fmt.Sprintf("%s, path: %s\r\n", f.config.echoBody, f.path)
	f.callbacks.SendLocalReply(200, body, nil, 0, "")
	return api.LocalReply
}

// Callbacks which are called in request path
func (f *filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.StatusType {
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
		parsedHeader, err := DecodeXEnvoyPeerMetadataHeader(xEnvoyPeerMetadata)
		if err != nil {
			log.Printf("Error decoding x-envoy-peer-metadata header: %s", err)
			return api.Continue
		}
		f.istioHeader = parsedHeader
	}

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

func (f *filter) DecodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	log.Println(">>> DECODE DATA")
	log.Println("  <<About to forward", buffer.Len(), "bytes of data to service>>")

	if f.contentType == "application/x-www-form-urlencoded" {
		jsonBody := json.NewDecoder(bytes.NewReader(buffer.Bytes()))
		if jsonBody == nil {
			log.Printf("Failed to start decoding JSON data")
			return api.Continue
		}
		log.Printf("  <<decoded data: ", jsonBody)
	}

	//if f.contentType == "application/json" {
	var jsonBody = []byte(`{
		"json_to_analyze": {
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
		}
	}`)
	resp, err := http.Post("http://presidio.prose-system.svc.cluster.local:3000/batchanalyze", "application/json", bytes.NewBuffer(jsonBody))
	// var jsonData = buffer.Bytes()
	//resp2, err := http.PostForm("http://presidio.prose-system.svc.cluster.local:3000/batchanalyze",
	//	url.Values{"json_to_analyze": {string(jsonData)}})

	if err != nil {
		log.Printf("presidio post error: %v\n", err.Error())
		return api.Continue
	}

	log.Printf("presidio responded '%v', content-length is %v bytes\n", resp.Status, resp.ContentLength)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Could not read Presidio response, %v\n", err.Error())
		return api.Continue
	}

	err = resp.Body.Close()
	if err != nil {
		log.Printf("could not close presidio response body, %v\n", err.Error())
		return api.Continue
	}

	log.Println("presidio response headers:")
	for key, value := range resp.Header {
		log.Printf("  \"%v\": %v\n", key, value)
	}
	log.Println("presidio response body:")
	log.Printf("%v\n", string(body))

	//}

	piiTypes := []string{"EMAIL", "LOCATION"}
	thirdParties := make([]string, 0)
	return sendComposedProfile("advertising.svc.internal", "advertising", piiTypes, thirdParties)
}

func (f *filter) DecodeTrailers(trailers api.RequestTrailerMap) api.StatusType {
	log.Println(">>> DECODE TRAILERS")
	log.Printf("%+v", trailers)
	return api.Continue
}

func (f *filter) EncodeHeaders(header api.ResponseHeaderMap, endStream bool) api.StatusType {
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
func (f *filter) EncodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
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

func (f *filter) EncodeTrailers(trailers api.ResponseTrailerMap) api.StatusType {
	log.Println("<<< ENCODE TRAILERS")
	log.Printf("%+v", trailers)
	return api.Continue
}

func (f *filter) OnDestroy(reason api.DestroyReason) {
}
