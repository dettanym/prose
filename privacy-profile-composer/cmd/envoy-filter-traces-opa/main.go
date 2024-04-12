package main

import (
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"

	"privacy-profile-composer/pkg/envoyfilter"
	"privacy-profile-composer/pkg/envoyfilter/passthrough_buffer_traces_opa"
)

const Name = "traces-opa"

func init() {
	http.RegisterHttpFilterConfigFactoryAndParser(
		Name,
		passthrough_buffer_traces_opa.FilterFactory,
		&envoyfilter.ConfigParser{},
	)
}

func main() {}
