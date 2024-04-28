# Conservatively, this version should match the version in `go.mod` file.
FROM golang:1.20 as builder

WORKDIR /app

COPY ./privacy-profile-composer/go.mod ./privacy-profile-composer/go.sum ./
RUN go mod download

COPY ./privacy-profile-composer/ ./
RUN go build -o dist/prose.so -buildmode=c-shared ./cmd/prose-filter
RUN go build -o dist/passthrough.so -buildmode=c-shared ./cmd/passthrough-filter
RUN go build -o dist/tooling.so -buildmode=c-shared ./cmd/tooling-filter

# This version needs to match the deployed istiod helmrelease version
FROM istio/proxyv2:1.20.3@sha256:f4e94588a14eee4f053a80a767128ffc482a219f6c6e23039b7db1b6a6081a77

ENV GODEBUG="cgocheck=0"

COPY --from=builder /app/dist/ /etc/envoy/
