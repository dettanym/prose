FROM golang:1.18 as builder

RUN apt-get update \
    && apt-get install -y protobuf-compiler \
    && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28 \
    && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

WORKDIR /app

COPY ./privacy-profile-composer/ ./
RUN make build-envoy-filter

FROM istio/proxyv2:1.19.1@sha256:a521ab45c2a8f9356ee5b47790ad575968d663b705c194ad3053a4fca87dcce9

ENV GODEBUG="cgocheck=0"

COPY --from=builder /app/dist/simple.so /etc/envoy/simple.so
