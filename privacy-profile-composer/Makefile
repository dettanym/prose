all:
	cd proto && protoc --go_out=. --go_opt=paths=source_relative  --go-grpc_out=. --go-grpc_opt=paths=source_relative ./privacy_profiles.proto
	go build -C envoy-filter -o simple -buildmode=c-shared .
	go build -C grpc_test .