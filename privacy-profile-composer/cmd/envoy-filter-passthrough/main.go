package main

import (
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"

	"privacy-profile-composer/pkg/envoyfilter"
)

const Name = "passthrough"

func init() {
	http.RegisterHttpFilterConfigFactoryAndParser(Name, PassthroughFilterConfigFactory, &envoyfilter.ConfigParser{})
}

func main() {}

func PassthroughFilterConfigFactory(config interface{}) api.StreamFilterFactory {
	return func(callbacks api.FilterCallbackHandler) api.StreamFilter {
		return &api.PassThroughStreamFilter{}
	}
}
