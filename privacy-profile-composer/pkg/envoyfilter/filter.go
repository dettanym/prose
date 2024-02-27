package envoyfilter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/open-policy-agent/opa/sdk"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"log"
	"net/url"
	"privacy-profile-composer/pkg/envoyfilter/internal/common"
	"strconv"
)

func NewFilter(callbacks api.FilterCallbackHandler, config *config) api.StreamFilter {
	sidecarDirection, err := getDirection(callbacks)
	if err != nil {
		log.Fatal(err)
	}

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
		return &Filter{
			callbacks:        callbacks,
			config:           config,
			tracer:           tracer,
			sidecarDirection: sidecarDirection,
			opa:              opaObj,
		}
	}
	return &Filter{
		callbacks:        callbacks,
		config:           config,
		tracer:           tracer,
		sidecarDirection: sidecarDirection}
}

type Filter struct {
	api.PassThroughStreamFilter

	callbacks        api.FilterCallbackHandler
	config           *config
	tracer           *common.ZipkinTracer
	opa              *sdk.OPA
	sidecarDirection SidecarDirection

	// Runtime state of the filter
	parentSpanContext model.SpanContext
	headerMetadata    common.HeaderMetadata
	piiTypes          string
}

type SidecarDirection int

const (
	Inbound SidecarDirection = iota
	Outbound
)

// Callbacks which are called in request path
func (f *Filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.StatusType {
	log.Println(">>> DECODE HEADERS")

	f.parentSpanContext = f.tracer.Extract(header)

	span := f.tracer.StartSpan("test span in decode headers", zipkin.Parent(f.parentSpanContext))
	defer span.Finish()

	f.headerMetadata = common.ExtractHeaderData(header)

	// TODO: Insert it into OpenTelemetry baggage for tracing?
	header.Add("x-prose-purpose", f.headerMetadata.Purpose) // For OPA

	common.LogDecodeHeaderData(header)

	return api.Continue
}

func (f *Filter) DecodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	log.Println(">>> DECODE DATA")
	log.Println("  <<About to forward", buffer.Len(), "bytes of data to service>>")

	var jsonBody []byte

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

	var err error
	if f.piiTypes, err = common.PiiAnalysis(f.config.presidioUrl, f.headerMetadata.SvcName, jsonBody); err != nil {
		log.Println(err)
		return api.Continue
	}

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

func (f *Filter) DecodeTrailers(trailers api.RequestTrailerMap) api.StatusType {
	log.Println(">>> DECODE TRAILERS")
	log.Printf("%+v", trailers)
	if f.piiTypes != "" {
		trailers.Add("x-prose-pii-types", f.piiTypes) // For OPA
	}
	return api.Continue
}

func (f *Filter) EncodeHeaders(header api.ResponseHeaderMap, endStream bool) api.StatusType {
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
func (f *Filter) EncodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
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

func (f *Filter) EncodeTrailers(trailers api.ResponseTrailerMap) api.StatusType {
	log.Println("<<< ENCODE TRAILERS")
	log.Printf("%+v", trailers)
	return api.Continue
}

func (f *Filter) OnDestroy(reason api.DestroyReason) {
	f.tracer.Close()
	if f.config.opaEnable {
		f.opa.Stop(context.Background())
	}
}

func getDirection(callbacks api.FilterCallbackHandler) (SidecarDirection, error) {
	directionEnum, err := callbacks.GetProperty("xds.listener_direction")
	if err != nil {
		return -1, fmt.Errorf("cannot determine sidecar direction as there is no xds.listener_direction key")
	}
	directionInt, err := strconv.Atoi(directionEnum)
	if err != nil {
		// check https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/core/v3/base.proto#envoy-v3-api-enum-config-core-v3-trafficdirection
		return -1, fmt.Errorf("envoy's xds.listener_direction key does not contain an integer " +
			"check the Envoy docs for the range of values for this key")
	}

	if directionInt == 0 {
		return -1, fmt.Errorf("envoy's xds.listener_direction key indicates that this sidecar is deployed as a gateway." +
			"Prose does not need to be run in a gateway sidecar." +
			"It will continue to get deployed in other sidecars that are configured as inbound or outbound sidecars")
	}

	if directionInt == 1 {
		return Inbound, nil
	}
	if directionInt == 2 {
		return Outbound, nil
	}

	return -1, fmt.Errorf("envoy's xds.listener_direction key contains an unsupported value for the direction enum: %d "+
		"check the Envoy docs for the range of values for this key", directionInt)
}
