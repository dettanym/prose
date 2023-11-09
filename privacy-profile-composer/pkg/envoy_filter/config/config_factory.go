package config

import (
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"

	"privacy-profile-composer/pkg/envoy_filter/inbound"
)

func ConfigFactory(c interface{}) api.StreamFilterFactory {
	conf, ok := c.(*Config)
	if !ok {
		panic("unexpected config type")
	}

	return func(callbacks api.FilterCallbackHandler) api.StreamFilter {
		return inbound.NewFilter(callbacks, conf)
	}
}
