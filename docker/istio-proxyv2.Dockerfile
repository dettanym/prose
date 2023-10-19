FROM golang:1.18 as builder

RUN apt-get update \
    && apt-get install -y protobuf-compiler \
    && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28 \
    && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

WORKDIR /app

COPY ./privacy-profile-composer/ ./
RUN make build-envoy-filter

FROM istio/proxyv2:1.19.3@sha256:47b08c6d59b97e69c43387df40e6b969a2c1979af29665e85698c178fceda0a4

ENV GODEBUG="cgocheck=0"

COPY --from=builder /app/dist/simple.so /etc/envoy/simple.so
