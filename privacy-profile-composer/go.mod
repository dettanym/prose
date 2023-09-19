module privacy-profile-composer

go 1.18

require (
	github.com/cncf/xds/go v0.0.0-20230607035331-e9ce68804cb4
	// Requires this commit or later: https://github.com/envoyproxy/envoy/commit/782bb0c102ffbd9880615526c54c00301a6f3f82
	github.com/envoyproxy/envoy v1.27.1-0.20230915211138-782bb0c102ff
	google.golang.org/grpc v1.58.0
	google.golang.org/protobuf v1.31.0
)

require (
	github.com/envoyproxy/protoc-gen-validate v1.0.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	golang.org/x/net v0.12.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
	golang.org/x/text v0.11.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230711160842-782d3b101e98 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230711160842-782d3b101e98 // indirect
)
