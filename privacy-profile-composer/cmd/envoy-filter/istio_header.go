package main

import (
	"encoding/base64"
	"fmt"
	"github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"strings"
)

// X-Envoy-Peer-Metadata header is set by Istio.
// It's a base64 encoded value of what looks like a Protobuf struct.
// We want to transform this header value, from base64 to a native Golang struct.

// The headers of this struct are upper-case versions of the Flatbuffers file at
// https://github.com/istio/proxy/blob/ab736e6f1c52bf35896523151d2f068edac0253e/extensions/common/node_info.fbs
// The proto_util.cc file in that folder
// https://github.com/istio/proxy/blob/ab736e6f1c52bf35896523151d2f068edac0253e/extensions/common/proto_util.cc
// parses a dynamic Protobuf structure (google::protobuf::Struct in C++) to a Flatbuffers structure.
// In Go, this is structpb: it's used to unmarshal arbitrary JSON into PB.
// So Istio is probably using google::protobuf::Struct to transform arbitrary JSON to PB
// which may be transformed to Flatbuffers for internal use as in proto_util.cc
// The extractNodeFlatBufferFromStruct method manually iterates over all keys in the input
// struct and transforms them into values.
// We similarly iterate through keys in the generic PB struct (internally uses PB reflection)
// in our function parseXEnvoyPeerMetadataToGolangStruct, in order to fill in our native Golang struct

type XEnvoyPeerMetadataHeader struct {
	Name          string            `json:"NAME"`
	Namespace     string            `json:"NAMESPACE"`
	Labels        map[string]string `json:"LABELS"`
	AppContainers []string          `json:"APP_CONTAINERS"`
}

func DecodeXEnvoyPeerMetadataHeader(base64encoded string) (XEnvoyPeerMetadataHeader, error) {
	istioHeader := XEnvoyPeerMetadataHeader{
		Name:          "",
		Namespace:     "",
		Labels:        map[string]string{},
		AppContainers: []string{},
	}

	decodedBinary, err := base64.StdEncoding.DecodeString(base64encoded)
	if err != nil {
		fmt.Println("decode error:", err)
		return istioHeader, err
	}

	genericPBstruct, err := structpb.NewStruct(map[string]interface{}{})
	if err != nil {
		fmt.Println("cannot initialize a generic PB struct: ", err)
		return istioHeader, err
	}

	err = proto.Unmarshal(decodedBinary, genericPBstruct)
	if err != nil {
		fmt.Println("cannot unmarshal decoded binary into generic PB struct:", err)
		return istioHeader, err
	}

	istioHeader = parseXEnvoyPeerMetadataToGolangStruct(genericPBstruct)

	return istioHeader, nil
}

func parseXEnvoyPeerMetadataToGolangStruct(metadata *structpb.Struct) XEnvoyPeerMetadataHeader {
	istioHeader := XEnvoyPeerMetadataHeader{
		Name:          "",
		Namespace:     "",
		Labels:        map[string]string{},
		AppContainers: []string{},
	}

	for key, value := range metadata.GetFields() {
		switch key {
		case "NAME":
			istioHeader.Name = value.GetStringValue()
		case "NAMESPACE":
			istioHeader.Namespace = value.GetStringValue()
		case "LABELS":
			for labelKey, labelValue := range value.GetStructValue().GetFields() {
				istioHeader.Labels[labelKey] = labelValue.GetStringValue()
			}
		case "APP_CONTAINERS":
			istioHeader.AppContainers = strings.Split(value.GetStringValue(), ",")
		}
	}

	return istioHeader
}

func PrintXEnvoyPeerMetadataHeader(header XEnvoyPeerMetadataHeader) {
	fmt.Println("--> Pod Namespace: ", header.Namespace)
	fmt.Println("--> Pod Name: ", header.Name)
	fmt.Println("--> Pod labels: ")
	for key, value := range header.Labels {
		fmt.Printf("---> Label \t key: %s \t Value: %s\n", key, value)
	}
	fmt.Println("--> Pod containers: ")
	for _, container := range header.AppContainers {
		fmt.Printf("---> Container: %s\n", container)
	}
}
