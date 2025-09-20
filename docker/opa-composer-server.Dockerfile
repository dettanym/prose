FROM golang:1.21@sha256:4746d26432a9117a5f58e95cb9f954ddf0de128e9d5816886514199316e4a2fb AS builder

WORKDIR /app

COPY ./privacy-profile-composer/ ./

RUN go build -o bin/ ./cmd/opa-composer-server

FROM debian:12-slim@sha256:df52e55e3361a81ac1bead266f3373ee55d29aa50cf0975d440c2be3483d8ed3

WORKDIR /app

COPY --from=builder /app/bin/opa-composer-server /app/bin/opa-composer-server
COPY --from=builder /app/resources/opa_bundle/ /app/bundle/

RUN mkdir -p /app/tmp \
    && chmod 755 /app/tmp \
    && chown nobody:nogroup /app/tmp

EXPOSE 8080
EXPOSE 50051

USER nobody:nogroup

CMD [ \
    "/app/bin/opa-composer-server", \
    "--policy_bundle_dir", "/app/bundle", \
    "--compiled_bundle", "/app/tmp/bundle.tar.gz" \
]
