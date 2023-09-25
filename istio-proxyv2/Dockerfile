FROM golang:1.21 as builder

RUN apt-get update \
    && apt-get install -y protobuf-compiler \
    && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28 \
    && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

WORKDIR /app

COPY ./privacy-profile-composer/ ./
RUN make all

FROM istio/proxyv2:1.18.3

COPY --from=builder /app/dist/simple.so /etc/envoy/simple.so
