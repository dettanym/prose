package noopconfig

import (
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"google.golang.org/protobuf/types/known/anypb"
)

type Parser struct {
	api.StreamFilterConfigParser
}

func (p Parser) Parse(any *anypb.Any, callbacks api.ConfigCallbackHandler) (interface{}, error) {
	return nil, nil
}

func (p Parser) Merge(parent interface{}, child interface{}) interface{} {
	return child
}
