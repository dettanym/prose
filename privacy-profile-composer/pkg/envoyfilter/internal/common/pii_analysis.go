package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type PresidioDataFormat struct {
	JsonToAnalyze interface{} `json:"json_to_analyze"`
	DerivePurpose string      `json:"derive_purpose,omitempty"`
}

func PiiAnalysis(ctx context.Context, presidioSvcURL string, svcName string, bufferBytes interface{}) ([]string, error) {
	span, ctx := GlobalTracer.StartSpanFromContext(ctx, "PiiAnalysis")
	defer span.Finish()

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

	resp, err := http.Post(presidioSvcURL, "application/json", bytes.NewBuffer(msgString))
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
