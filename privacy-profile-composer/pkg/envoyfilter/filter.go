package envoyfilter

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/open-policy-agent/opa/sdk"
	"github.com/open-policy-agent/opa/topdown"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"

	"privacy-profile-composer/pkg/envoyfilter/internal/common"
)

func NewFilter(callbacks api.FilterCallbackHandler, config *config) api.StreamFilter {
	sidecarDirection, err := common.GetDirection(callbacks)
	if err != nil {
		log.Fatal(err)
	}

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

	return &Filter{
		callbacks:        callbacks,
		config:           config,
		tracer:           tracer,
		sidecarDirection: sidecarDirection,
		opa:              opaObj,
	}
}

type Filter struct {
	api.PassThroughStreamFilter

	callbacks        api.FilterCallbackHandler
	config           *config
	tracer           *common.ZipkinTracer
	opa              *sdk.OPA
	sidecarDirection common.SidecarDirection

	// Runtime state of the filter
	parentSpanContext model.SpanContext
	headerMetadata    common.HeaderMetadata
}

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
	span, ctx := f.tracer.StartSpanFromContext(
		context.Background(),
		"DecodeData",
		zipkin.Parent(f.parentSpanContext),
	)
	defer span.Finish()

	log.Println(">>> DECODE DATA")
	log.Println("  <<About to forward", buffer.Len(), "bytes of data to service>>")

	processBody := false
	// If it is an inbound sidecar, then do process the body
	// run PII Analysis + OPA directly
	if f.sidecarDirection == common.Inbound {
		processBody = true
	}

	//  If it is an outbound sidecar, then check if it's a request to a third party
	//  and only process the body in this case
	if f.sidecarDirection == common.Outbound {
		thirdPartyURL, err := f.checkIfRequestToThirdParty()
		if err != nil {
			log.Println(err)
			return api.Continue
		} else if thirdPartyURL == "" {
			log.Printf("outbound sidecar processed a request to another sidecar in the mesh" +
				"Prose will process it through the inbound decode function\n")
			return api.Continue
		}
		processBody = true
	}

	if processBody {
		sendLocalReply, err, proseTags := f.processBody(ctx, buffer, true)
		// Some of these tags may include error info,
		// so need to add them irrespective of the error
		for k, v := range proseTags {
			span.Tag(k, v)
		}
		if err != nil {
			log.Println(err)
			return api.Continue
		}

		// If OPA is configured to an enforce mode (for production),
		// then actually drop the request when it violates the policy
		if sendLocalReply && f.config.opaEnforce {
			body := "OPA target policy rejected the input data"
			f.callbacks.SendLocalReply(403, body, nil, 0, "")
			return api.LocalReply
		}
	}

	return api.Continue
}

func (f *Filter) DecodeTrailers(trailers api.RequestTrailerMap) api.StatusType {
	log.Println(">>> DECODE TRAILERS")
	log.Printf("%+v", trailers)
	return api.Continue
}

func (f *Filter) EncodeHeaders(header api.ResponseHeaderMap, endStream bool) api.StatusType {
	log.Println("<<< ENCODE HEADERS")

	common.LogEncodeHeaderData(header)

	span := f.tracer.StartSpan("test span in encode headers", zipkin.Parent(f.parentSpanContext))
	defer span.Finish()

	return api.Continue
}

// Callbacks which are called in response path
func (f *Filter) EncodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	span, ctx := f.tracer.StartSpanFromContext(
		context.Background(),
		"EncodeData",
		zipkin.Parent(f.parentSpanContext),
	)
	defer span.Finish()

	log.Println("<<< ENCODE DATA")
	log.Println("  <<About to forward", buffer.Len(), "bytes of data to client>>")

	// if outbound then indirect purpose of use violation
	// TODO: This is usually data obtained from another service
	//  but it could also be data obtained from a third party. I.e. a kind of join violation.
	//  Not sure if we'll run into those cases in the examples we look at.
	if f.sidecarDirection == common.Outbound {
		sendLocalReply, err, proseTags := f.processBody(ctx, buffer, false)
		for k, v := range proseTags {
			span.Tag(k, v)
		}
		if err != nil {
			log.Println(err)
			return api.Continue
		}

		// If OPA is configured to an enforce mode (for production),
		// then actually drop the request when it violates the policy
		if sendLocalReply && f.config.opaEnforce {
			body := "OPA target policy rejected the input data"
			f.callbacks.SendLocalReply(403, body, nil, 0, "")
			return api.LocalReply
		}
	}

	// if inbound then ignore
	// we will just address them in the inbound call to the caller svc
	return api.Continue
}

func (f *Filter) EncodeTrailers(trailers api.ResponseTrailerMap) api.StatusType {
	log.Println("<<< ENCODE TRAILERS")
	log.Printf("%+v", trailers)
	return api.Continue
}

func (f *Filter) OnDestroy(reason api.DestroyReason) {
	f.tracer.Close()
	f.opa.Stop(context.Background())
}

func (f *Filter) processBody(ctx context.Context, buffer api.BufferInstance, isDecode bool) (sendLocalReply bool, err error, proseTags map[string]string) {
	span, ctx := f.tracer.StartSpanFromContext(ctx, "processBody")
	defer span.Finish()

	proseTags = map[string]string{}

	jsonBody, err := common.GetJSONBody(f.headerMetadata, buffer)
	if err != nil {
		return false, err, proseTags
	}

	// Run Presidio and add tags for PII types or an error from Presidio
	piiTypes, err := common.PiiAnalysis(f.config.presidioUrl, f.headerMetadata.SvcName, jsonBody)
	if err != nil {
		proseTags[PROSE_PRESIDIO_ERROR] = fmt.Sprintf("%s", err)
		return false, err, proseTags
	}
	proseTags[PROSE_PII_TYPES] = piiTypes

	proseTags[PROSE_OPA_ENFORCE] = strconv.FormatBool(f.config.opaEnforce)

	sendLocalReply, err, opaTags := f.runOPA(ctx, isDecode)
	for k, v := range opaTags {
		proseTags[k] = v
	}

	return sendLocalReply, err, proseTags
}

func (f *Filter) runOPA(ctx context.Context, isDecode bool) (sendLocalReply bool, err error, proseTags map[string]string) {
	proseTags = map[string]string{}

	// get the named policy decision for the specified input
	result, err := f.opa.Decision(
		ctx,
		sdk.DecisionOptions{
			Path: "/authz/allow",
			// TODO: Pass in the purpose of use,
			//  the PII types and optionally, the third parties
			//  (if isDecode is true and f.sidecarDirection is outbound)
			//  following the structure in simple_test.rego
			//  note that those test-cases are potentially out of date wrt simple.rego
			//  as simple.rego expects PII type & purpose to be passed as headers
			//  (i.e. as if we had an OPA sidecar)
			Input:  map[string]interface{}{"hello": "world"},
			Tracer: topdown.NewBufferTracer(),
		},
	)

	if err != nil {
		errStr := fmt.Sprintf("had an error evaluating the policy: %s", err)
		proseTags[PROSE_OPA_ERROR] = errStr
		return false, fmt.Errorf("%s\n", errStr), proseTags
	}

	decision, ok := result.Result.(bool)
	if !ok {
		errStr := fmt.Sprintf("result: Result type: %v", decision)
		proseTags[PROSE_OPA_ERROR] = errStr
		return false, fmt.Errorf("%s\n", errStr), proseTags
	}

	if decision {
		proseTags[PROSE_OPA_DECISION] = "accept"
		log.Printf("policy accepted the input data \n")
		return false, nil, proseTags
	}

	proseTags[PROSE_OPA_DECISION] = "deny"
	log.Printf("policy rejected the input data \n")

	// Ideally, get the reason why it was rejected, e.g. which clause was violated
	//  the result.Provenance field includes version info, bundle info etc.
	//  https://github.com/open-policy-agent/opa/pull/5460
	//  but afaict the "explanation" is through a special tracer that they built-in to OPA
	//  https://github.com/open-policy-agent/opa/pull/5447
	//  can initialize it in the DecisionOptions above

	// Include a tag for the violation type
	if isDecode {
		if f.sidecarDirection == common.Outbound {
			proseTags[PROSE_VIOLATION_TYPE] = DataSharing
		} else { // inbound sidecar within decode method
			proseTags[PROSE_VIOLATION_TYPE] = PurposeOfUseDirect
		}
	} else { // encode method
		if f.sidecarDirection == common.Outbound {
			proseTags[PROSE_VIOLATION_TYPE] = PurposeOfUseIndirect
		}
		// we don't call this method (from EncodeData) if it's an inbound sidecar
	}

	return true, nil, proseTags
}

func (f *Filter) checkIfRequestToThirdParty() (string, error) {
	// use f.callbacks
	// use f.headerMetadata
	return "", nil
}
