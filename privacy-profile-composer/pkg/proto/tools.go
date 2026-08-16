// Created based on:
// - https://go.dev/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
// - https://github.com/go-modules-by-example/index/blob/master/010_tools/README.md

//go:build tools
// +build tools

package proto

import (
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
)
