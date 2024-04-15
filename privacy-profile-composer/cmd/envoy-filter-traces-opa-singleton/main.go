package main

import (
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"

	"privacy-profile-composer/pkg/envoyfilter"
	"privacy-profile-composer/pkg/envoyfilter/passthrough_buffer_traces_opa_singleton"
)

const Name = "traces-opa-singleton"

func init() {
	http.RegisterHttpFilterConfigFactoryAndParser(
		Name,
		passthrough_buffer_traces_opa_singleton.FilterFactory,
		&envoyfilter.ConfigParser{},
	)
}

func main() {}
