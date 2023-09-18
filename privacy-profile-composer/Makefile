all:
	cd proto && protoc --go_out=. --go_opt=paths=source_relative  --go-grpc_out=. --go-grpc_opt=paths=source_relative ./privacy_profiles.proto
	go build -o dist/simple.so -buildmode=c-shared ./envoy-filter
	go build -o dist/grpc_test ./grpc_test