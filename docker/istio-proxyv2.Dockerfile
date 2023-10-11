FROM golang:1.18 as builder

RUN apt-get update \
    && apt-get install -y protobuf-compiler \
    && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28 \
    && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

WORKDIR /app

COPY ./privacy-profile-composer/ ./
RUN make build-envoy-filter

FROM istio/proxyv2:1.19.1@sha256:62eba4096af83c286fc8898e77fda09efde37492bd91c16d06f3f99010539ada

ENV GODEBUG="cgocheck=0"

COPY --from=builder /app/dist/simple.so /etc/envoy/simple.so
