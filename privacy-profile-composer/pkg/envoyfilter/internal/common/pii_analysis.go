package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type PresidioDataFormat struct {
	JsonToAnalyze interface{} `json:"json_to_analyze"`
	DerivePurpose string      `json:"derive_purpose,omitempty"`
}

func PiiAnalysis(
	ctx context.Context,
	disablePresidioRequest bool,
	hardcodedPiiTypes *[]string,
	presidioSvcURL string,
	svcName string,
	bufferBytes interface{},
) ([]string, error) {
	span, ctx := GlobalTracer.StartSpanFromContext(ctx, "PiiAnalysis")
	defer span.Finish()

	otelSpanCtx, err := otelSpanContextFromZipkin(span.Context())
	if err != nil {
		fmt.Printf("Error converting span context: %v\n", err)
	}

	ctx = trace.ContextWithSpanContext(ctx, otelSpanCtx)

	empty := []string{}

	msgString, err := json.Marshal(
		PresidioDataFormat{
			JsonToAnalyze: bufferBytes,
			DerivePurpose: svcName,
		},
	)
	if err != nil {
		return empty, fmt.Errorf("could not convert data for presidio into json: %w", err)
	}

	if disablePresidioRequest {
		if hardcodedPiiTypes == nil {
			return empty, nil
		}

		return *hardcodedPiiTypes, nil
	}

	req, err := http.NewRequest("POST", presidioSvcURL, bytes.NewBuffer(msgString))
	if err != nil {
		return empty, fmt.Errorf("could not create new request object: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	GlobalOtelPropagator.Inject(ctx, propagation.HeaderCarrier(req.Header))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return empty, fmt.Errorf("presidio post error: %w", err)
	}

	jsonResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return empty, fmt.Errorf("could not read Presidio response, %w", err)
	}

	err = resp.Body.Close()
	if err != nil {
		return empty, fmt.Errorf("could not close presidio response body, %w", err)
	}

	var unmarshalledData []string

	err = json.Unmarshal(jsonResp, &unmarshalledData)
	if err != nil {
		return empty, fmt.Errorf("could not unmarshall response body: %w", err)
	}

	return unmarshalledData, nil
}
