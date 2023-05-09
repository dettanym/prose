FROM alpine:3.14

COPY bin/main /app/main
CMD ["/app/main"]
