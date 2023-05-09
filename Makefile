REGISTRY?=dettanym/prose
APP_VERSION?=latest
OS=linux
ARCH=amd64

default: build

build:
	GOOS=$(OS) GOARCH=$(ARCH) go build -o bin/ .
