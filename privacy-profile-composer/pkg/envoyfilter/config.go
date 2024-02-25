package envoyfilter

import (
	"errors"
	"fmt"

	xds "github.com/cncf/xds/go/xds/type/v3"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"google.golang.org/protobuf/types/known/anypb"
)

type config struct {
	zipkinUrl          string
	opaBundleServerUrl string // http://prose-server.prose-system.svc.cluster.local:8080
}

type ConfigParser struct {
	api.StreamFilterConfigParser
}

func (p *ConfigParser) Parse(any *anypb.Any, callbacks api.ConfigCallbackHandler) (interface{}, error) {
	configStruct, err := unmarshalConfig(any)
	if err != nil {
		return nil, err
	}

	conf := &config{}

	if zipkinUrl, ok := configStruct["zipkin_url"]; !ok {
		return nil, errors.New("missing zipkin_url")
	} else if str, ok := zipkinUrl.(string); !ok {
		return nil, fmt.Errorf("zipkin_url: expect string while got %T", zipkinUrl)
	} else {
		conf.zipkinUrl = str
	}

	if parsedStr, ok := configStruct["opa_bundle_server_url"]; !ok {
		return nil, errors.New("missing opa_bundle_server_url")
	} else if opaBundleServerUrl, ok := parsedStr.(string); !ok {
		return nil, fmt.Errorf("opa_bundle_server_url: expect string while got %T", opaBundleServerUrl)
	} else {
		conf.opaBundleServerUrl = opaBundleServerUrl
	}
	return conf, nil
}

func (p *ConfigParser) Merge(parent interface{}, child interface{}) interface{} {
	parentConfig := parent.(*config)
	childConfig := child.(*config)

	// copy one, do not update parentConfig directly.
	newConfig := *parentConfig

	if childConfig.zipkinUrl != "" {
		newConfig.zipkinUrl = childConfig.zipkinUrl
	}

	if childConfig.opaBundleServerUrl != "" {
		newConfig.opaBundleServerUrl = childConfig.opaBundleServerUrl
	}

	return &newConfig
}

func unmarshalConfig(any *anypb.Any) (map[string]interface{}, error) {
	configStruct := &xds.TypedStruct{}

	if err := any.UnmarshalTo(configStruct); err != nil {
		return nil, err
	}

	return configStruct.Value.AsMap(), nil
}
