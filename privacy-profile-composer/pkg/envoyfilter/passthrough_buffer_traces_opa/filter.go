package passthrough_buffer_traces_opa

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/open-policy-agent/opa/sdk"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"

	"privacy-profile-composer/pkg/envoyfilter"
	"privacy-profile-composer/pkg/envoyfilter/internal/common"
)

func FilterFactory(c interface{}) api.StreamFilterFactory {
	conf, ok := c.(*envoyfilter.Config)
	if !ok {
		panic("unexpected config type")
	}

	_, err := common.UpdateTracer(conf.ZipkinUrl)
	if err != nil {
		panic(err)
	}

	return func(callbacks api.FilterCallbackHandler) api.StreamFilter {
		filter, err := newFilter(callbacks, conf)
		if err != nil {
			panic(err)
		}
		return filter
	}
}

func newFilter(callbacks api.FilterCallbackHandler, config *envoyfilter.Config) (api.StreamFilter, error) {
	opaObj, err := sdk.New(context.Background(), sdk.Options{
		ID:     "golang-filter-opa",
		Config: bytes.NewReader([]byte(config.OpaConfig)),
	})

	if err != nil {
		return nil, fmt.Errorf("could not initialize an OPA object --- "+
			"this means that the data plane cannot evaluate the target privacy policy ----- %+v\n", err)
	}

	return &filter{
		callbacks: callbacks,
		config:    config,
		opa:       opaObj,
	}, nil
}

type filter struct {
	api.PassThroughStreamFilter

	callbacks api.FilterCallbackHandler
	config    *envoyfilter.Config
	opa       *sdk.OPA

	parentSpanContext model.SpanContext
	decodeDataBuffer  string
	encodeDataBuffer  string
}

func (f *filter) OnDestroy(reason api.DestroyReason) {
	f.opa.Stop(context.Background())
}

func (f *filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.StatusType {
	f.parentSpanContext = common.GlobalTracer.Extract(header)

	span := common.GlobalTracer.StartSpan(
		"DecodeHeaders",
		zipkin.Parent(f.parentSpanContext),
		zipkin.Tags(map[string]string{
			"endStream": strconv.FormatBool(endStream),
		}),
	)
	defer span.Finish()

	if !endStream {
		return api.StopAndBuffer
	}

	return api.Continue
}

func (f *filter) DecodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	span := common.GlobalTracer.StartSpan(
		"DecodeData",
		zipkin.Parent(f.parentSpanContext),
		zipkin.Tags(map[string]string{
			"endStream": strconv.FormatBool(endStream),
		}),
	)
	defer span.Finish()

	f.decodeDataBuffer += buffer.String()

	if !endStream {
		return api.StopAndBuffer
	}

	return api.Continue
}

func (f *filter) DecodeTrailers(trailers api.RequestTrailerMap) api.StatusType {
	span := common.GlobalTracer.StartSpan("DecodeTrailers", zipkin.Parent(f.parentSpanContext))
	defer span.Finish()

	return api.Continue
}

func (f *filter) EncodeHeaders(header api.ResponseHeaderMap, endStream bool) api.StatusType {
	span := common.GlobalTracer.StartSpan(
		"EncodeHeaders",
		zipkin.Parent(f.parentSpanContext),
		zipkin.Tags(map[string]string{
			"endStream": strconv.FormatBool(endStream),
		}),
	)
	defer span.Finish()

	if !endStream {
		return api.StopAndBuffer
	}

	return api.Continue
}

func (f *filter) EncodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	span := common.GlobalTracer.StartSpan(
		"EncodeData",
		zipkin.Parent(f.parentSpanContext),
		zipkin.Tags(map[string]string{
			"endStream": strconv.FormatBool(endStream),
		}),
	)
	defer span.Finish()

	f.encodeDataBuffer += buffer.String()

	if !endStream {
		return api.StopAndBuffer
	}

	return api.Continue
}

func (f *filter) EncodeTrailers(trailers api.ResponseTrailerMap) api.StatusType {
	span := common.GlobalTracer.StartSpan("EncodeTrailers", zipkin.Parent(f.parentSpanContext))
	defer span.Finish()

	return api.Continue
}
