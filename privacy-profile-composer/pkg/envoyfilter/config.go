package envoyfilter

import (
	"errors"
	"fmt"

	xds "github.com/cncf/xds/go/xds/type/v3"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"google.golang.org/protobuf/types/known/anypb"

	"privacy-profile-composer/pkg/envoyfilter/internal/common"
)

type config struct {
	direction   common.SidecarDirection
	zipkinUrl   string
	opaEnforce  bool
	opaConfig   string
	presidioUrl string
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

	if val, ok := configStruct["direction"]; !ok {
		return nil, errors.New("missing direction")
	} else if str, ok := val.(string); !ok {
		return nil, fmt.Errorf("direction: expect string while got %T", str)
	} else {
		switch str {
		case "SIDECAR_INBOUND":
			conf.direction = common.Inbound
		case "SIDECAR_OUTBOUND":
			conf.direction = common.Outbound
		default:
			return nil, fmt.Errorf("direction: expected either `SIDECAR_INBOUND` or `SIDECAR_OUTBOUND`, but got `%v`", str)
		}
	}

	if zipkinUrl, ok := configStruct["zipkin_url"]; !ok {
		return nil, errors.New("missing zipkin_url")
	} else if str, ok := zipkinUrl.(string); !ok {
		return nil, fmt.Errorf("zipkin_url: expect string while got %T", zipkinUrl)
	} else {
		conf.zipkinUrl = str
	}

	// decide whether to drop requests after a violation or not
	if val, ok := configStruct["opa_enforce"]; !ok {
		conf.opaEnforce = false // by default, don't drop requests (i.e. dev mode)
	} else if opaEnforce, ok := val.(bool); !ok {
		return nil, fmt.Errorf("opa_enforce: expect bool while got %T", opaEnforce)
	} else {
		conf.opaEnforce = opaEnforce
	}

	// opa_config should be a YAML inline string,
	// following this example: https://www.openpolicyagent.org/docs/latest/configuration/#example
	if parsedStr, ok := configStruct["opa_config"]; !ok {
		return nil, errors.New("missing opa_config")
	} else if opaConfig, ok := parsedStr.(string); !ok {
		return nil, fmt.Errorf("opa_config: expect (YAML inline) string while got %T", opaConfig)
	} else {
		conf.opaConfig = opaConfig
	}

	if parsedStr, ok := configStruct["presidio_url"]; !ok {
		return nil, errors.New("missing presidio_url")
	} else if presidioUrl, ok := parsedStr.(string); !ok {
		return nil, fmt.Errorf("presidio_url: expect string while got %T", presidioUrl)
	} else {
		conf.presidioUrl = presidioUrl
	}
	return conf, nil
}

func (p *ConfigParser) Merge(parent interface{}, child interface{}) interface{} {
	parentConfig := parent.(*config)
	childConfig := child.(*config)

	// copy one, do not update parentConfig directly.
	newConfig := *parentConfig

	newConfig.direction = childConfig.direction

	if childConfig.zipkinUrl != "" {
		newConfig.zipkinUrl = childConfig.zipkinUrl
	}

	if childConfig.opaConfig != "" {
		newConfig.opaConfig = childConfig.opaConfig
	}

	if childConfig.presidioUrl != "" {
		newConfig.presidioUrl = childConfig.presidioUrl
	}

	newConfig.opaEnforce = childConfig.opaEnforce

	return &newConfig
}

func unmarshalConfig(any *anypb.Any) (map[string]interface{}, error) {
	configStruct := &xds.TypedStruct{}

	if err := any.UnmarshalTo(configStruct); err != nil {
		return nil, err
	}

	return configStruct.Value.AsMap(), nil
}
