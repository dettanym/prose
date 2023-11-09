package main

import (
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"

	"privacy-profile-composer/pkg/envoy_filter/config"
)

const Name = "simple"

func init() {
	http.RegisterHttpFilterConfigFactoryAndParser(Name, config.ConfigFactory, &config.Parser{})
}

func main() {}
