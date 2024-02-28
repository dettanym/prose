package common

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

const anyPurpose = "ANY"
const unknownSvcName = "UNKNOWN SVC"

type HeaderMetadata struct {
	Host   string
	Method string
	Path   string

	ContentType       *string
	ContentLength     *string
	EnvoyPeerMetadata *XEnvoyPeerMetadataHeader

	SvcName string
	Purpose string
}

type SidecarDirection int

const (
	Inbound SidecarDirection = iota
	Outbound
)

func LogDecodeHeaderData(header api.RequestHeaderMap) {
	log.Printf("%v (%v) %v://%v%v\n", header.Method(), header.Protocol(), header.Scheme(), header.Host(), header.Path())
	header.Range(func(key, value string) bool {
		log.Printf("  \"%v\": %v\n", key, value)
		return true
	})
}

func LogEncodeHeaderData(header api.ResponseHeaderMap) {
	status, statusWasSet := header.Status()
	log.Printf("Status was set (%v) to %v with response headers:\n", statusWasSet, status)
	header.Range(func(key, value string) bool {
		log.Printf("  \"%v\": %v\n", key, value)
		return true
	})
}

func ExtractHeaderData(header api.RequestHeaderMap) HeaderMetadata {
	metadata := HeaderMetadata{
		Host:   header.Host(),
		Method: header.Method(),
		Path:   header.Path(),

		SvcName: unknownSvcName,
		Purpose: anyPurpose,
	}

	contentType, exists := header.Get("content-type")
	if exists {
		metadata.ContentType = &contentType
	}

	contentLength, exists := header.Get("content-length")
	if exists {
		metadata.ContentLength = &contentLength
	}

	xEnvoyPeerMetadata, exists := header.Get("x-envoy-peer-metadata")
	if exists {
		parsedHeader, err := DecodeXEnvoyPeerMetadataHeader(xEnvoyPeerMetadata)
		if err != nil {
			log.Printf("Error decoding x-envoy-peer-metadata header: %s", err)
		} else {
			metadata.EnvoyPeerMetadata = &parsedHeader
		}
	}

	if metadata.EnvoyPeerMetadata != nil {
		name := metadata.EnvoyPeerMetadata.Name
		metadata.SvcName = name

		// The pod hasn't been labelled with a purpose
		// Initialize the purpose to the svcName
		// Infer it from the svcName using presidio
		metadata.Purpose = name

		labels := metadata.EnvoyPeerMetadata.Labels
		if purpose, purposeExists := labels["purpose"]; purposeExists {
			metadata.Purpose = purpose
		}
	}

	return metadata
}

func GetDirection(callbacks api.FilterCallbackHandler) (SidecarDirection, error) {
	directionEnum, err := callbacks.GetProperty("xds.listener_direction")
	if err != nil {
		return -1, fmt.Errorf("cannot determine sidecar direction as there is no xds.listener_direction key")
	}
	directionInt, err := strconv.Atoi(directionEnum)
	if err != nil {
		// check https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/core/v3/base.proto#envoy-v3-api-enum-config-core-v3-trafficdirection
		return -1, fmt.Errorf("envoy's xds.listener_direction key does not contain an integer " +
			"check the Envoy docs for the range of values for this key")
	}

	if directionInt == 0 {
		return -1, fmt.Errorf("envoy's xds.listener_direction key indicates that this sidecar is deployed as a gateway." +
			"Prose does not need to be run in a gateway sidecar." +
			"It will continue to get deployed in other sidecars that are configured as inbound or outbound sidecars")
	}

	if directionInt == 1 {
		return Inbound, nil
	}
	if directionInt == 2 {
		return Outbound, nil
	}

	return -1, fmt.Errorf("envoy's xds.listener_direction key contains an unsupported value for the direction enum: %d "+
		"check the Envoy docs for the range of values for this key", directionInt)
}

func GetJSONBody(headerMetadata HeaderMetadata, buffer api.BufferInstance) ([]byte, error) {
	var jsonBody []byte

	if headerMetadata.ContentType == nil {
		return nil, fmt.Errorf("ContentType header is not set. Cannot analyze body")
	} else if *headerMetadata.ContentType == "application/x-www-form-urlencoded" {
		query, err := url.ParseQuery(buffer.String())
		if err != nil {
			return nil, fmt.Errorf("Failed to start decoding JSON data")
		}
		log.Println("  <<decoded x-www-form-urlencoded data: ", query)
		jsonBody, err = json.Marshal(query)
		if err != nil {
			return nil, fmt.Errorf("Could not transform URL encoded data to JSON to pass to Presidio")
		}
	} else if *headerMetadata.ContentType == "application/json" {
		jsonBody = buffer.Bytes()
	} else {
		return nil, fmt.Errorf("Cannot analyze a body with contentType '%s'\n", *headerMetadata.ContentType)
	}
	return jsonBody, nil
}
