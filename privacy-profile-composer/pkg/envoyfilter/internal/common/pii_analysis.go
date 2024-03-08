package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type PresidioDataFormat struct {
	JsonToAnalyze interface{} `json:"json_to_analyze"`
	DerivePurpose string      `json:"derive_purpose,omitempty"`
}

func PiiAnalysis(presidioSvcURL string, svcName string, bufferBytes interface{}) (string, error) {
	msgString, err := json.Marshal(
		PresidioDataFormat{
			JsonToAnalyze: bufferBytes,
			DerivePurpose: svcName,
		},
	)
	if err != nil {
		return "", fmt.Errorf("could not convert data for presidio into json: %w", err)
	}

	resp, err := http.Post(presidioSvcURL, "application/json", bytes.NewBuffer(msgString))
	if err != nil {
		return "", fmt.Errorf("presidio post error: %w", err)
	}

	jsonResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not read Presidio response, %w", err)
	}

	err = resp.Body.Close()
	if err != nil {
		return "", fmt.Errorf("could not close presidio response body, %w", err)
	}

	return string(jsonResp), nil
}
