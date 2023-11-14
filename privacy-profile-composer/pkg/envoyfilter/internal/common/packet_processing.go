package common

import (
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"log"
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
