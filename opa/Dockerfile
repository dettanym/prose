FROM golang:1.20 as builder

WORKDIR /app
# Copy the Go Modules manifests
COPY go.mod ./
COPY go.sum ./
COPY policy-and-logic/ ./
# Copy the go source
COPY main.go ./

RUN go build -o bin/ .
RUN cp ./bin/main /app/main
CMD ["/app/main"]
