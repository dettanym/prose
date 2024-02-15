package main

import (
	"context"
	"fmt"
	"github.com/gogo/protobuf/types"
	"github.com/jaegertracing/jaeger/model/v2"
	"github.com/jaegertracing/jaeger/proto-gen/api_v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"strings"
	"time"
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
	// https://github.com/jaegertracing/jaeger-idl/blob/main/proto/api_v3/query.proto
	// https://github.com/jaegertracing/jaeger/blob/main/proto-gen/api_v3/query.pb.go
	jaegerQueryClient := api_v3.NewQueryServiceClient(grpcCC)

	services, err := jaegerQueryClient.GetServices(
		context.Background(),
		&api_v3.GetServicesRequest{},
	)

	if err != nil {
		fmt.Printf("error loading services:\n%v\n", err)
		return
	} else {
		fmt.Printf("loaded services:\n%v\n", services.Services)
	}

	startDate := time.Date(2024, 02, 10, 00, 00, 00, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)

	spanStartDate, err := types.TimestampProto(startDate)
	if err != nil {
		fmt.Errorf("could not set span start date for FindTraces query: %v\n", err)
	}
	spanEndDate, err := types.TimestampProto(endDate)
	if err != nil {
		fmt.Errorf("could not set span end date for FindTraces query: %v\n", err)
	}

	findTracesClient, err := jaegerQueryClient.FindTraces(
		context.Background(),
		&api_v3.FindTracesRequest{
			Query: &api_v3.TraceQueryParameters{
				// ServiceName seems to be required
				ServiceName: "golang-filter",
				//OperationName: "SQL SELECT",
				//Tags: map[string]string{
				//	"span.kind": "client",
				//},
				StartTimeMin: spanStartDate,
				StartTimeMax: spanEndDate,
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

	spans := spansResponse.GetResourceSpans()

	if len(spans) == 0 {
		fmt.Printf("successfully queried. no spans found")
	} else {
		fmt.Printf("successfully queried. found %d *resource* spans\n", len(spans))
	}

	var traceids []model.TraceID
	traceidsSeen := map[model.TraceID]bool{}

	for _, resourceSpan := range spans {
		fmt.Printf("--> --> New resource span. Number of scope spans: %d\n", len(resourceSpan.ScopeSpans))
		for _, scopeSpan := range resourceSpan.ScopeSpans {
			fmt.Printf("New scope span with %d spans\n", len(scopeSpan.Spans))
			for _, s := range scopeSpan.GetSpans() {
				fmt.Printf("--> Span name: %s\n", s.Name)
				fmt.Printf("Span details: trace id %02x, span id: %02x, parent span id: %02x,\n", s.TraceID, s.SpanID, s.ParentSpanID)

				// check if the span is generated by our filter crudely using the span name
				if strings.Contains(s.Name, "encode") || strings.Contains(s.Name, "decode") {
					// scan through our attributes for the direction and the service name
					fmt.Printf("Golang filter's span\n")
				}
				for _, kv := range s.Attributes {
					if strings.Contains(kv.Key, "direction") || strings.Contains(kv.Key, "svcName") {
						fmt.Printf("Golang filter with key %s value %s\n", kv.Key, kv.Value.GetStringValue())
					} else if strings.Contains(kv.Key, "x-request-id") || strings.Contains(kv.Key, "address") || strings.Contains(kv.Key, "ip") || strings.Contains(kv.Key, "url") {
						fmt.Printf("%s: %s\n", kv.Key, kv.Value.GetStringValue())
					}
					// fmt.Printf("found an attribute with key %s value %s\n", kv.Key, kv.Value.GetStringValue())
				}

				if !traceidsSeen[s.TraceID] {
					traceids = append(traceids, s.TraceID)
				}
				traceidsSeen[s.TraceID] = true
			}
		}
	}

	fmt.Printf("\nfound %d relevant traces\n", len(traceids))

	fmt.Printf("selected 0th trace with name '%02x'\n", traceids[0])

	//entireTraceClient, err := jaegerQueryClient.GetTrace(
	//	context.Background(),
	//	&api_v3.GetTraceRequest{
	//		TraceId: traceids[0],
	//	},
	//)
	//if err != nil {
	//	fmt.Printf("error receiving details about the trace id \"%s\":\n%v\n", traceids[0], err)
	//	return
	//}
	//
	//entiretrace, err := entireTraceClient.Recv()
	//if err != nil {
	//	fmt.Printf("error receiving the response:\n%v\n", err)
	//	return
	//}
	////
	//allSpansInTrace := entiretrace.GetResourceSpans()
	//fmt.Printf("got all spans in trace: ", allSpansInTrace)
	////if len(allSpansInTrace) == 0 {
	////	fmt.Printf("no spans inside trace '%s'", tName)
	////}
	////for _, s := range allSpansInTrace {
	////	fmt.Printf("received spans inside a trace '%s':\n%v\n", tName, s)
	////}
	////}

}
