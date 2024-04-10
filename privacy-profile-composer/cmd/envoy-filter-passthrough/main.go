package main

import (
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"
	"google.golang.org/protobuf/types/known/anypb"
)

const Name = "passthrough"

func init() {
	http.RegisterHttpFilterConfigFactoryAndParser(Name, PassthroughFilterConfigFactory, NoopConfigParser{})
}

func main() {}

func PassthroughFilterConfigFactory(config interface{}) api.StreamFilterFactory {
	return func(callbacks api.FilterCallbackHandler) api.StreamFilter {
		return &api.PassThroughStreamFilter{}
	}
}

type NoopConfigParser struct {
	api.StreamFilterConfigParser
}

func (p NoopConfigParser) Parse(any *anypb.Any, callbacks api.ConfigCallbackHandler) (interface{}, error) {
	return nil, nil
}

func (p NoopConfigParser) Merge(parent interface{}, child interface{}) interface{} {
	return child
}
