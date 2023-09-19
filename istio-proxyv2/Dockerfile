FROM golang:1.21 as builder

WORKDIR /app

COPY ./privacy-profile-composer/ ./
RUN make all

FROM istio/proxyv2:latest

COPY --from=builder /app/dist/simple.so /etc/envoy/simple.so
