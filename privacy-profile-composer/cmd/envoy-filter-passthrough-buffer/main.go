package main

import (
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"

	"privacy-profile-composer/pkg/envoyfilter"
	"privacy-profile-composer/pkg/envoyfilter/passthrough_buffer"
)

const Name = "passthrough-buffer"

func init() {
	http.RegisterHttpFilterConfigFactoryAndParser(Name, passthrough_buffer.FilterFactory, &envoyfilter.ConfigParser{})
}

func main() {}
