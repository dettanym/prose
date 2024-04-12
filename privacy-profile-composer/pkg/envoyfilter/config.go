package envoyfilter

import (
	"fmt"
	"net"

	xds "github.com/cncf/xds/go/xds/type/v3"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"google.golang.org/protobuf/types/known/anypb"

	"privacy-profile-composer/pkg/envoyfilter/internal/common"
)

type Config struct {
	direction     common.SidecarDirection
	ZipkinUrl     string
	opaEnforce    bool
	OpaConfig     string
	presidioUrl   string
	internalCidrs []net.IPNet
	purpose       string
}

type ConfigParser struct {
	api.StreamFilterConfigParser
}

func (p *ConfigParser) Parse(any *anypb.Any, callbacks api.ConfigCallbackHandler) (interface{}, error) {
	configStruct, err := unmarshalConfig(any)
	if err != nil {
		return nil, err
	}

	conf := &Config{}

	if val, ok := configStruct["direction"]; !ok {
		return nil, fmt.Errorf("missing direction")
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
		return nil, fmt.Errorf("missing zipkin_url")
	} else if str, ok := zipkinUrl.(string); !ok {
		return nil, fmt.Errorf("zipkin_url: expect string while got %T", zipkinUrl)
	} else {
		conf.ZipkinUrl = str
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
		return nil, fmt.Errorf("missing opa_config")
	} else if opaConfig, ok := parsedStr.(string); !ok {
		return nil, fmt.Errorf("opa_config: expect (YAML inline) string while got %T", opaConfig)
	} else {
		conf.OpaConfig = opaConfig
	}

	if parsedStr, ok := configStruct["presidio_url"]; !ok {
		return nil, fmt.Errorf("missing presidio_url")
	} else if presidioUrl, ok := parsedStr.(string); !ok {
		return nil, fmt.Errorf("presidio_url: expect string while got %T", presidioUrl)
	} else {
		conf.presidioUrl = presidioUrl
	}

	// Values for this field are usually cluster settings set at creation time.
	// One possible way to find these can be using these grep commands:
	// `kubectl cluster-info dump | grep -m 1 cluster-cidr` and
	// `kubectl cluster-info dump | grep -m 1 service-cluster-ip-range`
	if internalCidrsExist, ok := configStruct["internal_cidrs"]; !ok {
		return nil, fmt.Errorf("missing internal_cidrs")
	} else if internalCidrList, ok := internalCidrsExist.([]interface{}); !ok {
		return nil, fmt.Errorf("internal_cidrs: expect a list of strings while got %T", internalCidrsExist)
	} else {
		conf.internalCidrs = make([]net.IPNet, 0, len(internalCidrList))

		for i, v := range internalCidrList {
			if internalCidrStr, ok := v.(string); !ok {
				return nil, fmt.Errorf("internal_cidrs[%d]: expected a string while got %T", i, v)
			} else if _, cidr, err := net.ParseCIDR(internalCidrStr); err != nil {
				return nil, fmt.Errorf("invalid internal_cidrs[%d]: %v (%v)", i, cidr, err)
			} else {
				conf.internalCidrs = append(conf.internalCidrs, *cidr)
			}
		}
	}

	if purpose, ok := configStruct["purpose"]; !ok {
		return nil, fmt.Errorf("missing purpose")
	} else if str, ok := purpose.(string); !ok {
		return nil, fmt.Errorf("purpose: expect string while got %T", purpose)
	} else {
		conf.purpose = str
	}

	return conf, nil
}

func (p *ConfigParser) Merge(parent interface{}, child interface{}) interface{} {
	parentConfig := parent.(*Config)
	childConfig := child.(*Config)

	// copy one, do not update parentConfig directly.
	newConfig := *parentConfig

	newConfig.direction = childConfig.direction

	if childConfig.ZipkinUrl != "" {
		newConfig.ZipkinUrl = childConfig.ZipkinUrl
	}

	if childConfig.OpaConfig != "" {
		newConfig.OpaConfig = childConfig.OpaConfig
	}

	if childConfig.presidioUrl != "" {
		newConfig.presidioUrl = childConfig.presidioUrl
	}

	newConfig.opaEnforce = childConfig.opaEnforce
	newConfig.internalCidrs = childConfig.internalCidrs
	newConfig.purpose = childConfig.purpose

	return &newConfig
}

func unmarshalConfig(any *anypb.Any) (map[string]interface{}, error) {
	configStruct := &xds.TypedStruct{}

	if err := any.UnmarshalTo(configStruct); err != nil {
		return nil, err
	}

	return configStruct.Value.AsMap(), nil
}
