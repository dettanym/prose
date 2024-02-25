package envoyfilter

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/url"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/open-policy-agent/opa/sdk"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"privacy-profile-composer/pkg/envoyfilter/internal/common"
	pb "privacy-profile-composer/pkg/proto"
)

func NewInboundFilter(callbacks api.FilterCallbackHandler, config *config) api.StreamFilter {
	tracer, err := common.NewZipkinTracer(config.zipkinUrl)
	if err != nil {
		log.Fatalf("unable to create tracer: %+v\n", err)
	}

	opaObj, err := sdk.New(context.Background(), sdk.Options{
		ID:     "golang-filter-opa",
		Config: bytes.NewReader([]byte(config.opaConfig)),
	})

	if err != nil {
		log.Fatalf("could not initialize an OPA object --- "+
			"this means that the data plane cannot evaluate the target privacy policy ----- %+v\n", err)
	}

	return &inboundFilter{callbacks: callbacks, config: config, tracer: tracer, opa: opaObj}
}

type inboundFilter struct {
	api.PassThroughStreamFilter

	callbacks api.FilterCallbackHandler
	config    *config

	parentSpanContext model.SpanContext
	headerMetadata    common.HeaderMetadata
	piiTypes          string
	tracer            *common.ZipkinTracer
	opa               *sdk.OPA
}

// Callbacks which are called in request path
func (f *inboundFilter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.StatusType {
	log.Println(">>> DECODE HEADERS")

	f.parentSpanContext = f.tracer.Extract(header)

	span := f.tracer.StartSpan("test span in decode headers", zipkin.Parent(f.parentSpanContext))
	defer span.Finish()

	f.headerMetadata = common.ExtractHeaderData(header)

	// TODO: Insert it into OpenTelemetry baggage for tracing?
	header.Add("x-prose-purpose", f.headerMetadata.Purpose) // For OPA

	common.LogDecodeHeaderData(header)

	// get the named policy decision for the specified input
	if result, err := f.opa.Decision(context.Background(), sdk.DecisionOptions{Path: "/authz/allow", Input: map[string]interface{}{"hello": "world"}}); err != nil {
		log.Printf("had an error evaluating the policy: %s\n", err)
	} else if decision, ok := result.Result.(bool); !ok || !decision {
		log.Printf("result: descision: %v, ok: %v\n", decision, ok)
	} else {
		log.Printf("policy accepted the input data \n")
	}

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

func (f *inboundFilter) DecodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	log.Println(">>> DECODE DATA")
	log.Println("  <<About to forward", buffer.Len(), "bytes of data to service>>")

	canAnalyzePIIOnBody := false
	var jsonBody []byte

	if f.headerMetadata.ContentType == nil {
		log.Println("ContentType header is not set. Cannot analyze body")
	} else if *f.headerMetadata.ContentType == "application/x-www-form-urlencoded" {
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
		canAnalyzePIIOnBody = true
	} else if *f.headerMetadata.ContentType == "application/json" {
		jsonBody = buffer.Bytes()
		canAnalyzePIIOnBody = true
	} else {
		log.Printf("Cannot analyze a body with contentType '%s'\n", f.headerMetadata.ContentType)
	}

	if !canAnalyzePIIOnBody {
		return api.Continue
	}

	var err error
	if f.piiTypes, err = common.PiiAnalysis(f.config.presidioUrl, f.headerMetadata.SvcName, jsonBody); err != nil {
		log.Println(err)
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
	//if f.headerMetadata.Path == "/update_upstream_response" {
	//	header.Set("Content-Length", strconv.Itoa(len(UpdateUpstreamBody)))
	//}
	//header.Set("Rsp-Header-From-Go", "bar-test")

	log.Println("<<< ENCODE HEADERS")

	common.LogEncodeHeaderData(header)

	span := f.tracer.StartSpan("test span in encode headers", zipkin.Parent(f.parentSpanContext))
	defer span.Finish()

	return api.Continue
}

// Callbacks which are called in response path
func (f *inboundFilter) EncodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	//if f.headerMetadata.Path == "/update_upstream_response" {
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
	f.tracer.Close()
	f.opa.Stop(context.Background())
}
