package main

import (
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"

	"privacy-profile-composer/pkg/envoyfilter"
	"privacy-profile-composer/pkg/envoyfilter/passthrough_filter"
)

const Name = "passthrough"

func init() {
	http.RegisterHttpFilterConfigFactoryAndParser(Name, passthrough_filter.FilterFactory, &envoyfilter.ConfigParser{})
}

func main() {}
