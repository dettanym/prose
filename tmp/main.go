package main

import (
	"context"
	"fmt"
	"github.com/jaegertracing/jaeger/proto-gen/api_v2"

	"github.com/jaegertracing/jaeger/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Run `docker compose up -d` to start services before this program
func main() {
	//var err error
	//
	//// run some workloads so traces are created
	//_, err = http.Get("http://localhost:8080/dispatch?customer=123")
	//if err != nil {
	//	fmt.Printf("error querying hotrod app:\n%v\n", err)
	//	return
	//}
	//
	//_, err = http.Get("http://localhost:8080/dispatch?customer=392")
	//if err != nil {
	//	fmt.Printf("error querying hotrod app:\n%v\n", err)
	//	return
	//}
	//
	//_, err = http.Get("http://localhost:8080/dispatch?customer=731")
	//if err != nil {
	//	fmt.Printf("error querying hotrod app:\n%v\n", err)
	//	return
	//}
	//
	//_, err = http.Get("http://localhost:8080/dispatch?customer=567")
	//if err != nil {
	//	fmt.Printf("error querying hotrod app:\n%v\n", err)
	//	return
	//}

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
				ServiceName: "golang-filter",
				//OperationName: "SQL SELECT",
				//Tags: map[string]string{
				//	"span.kind": "client",
				//},
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
	// spans := spansResponse.GetResourceSpans()

	if len(spans) == 0 {
		fmt.Printf("successfully queried. no spans found")
	} else {
		fmt.Printf("successfully queried. found %d spans\n", len(spans))
	}

	var traceids []string
	traceidsSeen := map[string]bool{}
	for _, s := range spansResponse.GetSpans() {
		traceid := s.TraceID.String()
		fmt.Printf("\ntrace id: %s", traceid)
		if !traceidsSeen[traceid] {
			traceids = append(traceids, traceid)
		}
		traceidsSeen[traceid] = true
	}
	//for _, s := range spansResponse.GetResourceSpans() {
	//	for _, x := range s.ScopeSpans {
	//		y := x.GetSpans()
	//		for _, r := range y {
	//			if !traceidsSeen[] {
	//				traceids = append(traceids)
	//			}
	//			traceidsSeen[r.TraceID] = true
	//		}
	//	}
	//}

	fmt.Printf("\nfound relevant traces:\n%v\n", traceids)

	//for i, t := range traceids {
	t := traceids[0]
	tName := t
	traceid, err := model.TraceIDFromString(t)
	if err != nil {
		return
	}

	fmt.Printf("selected trace '%v' with name '%s'\n", t, tName)

	entireTraceClient, err := jaegerQueryClient.GetTrace(
		context.Background(),
		&api_v2.GetTraceRequest{
			TraceID: traceid,
		},
	)
	if err != nil {
		fmt.Printf("error receiving details about the trace id \"%s\":\n%v\n", tName, err)
		return
	}

	entiretrace, err := entireTraceClient.Recv()
	if err != nil {
		fmt.Printf("error receiving the response:\n%v\n", err)
		return
	}

	allSpansInTrace := entiretrace.GetSpans()
	if len(allSpansInTrace) == 0 {
		fmt.Printf("no spans inside trace '%s'", tName)
	}
	for _, s := range allSpansInTrace {
		fmt.Printf("received spans inside a trace '%s':\n%v\n", tName, s)
	}
	//}

}
