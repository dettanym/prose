FROM golang:1.21 as builder

WORKDIR /app

COPY ./privacy-profile-composer/ ./

RUN go build -o bin/ ./cmd/opa-composer-server

FROM debian:12-slim

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
