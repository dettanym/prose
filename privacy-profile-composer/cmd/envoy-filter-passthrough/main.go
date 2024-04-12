package main

import (
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"

	"privacy-profile-composer/pkg/envoyfilter"
	"privacy-profile-composer/pkg/envoyfilter/passthrough"
)

const Name = "passthrough"

func init() {
	http.RegisterHttpFilterConfigFactoryAndParser(Name, passthrough.FilterFactory, &envoyfilter.ConfigParser{})
}

func main() {}
