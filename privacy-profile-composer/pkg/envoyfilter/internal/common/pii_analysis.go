package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func PiiAnalysis(presidioSvcURL string, svcName string, bufferBytes []byte) (string, error) {
	var jsonBody = `{
			"key_F": {
				"key_a1": "My phone number is 212-121-1424"
			},
			"URL": "www.abc.com",
			"key_c": 3,
			"names": ["James Bond", "Clark Kent", "Hakeem Olajuwon", "No name here!"],
			"address": "123 Alpha Beta, Waterloo ON N2L3G1, Canada",
			"DOB": "01-01-1989",
			"gender": "Female",
			"race": "Asian",
			"language": "English"
		}`

	svcNameBuf, err := json.Marshal(svcName)
	if err != nil {
		return "", fmt.Errorf("could not marshal service name string into a valid JSON string: %w", err)
	}

	// TODO replace jsonBody with bufferBytes input arg
	msgString := `{"json_to_analyze":` + jsonBody + `,"derive_purpose":` + string(svcNameBuf) + `}`

	resp, err := http.Post(presidioSvcURL, "application/json", bytes.NewBufferString(msgString))
	if err != nil {
		return "", fmt.Errorf("presidio post error: %w", err)
	}

	log.Printf("presidio responded '%v', content-length is %v bytes\n", resp.Status, resp.ContentLength)

	jsonResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not read Presidio response, %w", err)
	}

	err = resp.Body.Close()
	if err != nil {
		return "", fmt.Errorf("could not close presidio response body, %w", err)
	}

	log.Println("presidio response headers:")
	for key, value := range resp.Header {
		log.Printf("  \"%v\": %v\n", key, value)
	}
	log.Println("presidio response body:")
	log.Printf("%v\n", string(jsonResp))

	return string(jsonResp), nil
}
