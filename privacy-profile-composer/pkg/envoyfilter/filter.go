package envoyfilter

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/open-policy-agent/opa/sdk"
	"github.com/open-policy-agent/opa/topdown"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"

	"privacy-profile-composer/pkg/envoyfilter/internal/common"
)

func NewFilter(callbacks api.FilterCallbackHandler, config *Config) (api.StreamFilter, error) {
	return &Filter{
		callbacks: callbacks,
		config:    config,
	}, nil
}

type Filter struct {
	api.PassThroughStreamFilter

	callbacks api.FilterCallbackHandler
	config    *Config

	// Runtime state of the filter
	parentSpanContext model.SpanContext
	reqHeaderMetadata common.RequestHeaderMetadata
	resHeaderMetadata common.ResponseHeaderMetadata
	thirdPartyURL     string
	processDecodeBody bool
	decodeDataBuffer  string
	processEncodeBody bool
	encodeDataBuffer  string
}

// Callbacks which are called in request path
func (f *Filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.StatusType {
	// log.Println(">>> DECODE HEADERS")

	f.parentSpanContext = common.GlobalTracer.Extract(header)

	span := common.GlobalTracer.StartSpan("test span in decode headers", zipkin.Parent(f.parentSpanContext))
	defer span.Finish()

	span.Tag("SIDECAR_DIRECTION", string(f.config.direction))
	span.Tag("DATA_FLOW", "DECODE_HEADERS")

	f.reqHeaderMetadata = common.ExtractRequestHeaderData(header)

	// common.LogDecodeHeaderData(header)

	if endStream {
		// here we have a header-only request
		return api.Continue
	}

	switch f.config.direction {
	case common.Inbound:
		// If it is an inbound sidecar, then do process the body
		// run PII Analysis + OPA directly
		f.processDecodeBody = true

	case common.Outbound:
		// If it is an outbound sidecar, then check if it's a request to a third party
		// and only process the body in this case
		destinationAddress, err := f.callbacks.GetProperty("destination.address")
		if err != nil {
			log.Println(err)
			return api.Continue
		}

		isInternalDestination, err := f.checkInternalAddress(destinationAddress)
		if err != nil {
			log.Println(err)
			return api.Continue
		}

		if isInternalDestination {
			// log.Printf("outbound sidecar processed a request to another sidecar in the mesh" +
			// 	"Prose will process it through the inbound decode function\n")
			return api.Continue
		}

		f.thirdPartyURL = f.reqHeaderMetadata.Host
		f.processDecodeBody = true

	default:
		log.Printf("unexpected filter direction: %s\n", f.config.direction)
		return api.Continue
	}

	return api.StopAndBuffer
}

func (f *Filter) DecodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	// TODO: we might need to be careful about collecting the data from all
	//  of these buffers. Maybe go has some builtin methods to work with it,
	//  instead of us collecting the entire body using string concat.
	// https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/buffer_filter
	// https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/file_system_buffer_filter
	// There might be data in `buffer` regardless of the `endStream` flag, so
	// it always needs to be collected.
	f.decodeDataBuffer += buffer.String()

	if !endStream {
		return api.StopAndBuffer
	}

	span, ctx := common.GlobalTracer.StartSpanFromContext(
		context.Background(),
		"DecodeData",
		zipkin.Parent(f.parentSpanContext),
	)
	defer span.Finish()

	span.Tag("SIDECAR_DIRECTION", string(f.config.direction))
	span.Tag("DATA_FLOW", "DECODE_DATA")

	// log.Println(">>> DECODE DATA")
	// log.Println("  <<About to forward", len(f.decodeDataBuffer), "bytes of data to service>>")

	// span.Tag("buffer-value", f.decodeDataBuffer)

	if f.processDecodeBody {
		sendLocalReply, err, proseTags := f.processBody(ctx, f.decodeDataBuffer, true)
		// Some of these tags may include error info,
		// so need to add them irrespective of the error
		for k, v := range proseTags {
			span.Tag(k, v)
		}
		if err != nil {
			// log.Println(err)
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
	// log.Println(">>> DECODE TRAILERS")
	// log.Printf("%+v", trailers)
	return api.Continue
}

func (f *Filter) EncodeHeaders(header api.ResponseHeaderMap, endStream bool) api.StatusType {
	// log.Println("<<< ENCODE HEADERS")

	// common.LogEncodeHeaderData(header)

	span := common.GlobalTracer.StartSpan("test span in encode headers", zipkin.Parent(f.parentSpanContext))
	defer span.Finish()

	span.Tag("SIDECAR_DIRECTION", string(f.config.direction))
	span.Tag("DATA_FLOW", "ENCODE_HEADERS")

	f.resHeaderMetadata = common.ExtractResponseHeaderData(header)

	if endStream {
		// here we have a header-only request
		return api.Continue
	}

	switch f.config.direction {
	case common.Inbound:
		// if inbound then ignore
		// we will just address them in the inbound call to the caller svc
		f.processEncodeBody = false

	case common.Outbound:
		// if outbound then indirect purpose of use violation
		// TODO: This is usually data obtained from another service
		//  but it could also be data obtained from a third party. I.e. a kind of join violation.
		//  Not sure if we'll run into those cases in the examples we look at.
		f.processEncodeBody = true

	default:
		log.Printf("unexpected filter direction: %s\n", f.config.direction)
		return api.Continue
	}

	return api.StopAndBuffer
}

// Callbacks which are called in response path
func (f *Filter) EncodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	// TODO: we might need to be careful about collecting the data from all
	//  of these buffers. Maybe go has some builtin methods to work with it,
	//  instead of us collecting the entire body using string concat.
	// https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/buffer_filter
	// https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/file_system_buffer_filter
	f.encodeDataBuffer += buffer.String()

	if !endStream {
		return api.StopAndBuffer
	}

	span, ctx := common.GlobalTracer.StartSpanFromContext(
		context.Background(),
		"EncodeData",
		zipkin.Parent(f.parentSpanContext),
	)
	defer span.Finish()

	span.Tag("SIDECAR_DIRECTION", string(f.config.direction))
	span.Tag("DATA_FLOW", "ENCODE_DATA")

	// log.Println("<<< ENCODE DATA")
	// log.Println("  <<About to forward", len(f.encodeDataBuffer), "bytes of data to client>>")

	// span.Tag("buffer-value", f.encodeDataBuffer)

	if f.processEncodeBody {
		sendLocalReply, err, proseTags := f.processBody(ctx, f.encodeDataBuffer, false)
		for k, v := range proseTags {
			span.Tag(k, v)
		}
		if err != nil {
			// log.Println(err)
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

func (f *Filter) EncodeTrailers(trailers api.ResponseTrailerMap) api.StatusType {
	// log.Println("<<< ENCODE TRAILERS")
	// log.Printf("%+v", trailers)
	return api.Continue
}

func (f *Filter) OnDestroy(reason api.DestroyReason) {
}

func (f *Filter) processBody(ctx context.Context, body string, isDecode bool) (sendLocalReply bool, err error, proseTags map[string]string) {
	span, ctx := common.GlobalTracer.StartSpanFromContext(ctx, "processBody")
	defer span.Finish()

	proseTags = map[string]string{}

	proseTags[PROSE_SIDECAR_DIRECTION] = string(f.config.direction)

	var contentType *string
	if isDecode {
		contentType = f.reqHeaderMetadata.ContentType
	} else {
		contentType = f.resHeaderMetadata.ContentType
	}

	jsonBody, err := common.GetJSONBody(ctx, contentType, body)
	if err != nil {
		proseTags[PROSE_JSON_BODY_ERROR] = fmt.Sprintf("%s", err)
		return false, err, proseTags
	}

	// Run Presidio and add tags for PII types or an error from Presidio
	piiTypes, err := common.PiiAnalysis(ctx, f.config.presidioUrl, f.reqHeaderMetadata.SvcName, jsonBody)
	if err != nil {
		proseTags[PROSE_PRESIDIO_ERROR] = fmt.Sprintf("%s", err)
		return false, err, proseTags
	}
	proseTags[PROSE_PII_TYPES] = strings.Join(piiTypes, ",")

	proseTags[PROSE_OPA_ENFORCE] = strconv.FormatBool(f.config.opaEnforce)

	sendLocalReply, err, opaTags := f.runOPA(ctx, isDecode, piiTypes)
	for k, v := range opaTags {
		proseTags[k] = v
	}

	return sendLocalReply, err, proseTags
}

func (f *Filter) runOPA(ctx context.Context, isDecode bool, dataItems []string) (sendLocalReply bool, err error, proseTags map[string]string) {
	span, ctx := common.GlobalTracer.StartSpanFromContext(ctx, "runOPA")
	defer span.Finish()

	proseTags = map[string]string{}

	// get the named policy decision for the specified input
	result, err := common.GlobalAuthAgent.Decision(
		ctx,
		sdk.DecisionOptions{
			Path: "/prose/authz_logic/allow",
			// TODO: Pass in the purpose of use,
			//  the PII types and optionally, the third parties
			//  (if isDecode is true and f.sidecarDirection is outbound)
			//  following the structure in simple_test.rego
			//  note that those test-cases are potentially out of date wrt simple.rego
			//  as simple.rego expects PII type & purpose to be passed as headers
			//  (i.e. as if we had an OPA sidecar)
			Input: map[string]interface{}{
				"purpose_of_use": f.config.purpose,
				"data_items":     dataItems,
				// todo double check that this is non-null only in outbound and decode mode
				"external_domain": f.thirdPartyURL, // path or null
			},
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
		// log.Printf("policy accepted the input data \n")
		return false, nil, proseTags
	}

	proseTags[PROSE_OPA_DECISION] = "deny"
	// log.Printf("policy rejected the input data \n")

	// Ideally, get the reason why it was rejected, e.g. which clause was violated
	//  the result.Provenance field includes version info, bundle info etc.
	//  https://github.com/open-policy-agent/opa/pull/5460
	//  but afaict the "explanation" is through a special tracer that they built-in to OPA
	//  https://github.com/open-policy-agent/opa/pull/5447
	//  can initialize it in the DecisionOptions above

	// Include a tag for the violation type
	if isDecode {
		if f.config.direction == common.Outbound {
			proseTags[PROSE_VIOLATION_TYPE] = DataSharing
		} else { // inbound sidecar within decode method
			proseTags[PROSE_VIOLATION_TYPE] = PurposeOfUseDirect
		}
	} else { // encode method
		if f.config.direction == common.Outbound {
			proseTags[PROSE_VIOLATION_TYPE] = PurposeOfUseIndirect
		}
		// we don't call this method (from EncodeData) if it's an inbound sidecar
	}

	return true, nil, proseTags
}

func (f *Filter) checkInternalAddress(destinationAddress string) (bool, error) {

	hostIpStr, _, err := net.SplitHostPort(destinationAddress)
	if err != nil {
		return false, err
	}

	hostIp := net.ParseIP(hostIpStr)
	if hostIp == nil {
		return false, fmt.Errorf("invalid IP address: %s", hostIpStr)
	}

	for _, cidr := range f.config.internalCidrs {
		if cidr.Contains(hostIp) {
			return true, nil
		}
	}

	return false, nil
}
