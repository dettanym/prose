package passthrough

import "github.com/envoyproxy/envoy/contrib/golang/common/go/api"

func FilterFactory(config interface{}) api.StreamFilterFactory {
	return func(callbacks api.FilterCallbackHandler) api.StreamFilter {
		return &api.PassThroughStreamFilter{}
	}
}
