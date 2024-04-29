package common

import (
	"fmt"

	"github.com/openzipkin/zipkin-go/model"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var GlobalOtelPropagator = propagation.NewCompositeTextMapPropagator(
	propagation.TraceContext{},
	propagation.Baggage{},
)

func otelSpanContextFromZipkin(span model.SpanContext) (trace.SpanContext, error) {
	traceFlags := trace.TraceFlags(0)
	if span.Sampled == nil {
		traceFlags = traceFlags.WithSampled(false)
	} else {
		traceFlags = traceFlags.WithSampled(*span.Sampled)
	}

	traceID, err := trace.TraceIDFromHex(fmt.Sprintf("%032s", span.TraceID.String()))
	if err != nil {
		return trace.NewSpanContext(
			trace.SpanContextConfig{
				TraceFlags: traceFlags,
				TraceState: trace.TraceState{},
				Remote:     false,
			},
		), fmt.Errorf("invalid trace ID: %w", err)
	}

	spanID, err := trace.SpanIDFromHex(span.ID.String())
	if err != nil {
		return trace.NewSpanContext(
			trace.SpanContextConfig{
				TraceID:    traceID,
				TraceFlags: traceFlags,
				TraceState: trace.TraceState{},
				Remote:     false,
			},
		), fmt.Errorf("invalid span ID: %w", err)
	}

	return trace.NewSpanContext(
		trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: traceFlags,
			TraceState: trace.TraceState{},
			Remote:     false,
		},
	), nil
}
