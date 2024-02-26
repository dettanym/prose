package composer

import (
	"context"
	"flag"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

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

// TODO: Call this within the jaeger trace querying API, to submit updates to all observed profiles,
//  after going through a batch of traces. Remove the flag as it won't be run via cli.
func sendComposedProfile(fqdn string, purpose string, piiTypes []string, thirdParties []string) {
	var (
		composerSvcAddr = flag.String("addr", "http://prose-server.prose-system.svc.cluster.local:50051", "the address to connect to")
	)

	flag.Parse()
	// Set up a connection to the server.
	conn, err := grpc.Dial(*composerSvcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("can not connect to Composer SVC at addr %v. ERROR: %v", composerSvcAddr, err)
		return
	}
	defer func(conn *grpc.ClientConn) {
		err = conn.Close()
		if err != nil {
			log.Printf("could not close connection to Composer server %s", err)
			return
		}
	}(conn)
	c := pb.NewPrivacyProfileComposerClient(conn)

	// Contact the server and print out its response.
	ctx := context.Background()

	processingEntries := make(map[string]*pb.DataItemAndThirdParties, len(piiTypes))
	for _, pii := range piiTypes {
		dataItemThirdParties := map[string]*pb.ThirdParties{
			pii: {
				ThirdParty: thirdParties,
			},
		}
		processingEntries[purpose] = &pb.DataItemAndThirdParties{Entry: dataItemThirdParties}
	}
	_, err = c.PostObservedProfile(
		ctx,
		&pb.SvcObservedProfile{
			SvcInternalFQDN: fqdn,
			ObservedProcessingEntries: &pb.PurposeBasedProcessing{
				ProcessingEntries: processingEntries},
		},
	)

	if err != nil {
		log.Printf("got this error when posting observed profile: %v", err)
	}
	return
}
