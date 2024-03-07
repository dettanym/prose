package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func PiiAnalysis(presidioSvcURL string, svcName string, bufferBytes []byte) (string, error) {
	svcNameBuf, err := json.Marshal(svcName)
	if err != nil {
		return "", fmt.Errorf("could not marshal service name string into a valid JSON string: %w", err)
	}

	msgString := fmt.Sprintf(
		`{
			"json_to_analyze": %s,
			"derive_purpose": %s
		}`,
		bufferBytes,
		svcNameBuf,
	)

	resp, err := http.Post(presidioSvcURL, "application/json", bytes.NewBufferString(msgString))
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
