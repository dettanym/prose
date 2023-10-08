package main

import (
	"bytes"
	"fmt"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"log"
	"net/http"
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
	config        *config
}

func (f *filter) sendLocalReplyInternal() api.StatusType {
	body := fmt.Sprintf("%s, path: %s\r\n", f.config.echoBody, f.path)
	f.callbacks.SendLocalReply(200, body, nil, 0, "")
	return api.LocalReply
}

// Callbacks which are called in request path
func (f *filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.StatusType {
	log.Println("+++ DECODE HEADERS")

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

	log.Printf("%v (%v) %v://%v%v\n", header.Method(), header.Protocol(), header.Scheme(), header.Host(), header.Path())
	header.Range(func(key, value string) bool {
		log.Printf("  \"%v\": %v\n", key, value)
		return true
	})

	return api.Continue
}

func (f *filter) DecodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	log.Println("+++ DECODE DATA")
	log.Println("  <<About to send", buffer.Len(), "bytes of data>>")

	//if f.contentType == "application/x-www-form-urlencoded" {
	//	dec := json.NewDecoder(bytes.NewReader(buffer.Bytes()))
	//	if dec == nil {
	//		log.Printf("Failed to start decoding JSON data")
	//		return api.Continue
	//	}
	//}

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
		log.Printf("presidio post error: ", err.Error())
		return api.Continue
	}
	contentLen := resp.ContentLength
	body := make([]byte, contentLen)
	read, err := resp.Body.Read(body)
	if err != nil {
		log.Printf("Could not read Presidio response, %v", err.Error())
		return api.Continue
	}
	err = resp.Body.Close()
	if err != nil {
		log.Printf("could not close presidio response body ", err.Error())
		return api.Continue
	}
	log.Printf("Presidio response status", resp.Status)
	for key, value := range resp.Header {
		log.Printf("Presidio response. Key: ", key, " Value: ", value)
	}
	log.Printf("Presidio response body --- read ", read, "bytes. Body string: ", body)

	//}
	return api.Continue
}

func (f *filter) DecodeTrailers(trailers api.RequestTrailerMap) api.StatusType {
	log.Println("+++ DECODE TRAILERS")
	log.Printf("%+v", trailers)
	return api.Continue
}

func (f *filter) EncodeHeaders(header api.ResponseHeaderMap, endStream bool) api.StatusType {
	//if f.path == "/update_upstream_response" {
	//	header.Set("Content-Length", strconv.Itoa(len(UpdateUpstreamBody)))
	//}
	//header.Set("Rsp-Header-From-Go", "bar-test")

	log.Println("+++ ENCODE HEADERS")
	log.Printf("%+v", header)
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
	log.Println("+++ ENCODE DATA")
	log.Printf("%+v", buffer)
	return api.Continue
}

func (f *filter) EncodeTrailers(trailers api.ResponseTrailerMap) api.StatusType {
	log.Println("+++ ENCODE TRAILERS")
	log.Printf("%+v", trailers)
	return api.Continue
}

func (f *filter) OnDestroy(reason api.DestroyReason) {
}

//var (
//	// addr = flag.String("addr", "localhost:50051", "the address to connect to")
//	addr = flag.String("addr", "prose-system.svc.cluster.local", "the address to connect to")
//)
//
//func setup() {
//	flag.Parse()
//	// Set up a connection to the server.
//	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
//	if err != nil {
//		log.Fatalf("did not connect: %v", err)
//	}
//	defer conn.Close()
//	c := pb.NewPrivacyProfileComposerClient(conn)
//
//	// Contact the server and print out its response.
//	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
//	defer cancel()
//
//	_, err = c.PostObservedProfile(
//		ctx,
//		&pb.SvcObservedProfile{
//			SvcInternalFQDN: "advertising.svc.internal",
//			ObservedProcessingEntries: &pb.PurposeBasedProcessing{
//				ProcessingEntries: nil},
//		},
//	)
//	if err != nil {
//		log.Fatalf("could not post observed profile: %v", err)
//	}
//
//	profile, err := c.GetSystemWideProfile(ctx, &emptypb.Empty{})
//	if err != nil {
//		log.Fatalf("could not fetch system wide profile: %v", err)
//	}
//	log.Println(profile)
//}
