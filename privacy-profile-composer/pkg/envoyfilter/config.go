package envoyfilter

import (
	xds "github.com/cncf/xds/go/xds/type/v3"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"google.golang.org/protobuf/types/known/anypb"
)

type config struct {
}

type ConfigParser struct {
	api.StreamFilterConfigParser
}

func (p *ConfigParser) Parse(any *anypb.Any) (interface{}, error) {
	_, err := unmarshalConfig(any)
	if err != nil {
		return nil, err
	}

	conf := &config{}

	return conf, nil
}

func (p *ConfigParser) Merge(parent interface{}, child interface{}) interface{} {
	parentConfig := parent.(*config)
	_ = child.(*config)

	// copy one, do not update parentConfig directly.
	newConfig := *parentConfig

	return &newConfig
}

func unmarshalConfig(any *anypb.Any) (map[string]interface{}, error) {
	configStruct := &xds.TypedStruct{}

	if err := any.UnmarshalTo(configStruct); err != nil {
		return nil, err
	}

	return configStruct.Value.AsMap(), nil
}
