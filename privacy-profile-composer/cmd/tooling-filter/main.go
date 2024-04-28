package main

import (
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"

	"privacy-profile-composer/pkg/envoyfilter"
	"privacy-profile-composer/pkg/envoyfilter/tooling_filter"
)

const Name = "tooling"

func init() {
	http.RegisterHttpFilterConfigFactoryAndParser(
		Name,
		tooling_filter.FilterFactory,
		&envoyfilter.ConfigParser{},
	)
}

func main() {}
