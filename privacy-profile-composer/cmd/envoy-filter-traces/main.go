package main

import (
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"

	"privacy-profile-composer/pkg/envoyfilter"
	"privacy-profile-composer/pkg/envoyfilter/passthrough_buffer_traces"
)

const Name = "traces"

func init() {
	http.RegisterHttpFilterConfigFactoryAndParser(Name, passthrough_buffer_traces.FilterFactory, &envoyfilter.ConfigParser{})
}

func main() {}
