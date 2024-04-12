package passthrough_buffer

import "github.com/envoyproxy/envoy/contrib/golang/common/go/api"

func FilterFactory(config interface{}) api.StreamFilterFactory {
	return func(callbacks api.FilterCallbackHandler) api.StreamFilter {
		return &filter{}
	}
}

type filter struct {
	api.PassThroughStreamFilter

	decodeDataBuffer string
	encodeDataBuffer string
}

func (f *filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.StatusType {
	if !endStream {
		return api.StopAndBuffer
	}

	return api.Continue
}

func (f *filter) DecodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	f.decodeDataBuffer += buffer.String()

	if !endStream {
		return api.StopAndBuffer
	}

	return api.Continue
}

func (f *filter) DecodeTrailers(trailers api.RequestTrailerMap) api.StatusType {
	return api.Continue
}

func (f *filter) EncodeHeaders(header api.ResponseHeaderMap, endStream bool) api.StatusType {
	if !endStream {
		return api.StopAndBuffer
	}

	return api.Continue
}

func (f *filter) EncodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	f.encodeDataBuffer += buffer.String()

	if !endStream {
		return api.StopAndBuffer
	}

	return api.Continue
}

func (f *filter) EncodeTrailers(trailers api.ResponseTrailerMap) api.StatusType {
	return api.Continue
}
