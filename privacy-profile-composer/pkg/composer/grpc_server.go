package composer

import (
	"context"
	"google.golang.org/protobuf/types/known/emptypb"
	pb "privacy-profile-composer/pkg/proto"
)

func (s server) PostObservedProfile(
	ctx context.Context,
	profile *pb.SvcObservedProfile,
) (*emptypb.Empty, error) {
	if profile != nil {
		s.systemWideProfile = *Composer(&s.systemWideProfile, profile)
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

func NewComposerServer() server { return server{} }
