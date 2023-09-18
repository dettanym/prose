all:
	cd pkg/proto && protoc --go_out=. --go_opt=paths=source_relative  --go-grpc_out=. --go-grpc_opt=paths=source_relative ./privacy_profiles.proto
	go build -o dist/simple.so -buildmode=c-shared ./cmd/envoy-filter
	go build -o dist/grpc_test ./cmd/grpc_test