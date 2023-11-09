package envoyfilter

import (
	"errors"
	"fmt"

	xds "github.com/cncf/xds/go/xds/type/v3"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"google.golang.org/protobuf/types/known/anypb"
)

type config struct {
	echoBody string
}

type ConfigParser struct {
	api.StreamFilterConfigParser
}

func (p *ConfigParser) Parse(any *anypb.Any) (interface{}, error) {
	configStruct := &xds.TypedStruct{}
	if err := any.UnmarshalTo(configStruct); err != nil {
		return nil, err
	}

	conf := &config{}

	v := configStruct.Value
	prefix, ok := v.AsMap()["prefix_localreply_body"]
	if !ok {
		return nil, errors.New("missing prefix_localreply_body")
	}

	if str, ok := prefix.(string); ok {
		conf.echoBody = str
	} else {
		return nil, fmt.Errorf("prefix_localreply_body: expect string while got %T", prefix)
	}

	return conf, nil
}

func (p *ConfigParser) Merge(parent interface{}, child interface{}) interface{} {
	parentConfig := parent.(*config)
	childConfig := child.(*config)

	// copy one, do not update parentConfig directly.
	newConfig := *parentConfig

	if childConfig.echoBody != "" {
		newConfig.echoBody = childConfig.echoBody
	}

	return &newConfig
}
