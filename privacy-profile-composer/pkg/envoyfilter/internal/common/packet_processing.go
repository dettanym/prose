package common

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

const anyPurpose = "ANY"
const unknownSvcName = "UNKNOWN SVC"

type RequestHeaderMetadata struct {
	Host   string
	Method string
	Path   string

	ContentType       *string
	ContentLength     *string
	EnvoyPeerMetadata *XEnvoyPeerMetadataHeader

	SvcName string
	Purpose string
}

type ResponseHeaderMetadata struct {
	ContentType   *string
	ContentLength *string
}

type SidecarDirection string

const (
	Inbound  SidecarDirection = "Inbound"
	Outbound SidecarDirection = "Outbound"
)

func LogDecodeHeaderData(header api.RequestHeaderMap) {
	log.Printf("%v (%v) %v://%v%v\n", header.Method(), header.Protocol(), header.Scheme(), header.Host(), header.Path())
	//header.Range(func(key, value string) bool {
	//	log.Printf("  \"%v\": %v\n", key, value)
	//	return true
	//})
}

func LogEncodeHeaderData(header api.ResponseHeaderMap) {
	status, statusWasSet := header.Status()
	log.Printf("Status was set (%v) to %v with response headers:\n", statusWasSet, status)
	//header.Range(func(key, value string) bool {
	//	log.Printf("  \"%v\": %v\n", key, value)
	//	return true
	//})
}

func ExtractRequestHeaderData(header api.RequestHeaderMap) RequestHeaderMetadata {
	metadata := RequestHeaderMetadata{
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

func ExtractResponseHeaderData(headers api.ResponseHeaderMap) ResponseHeaderMetadata {
	metadata := ResponseHeaderMetadata{}

	contentType, exists := headers.Get("content-type")
	if exists {
		metadata.ContentType = &contentType
	}

	contentLength, exists := headers.Get("content-length")
	if exists {
		metadata.ContentLength = &contentLength
	}

	return metadata
}

func GetDirection(callbacks api.FilterCallbackHandler) (SidecarDirection, error) {
	directionEnum, err := callbacks.GetProperty("xds.listener_direction")
	if err != nil {
		return "", fmt.Errorf("cannot determine sidecar direction as there is no xds.listener_direction key")
	}
	directionInt, err := strconv.Atoi(directionEnum)
	if err != nil {
		// check https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/core/v3/base.proto#envoy-v3-api-enum-config-core-v3-trafficdirection
		return "", fmt.Errorf("envoy's xds.listener_direction key does not contain an integer " +
			"check the Envoy docs for the range of values for this key")
	}

	if directionInt == 0 {
		return "", fmt.Errorf("envoy's xds.listener_direction key indicates that this sidecar is deployed as a gateway." +
			"Prose does not need to be run in a gateway sidecar." +
			"It will continue to get deployed in other sidecars that are configured as inbound or outbound sidecars")
	}

	if directionInt == 1 {
		return Inbound, nil
	}
	if directionInt == 2 {
		return Outbound, nil
	}

	return "", fmt.Errorf("envoy's xds.listener_direction key contains an unsupported value for the direction enum: %d "+
		"check the Envoy docs for the range of values for this key", directionInt)
}

func GetJSONBody(ctx context.Context, contentType *string, body string) (interface{}, error) {
	span, ctx := GlobalTracer.StartSpanFromContext(ctx, "getJSONBody")
	defer span.Finish()

	if contentType == nil {
		return nil, fmt.Errorf("cannot analyze body, since 'ContentType' header is not set")
	}

	switch *contentType {
	case "application/json":
		var data interface{}

		err := json.Unmarshal([]byte(body), data)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal json body: %w", err)
		}

		return data, nil
	case "application/x-www-form-urlencoded":
		query, err := url.ParseQuery(body)
		if err != nil {
			return nil, fmt.Errorf("failed to decode urlencoded form: %w", err)
		}

		return query, nil
	default:
		return nil, fmt.Errorf("cannot analyze a body with ContentType '%s'", *contentType)
	}
}
