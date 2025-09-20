# Conservatively, this version should match the version in `go.mod` file.
FROM golang:1.20@sha256:8f9af7094d0cb27cc783c697ac5ba25efdc4da35f8526db21f7aebb0b0b4f18a AS builder

WORKDIR /app

COPY ./privacy-profile-composer/go.mod ./privacy-profile-composer/go.sum ./
RUN go mod download

COPY ./privacy-profile-composer/ ./
RUN go build -o dist/prose.so -buildmode=c-shared ./cmd/prose-filter
RUN go build -o dist/passthrough.so -buildmode=c-shared ./cmd/passthrough-filter
RUN go build -o dist/tooling.so -buildmode=c-shared ./cmd/tooling-filter
RUN go build -o dist/prose-no-presidio.so -buildmode=c-shared ./cmd/prose-no-presidio-filter

# This version needs to match the deployed istiod helmrelease version
FROM istio/proxyv2:1.20.3@sha256:18163bd4fdb641bdff1489e124a0b9f1059bb2cec9c8229161b73517db97c05a

ENV GODEBUG="cgocheck=0"

COPY --from=builder /app/dist/ /etc/envoy/
