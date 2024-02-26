package envoyfilter

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/url"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/open-policy-agent/opa/sdk"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"privacy-profile-composer/pkg/envoyfilter/internal/common"
)

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

func NewInboundFilter(callbacks api.FilterCallbackHandler, config *config) api.StreamFilter {
	tracer, err := common.NewZipkinTracer(config.zipkinUrl)
	if err != nil {
		log.Fatalf("unable to create tracer: %+v\n", err)
	}

	if config.opaEnable {
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
	return &inboundFilter{callbacks: callbacks, config: config, tracer: tracer}
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

	if f.config.opaEnable {
		// get the named policy decision for the specified input
		if result, err := f.opa.Decision(context.Background(), sdk.DecisionOptions{Path: "/authz/allow", Input: map[string]interface{}{"hello": "world"}}); err != nil {
			log.Printf("had an error evaluating the policy: %s\n", err)
		} else if decision, ok := result.Result.(bool); !ok || !decision {
			log.Printf("result: descision: %v, ok: %v\n", decision, ok)
		} else {
			log.Printf("policy accepted the input data \n")
		}
	}

	return api.Continue
}

func (f *inboundFilter) DecodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	log.Println(">>> DECODE DATA")
	log.Println("  <<About to forward", buffer.Len(), "bytes of data to service>>")

	var jsonBody []byte
	var err error
	if f.headerMetadata.ContentType == nil {
		log.Println("ContentType header is not set. Cannot analyze body")
		return api.Continue
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
	} else if *f.headerMetadata.ContentType == "application/json" {
		jsonBody = buffer.Bytes()
	} else {
		log.Printf("Cannot analyze a body with contentType '%s'\n", f.headerMetadata.ContentType)
		return api.Continue
	}

	if f.piiTypes, err = common.PiiAnalysis(f.config.presidioUrl, f.headerMetadata.SvcName, jsonBody); err != nil {
		log.Println(err)
		return api.Continue
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
	if f.config.opaEnable {
		f.opa.Stop(context.Background())
	}
}
