//go:build generate
// +build generate

package proto

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative ./privacy_profiles.proto

// new stuff that I'm working on (issue#109):

//go:generate go run privacy_profiles.go
