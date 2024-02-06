package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/proto-gen/api_v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Run `docker compose up -d` to start services before this program
func main() {
	var err error

	// run some workloads so traces are created
	_, err = http.Get("http://localhost:8080/dispatch?customer=123")
	if err != nil {
		fmt.Printf("error querying hotrod app:\n%v\n", err)
		return
	}

	_, err = http.Get("http://localhost:8080/dispatch?customer=392")
	if err != nil {
		fmt.Printf("error querying hotrod app:\n%v\n", err)
		return
	}

	_, err = http.Get("http://localhost:8080/dispatch?customer=731")
	if err != nil {
		fmt.Printf("error querying hotrod app:\n%v\n", err)
		return
	}

	_, err = http.Get("http://localhost:8080/dispatch?customer=567")
	if err != nil {
		fmt.Printf("error querying hotrod app:\n%v\n", err)
		return
	}

	// setup grpc client and query jaeger

	grpcCC, err := grpc.Dial(
		"localhost:16685",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		fmt.Printf("error dialing grpc server:\n%v\n", err)
		return
	}

	defer func() {
		err := grpcCC.Close()
		if err != nil {
			fmt.Printf("error closing grpc client connection:\n%v\n", err)
		}
	}()

	// for possible operations see:
	// https://github.com/jaegertracing/jaeger-idl/blob/main/proto/api_v2/query.proto
	// https://github.com/jaegertracing/jaeger/blob/main/proto-gen/api_v2/query.pb.go
	jaegerQueryClient := api_v2.NewQueryServiceClient(grpcCC)

	services, err := jaegerQueryClient.GetServices(
		context.Background(),
		&api_v2.GetServicesRequest{},
	)

	if err != nil {
		fmt.Printf("error loading services:\n%v\n", err)
		return
	} else {
		fmt.Printf("loaded services:\n%v\n", services.Services)
	}

	findTracesClient, err := jaegerQueryClient.FindTraces(
		context.Background(),
		&api_v2.FindTracesRequest{
			Query: &api_v2.TraceQueryParameters{
				// ServiceName seems to be required
				ServiceName:   "mysql",
				OperationName: "SQL SELECT",
				Tags: map[string]string{
					"span.kind": "client",
				},
			},
		},
	)
	if err != nil {
		fmt.Printf("error finding traces:\n%v\n", err)
		return
	}

	spansResponse, err := findTracesClient.Recv()
	if err != nil {
		fmt.Printf("error receiving the response:\n%v\n", err)
		return
	}

	spans := spansResponse.GetSpans()

	if len(spans) == 0 {
		fmt.Printf("successfully queried. no spans found")
	} else {
		fmt.Printf("successfully queried. found %d spans\n", len(spans))
	}

	traceids := []model.TraceID{}
	traceidsSeen := map[model.TraceID]bool{}
	for _, s := range spansResponse.GetSpans() {
		if !traceidsSeen[s.TraceID] {
			traceids = append(traceids, s.TraceID)
		}
		traceidsSeen[s.TraceID] = true
	}

	fmt.Printf("found relevant traces:\n%v\n", traceids)
}
