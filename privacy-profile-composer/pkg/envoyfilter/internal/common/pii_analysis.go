package common

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type PresidioDataFormat struct {
	JsonToAnalyze interface{} `json:"json_to_analyze"`
	DerivePurpose string      `json:"derive_purpose,omitempty"`
}

func PiiAnalysis(ctx context.Context, presidioSvcURL string, svcName string, bufferBytes interface{}) ([]string, error) {
	span, ctx := GlobalTracer.StartSpanFromContext(ctx, "PiiAnalysis")
	defer span.Finish()

	otelSpanCtx, err := otelSpanContextFromZipkin(span.Context())
	if err != nil {
		fmt.Printf("Error converting span context: %v\n", err)
	}

	ctx = trace.ContextWithSpanContext(ctx, otelSpanCtx)

	empty := []string{}

	_, err = json.Marshal(
		PresidioDataFormat{
			JsonToAnalyze: bufferBytes,
			DerivePurpose: svcName,
		},
	)
	if err != nil {
		return empty, fmt.Errorf("could not convert data for presidio into json: %w", err)
	}

	time.Sleep(time.Duration(20) * time.Millisecond)

	return empty, nil
}
