package main

import (
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"

	"privacy-profile-composer/pkg/envoyfilter/noopconfig"
)

const Name = "passthrough-buffer"

func init() {
	http.RegisterHttpFilterConfigFactoryAndParser(Name, passthroughFilterWithBufferFactory, noopconfig.Parser{})
}

func main() {}

func passthroughFilterWithBufferFactory(config interface{}) api.StreamFilterFactory {
	return func(callbacks api.FilterCallbackHandler) api.StreamFilter {
		return &passthroughBufferFilter{}
	}
}

type passthroughBufferFilter struct {
	api.PassThroughStreamFilter

	decodeDataBuffer string
	encodeDataBuffer string
}

func (f *passthroughBufferFilter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.StatusType {
	if !endStream {
		return api.StopAndBuffer
	}

	return api.Continue
}

func (f *passthroughBufferFilter) DecodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	f.decodeDataBuffer += buffer.String()

	if !endStream {
		return api.StopAndBuffer
	}

	return api.Continue
}

func (f *passthroughBufferFilter) DecodeTrailers(trailers api.RequestTrailerMap) api.StatusType {
	return api.Continue
}

func (f *passthroughBufferFilter) EncodeHeaders(header api.ResponseHeaderMap, endStream bool) api.StatusType {
	if !endStream {
		return api.StopAndBuffer
	}

	return api.Continue
}

func (f *passthroughBufferFilter) EncodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	f.encodeDataBuffer += buffer.String()

	if !endStream {
		return api.StopAndBuffer
	}

	return api.Continue
}

func (f *passthroughBufferFilter) EncodeTrailers(trailers api.ResponseTrailerMap) api.StatusType {
	return api.Continue
}
