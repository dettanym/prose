FROM golang:1.20 as builder

RUN apt-get update \
    && apt-get install -y protobuf-compiler \
    && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.32.0 \
    && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

WORKDIR /app

COPY ./privacy-profile-composer/ ./
RUN make build-envoy-filter

FROM istio/proxyv2:1.20.3@sha256:f4e94588a14eee4f053a80a767128ffc482a219f6c6e23039b7db1b6a6081a77

ENV GODEBUG="cgocheck=0"

COPY --from=builder /app/dist/simple.so /etc/envoy/simple.so
