package composer

import (
	"context"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"net"
	pb "privacy-profile-composer/pkg/proto"
)

func (s server) PostObservedProfile(
	ctx context.Context,
	profile *pb.SvcObservedProfile,
) (*emptypb.Empty, error) {
	if profile != nil {
		s.systemWideProfile = Composer(s.systemWideProfile, *profile)
	}
	return &emptypb.Empty{}, nil
}

func (s server) GetSystemWideProfile(
	ctx context.Context,
	e *emptypb.Empty,
) (*pb.SystemwideObservedProfile, error) {
	return &s.systemWideProfile, nil
}

func (s server) mustEmbedUnimplementedPrivacyProfileComposerServer() {
	panic("implement me")
}

type server struct {
	pb.UnimplementedPrivacyProfileComposerServer
	systemWideProfile pb.SystemwideObservedProfile
}

var (
	port = flag.Int("port", 50051, "The server port")
)

func Run_server() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterPrivacyProfileComposerServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
