package envoyfilter

import (
	"log"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"

	"privacy-profile-composer/pkg/envoyfilter/internal/common"
)

func ConfigFactory(c interface{}) api.StreamFilterFactory {
	conf, ok := c.(*Config)
	if !ok {
		panic("unexpected config type")
	}

	_, err := common.UpdateTracer(conf.ZipkinUrl)
	if err != nil {
		panic(err)
	}

	_, err = common.UpdateAuthAgent(conf.OpaConfig)
	if err != nil {
		log.Printf(
			"could not initialize an OPA object --- "+
				"this means that the data plane cannot evaluate the target privacy policy ----- "+
				"%+v",
			err,
		)
		panic(err)
	}

	return func(callbacks api.FilterCallbackHandler) api.StreamFilter {
		filter, err := NewFilter(callbacks, conf)
		if err != nil {
			panic(err)
		}
		return filter
	}
}
