all:
	protoc \
		--go_out=. --go_opt=module=privacy-profile-composer \
 		--go-grpc_out=. --go-grpc_opt=module=privacy-profile-composer \
 		./pkg/proto/privacy_profiles.proto
	go build -o dist/simple.so -buildmode=c-shared ./cmd/envoy-filter
	go build -o dist/grpc_test ./cmd/grpc_test

run:
	go run ./cmd/grpc_test
