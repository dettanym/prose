# Conservatively, this version should match the version in `go.mod` file.
FROM golang:1.20@sha256:8f9af7094d0cb27cc783c697ac5ba25efdc4da35f8526db21f7aebb0b0b4f18a as builder

WORKDIR /app

COPY ./privacy-profile-composer/go.mod ./privacy-profile-composer/go.sum ./
RUN go mod download

COPY ./privacy-profile-composer/ ./
RUN go build -o dist/simple.so -buildmode=c-shared ./cmd/envoy-filter

# This version needs to match the deployed istiod helmrelease version
FROM istio/proxyv2:1.20.3@sha256:f4e94588a14eee4f053a80a767128ffc482a219f6c6e23039b7db1b6a6081a77

ENV GODEBUG="cgocheck=0"

COPY --from=builder /app/dist/simple.so /etc/envoy/simple.so
