/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package composer

import (
	"context"
	"flag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	pb "privacy-profile-composer/pkg/proto"
)

var (
	addr = flag.String("addr", "localhost:50051", "the address to connect to")
)

func Run_client() {
	flag.Parse()
	// Set up a connection to the server.
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewPrivacyProfileComposerClient(conn)

	// Contact the server and print out its response.
	ctx := context.Background()

	_, err = c.PostObservedProfile(
		ctx,
		&pb.SvcObservedProfile{
			SvcInternalFQDN: "advertising.svc.internal",
			ObservedProcessingEntries: &pb.PurposeBasedProcessing{
				ProcessingEntries: map[string]*pb.DataItemAndThirdParties{
					"advertising": {
						Entry: map[string]*pb.ThirdParties{
							"<EMAIL_ADDRESS>": {
								ThirdParty: []string{
									"google.com",
									"facebook.com",
								},
							},
						},
					},
				},
			},
		},
	)
	if err != nil {
		log.Fatalf("got this error when posting observed profile: %v", err)
	}

	profile, err := c.GetSystemWideProfile(ctx, &emptypb.Empty{})
	if err != nil {
		log.Fatalf("got this error when fetching system wide profile: %v", err)
	}
	log.Println(profile)
}