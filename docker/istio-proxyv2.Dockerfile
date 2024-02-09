FROM golang:1.18 as builder

RUN apt-get update \
    && apt-get install -y protobuf-compiler \
    && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28 \
    && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

WORKDIR /app

COPY ./privacy-profile-composer/ ./
RUN make build-envoy-filter

FROM istio/proxyv2:1.20.3@sha256:18163bd4fdb641bdff1489e124a0b9f1059bb2cec9c8229161b73517db97c05a

ENV GODEBUG="cgocheck=0"

COPY --from=builder /app/dist/simple.so /etc/envoy/simple.so
