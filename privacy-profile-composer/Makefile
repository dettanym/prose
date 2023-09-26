build-envoy-filter: proto
	go build -o dist/simple.so -buildmode=c-shared ./cmd/envoy-filter

run-server: proto
	go run ./cmd/opa-composer-server --policy_file ./pkg/opa/policy-and-logic/policy.rego

proto:
	protoc \
		--go_out=. --go_opt=module=privacy-profile-composer \
 		--go-grpc_out=. --go-grpc_opt=module=privacy-profile-composer \
 		./pkg/proto/privacy_profiles.proto
