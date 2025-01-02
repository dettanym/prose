package main

import (
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"

	"privacy-profile-composer/pkg/envoyfilter"
)

const Name = "prose-no-presidio"

func init() {
	http.RegisterHttpFilterConfigFactoryAndParser(
		Name,
		envoyfilter.ConfigFactory,
		&envoyfilter.ConfigParser{
			DisablePresidioRequests: true,
		},
	)
}

func main() {}
