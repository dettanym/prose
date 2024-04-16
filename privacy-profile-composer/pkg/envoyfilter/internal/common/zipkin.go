package common

import (
	"fmt"

	envoyapi "github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	"github.com/openzipkin/zipkin-go/reporter"
	httpreporter "github.com/openzipkin/zipkin-go/reporter/http"
)

var GlobalTracer *ZipkinTracer

func init() {
	tracer, err := NewZipkinTracer("")
	if err != nil {
		panic(err)
	}

	GlobalTracer = tracer
}

func UpdateTracer(url string) (*ZipkinTracer, error) {
	if GlobalTracer.Url != url {
		tracer, err := NewZipkinTracer(url)
		if err != nil {
			return nil, err
		}

		err = GlobalTracer.Close()
		if err != nil {
			_ = tracer.Close()
			return nil, err
		}

		GlobalTracer = tracer
	}

	return GlobalTracer, nil
}

type ZipkinTracer struct {
	*zipkin.Tracer

	Url string

	endpoint *model.Endpoint
	reporter reporter.Reporter
}

// NewZipkinTracer creates a wrapped instance of Tracer from zipkin-go package
// It requires full url to be passed, including `/api/v2/spans`.
func NewZipkinTracer(url string) (*ZipkinTracer, error) {
	var tracerOptions []zipkin.TracerOption

	var rep reporter.Reporter
	if url == "" {
		rep = reporter.NewNoopReporter()
		tracerOptions = append(tracerOptions, zipkin.WithNoopTracer(true))
	} else {
		rep = httpreporter.NewReporter(url)
	}

	endpoint, err := zipkin.NewEndpoint("golang-filter", "")
	if err != nil {
		_ = rep.Close()
		return nil, fmt.Errorf("unable to create local endpoint: %w", err)
	}

	tracerOptions = append(tracerOptions, zipkin.WithLocalEndpoint(endpoint))

	tracer, err := zipkin.NewTracer(rep, tracerOptions...)
	if err != nil {
		_ = rep.Close()
		return nil, fmt.Errorf("unable to create tracer: %w", err)
	}

	return &ZipkinTracer{
		Tracer:   tracer,
		Url:      url,
		endpoint: endpoint,
		reporter: rep,
	}, nil
}

func (t ZipkinTracer) Close() error {
	return t.reporter.Close()
}

func (t ZipkinTracer) Extract(h envoyapi.HeaderMap) model.SpanContext {
	return t.Tracer.Extract(func() (*model.SpanContext, error) {
		// based on https://github.com/openzipkin/zipkin-go/blob/e84b2cf6d2d915fe0ee57c2dc4d736ec13a2ef6a/propagation/b3/http.go#L53
		var (
			traceIDHeader, _      = h.Get(b3.TraceID)
			spanIDHeader, _       = h.Get(b3.SpanID)
			parentSpanIDHeader, _ = h.Get(b3.ParentSpanID)
			sampledHeader, _      = h.Get(b3.Sampled)
			flagsHeader, _        = h.Get(b3.Flags)
			singleHeader, _       = h.Get(b3.Context)
		)

		var (
			sc   *model.SpanContext
			sErr error
			mErr error
		)
		if singleHeader != "" {
			sc, sErr = b3.ParseSingleHeader(singleHeader)
			if sErr == nil {
				return sc, nil
			}
		}

		sc, mErr = b3.ParseHeaders(
			traceIDHeader, spanIDHeader, parentSpanIDHeader,
			sampledHeader, flagsHeader,
		)

		if mErr != nil && sErr != nil {
			return nil, sErr
		}

		return sc, mErr
	})

}
