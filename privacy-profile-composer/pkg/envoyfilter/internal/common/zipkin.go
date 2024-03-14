package common

import (
	"context"
	"fmt"
	"log"

	envoyapi "github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	"github.com/openzipkin/zipkin-go/reporter"
	httpreporter "github.com/openzipkin/zipkin-go/reporter/http"
)

type ZipkinTracer struct {
	*zipkin.Tracer

	endpoint *model.Endpoint
	reporter reporter.Reporter
}

// NewZipkinTracer creates a wrapped instance of Tracer from zipkin-go package
// It requires full url to be passed, including `/api/v2/spans`.
func NewZipkinTracer(url string) (*ZipkinTracer, error) {
	// batch size 0 forces sending spans right away
	httpReporter := httpreporter.NewReporter(url, httpreporter.BatchSize(0))

	endpoint, err := zipkin.NewEndpoint("golang-filter", "")
	if err != nil {
		_ = httpReporter.Close()
		return nil, fmt.Errorf("unable to create local endpoint: %w", err)
	}

	tracer, err := zipkin.NewTracer(httpReporter, zipkin.WithLocalEndpoint(endpoint))
	if err != nil {
		_ = httpReporter.Close()
		return nil, fmt.Errorf("unable to create tracer: %w", err)
	}

	return &ZipkinTracer{
		Tracer:   tracer,
		endpoint: endpoint,
		reporter: httpReporter,
	}, nil
}

func (t ZipkinTracer) Close() {
	err := t.reporter.Close()
	if err != nil {
		log.Fatalf("unable to close zipkin reporter: %+v", err)
	}
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

func TracerFromContext(ctx context.Context) *ZipkinTracer {
	if t, ok := ctx.Value(tracerKey).(*ZipkinTracer); ok {
		return t
	}
	return nil
}

func AddTracerToContext(ctx context.Context, tracer *ZipkinTracer) context.Context {
	return context.WithValue(ctx, tracerKey, tracer)
}

type tracerCtxKey struct{}

var tracerKey = tracerCtxKey{}
