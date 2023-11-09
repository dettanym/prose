package main

import (
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"

	"privacy-profile-composer/pkg/envoy_filter/config"
)

const Name = "simple"

func init() {
	http.RegisterHttpFilterConfigFactoryAndParser(Name, ConfigFactory, &config.Parser{})
}

func main() {}

func ConfigFactory(c interface{}) api.StreamFilterFactory {
	conf, ok := c.(*config.Config)
	if !ok {
		panic("unexpected config type")
	}

	return func(callbacks api.FilterCallbackHandler) api.StreamFilter {
		return &filter{
			callbacks: callbacks,
			config:    conf,
		}
	}
}
