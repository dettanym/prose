// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v4.24.2
// source: pkg/proto/privacy_profiles.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// PrivacyProfileComposerClient is the client API for PrivacyProfileComposer service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type PrivacyProfileComposerClient interface {
	// Sends a greeting
	PostObservedProfile(ctx context.Context, in *SvcObservedProfile, opts ...grpc.CallOption) (*emptypb.Empty, error)
	// Sends another greeting
	GetSystemWideProfile(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*SystemwideObservedProfile, error)
}

type privacyProfileComposerClient struct {
	cc grpc.ClientConnInterface
}

func NewPrivacyProfileComposerClient(cc grpc.ClientConnInterface) PrivacyProfileComposerClient {
	return &privacyProfileComposerClient{cc}
}

func (c *privacyProfileComposerClient) PostObservedProfile(ctx context.Context, in *SvcObservedProfile, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/privacy_profiles.PrivacyProfileComposer/PostObservedProfile", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *privacyProfileComposerClient) GetSystemWideProfile(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*SystemwideObservedProfile, error) {
	out := new(SystemwideObservedProfile)
	err := c.cc.Invoke(ctx, "/privacy_profiles.PrivacyProfileComposer/GetSystemWideProfile", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// PrivacyProfileComposerServer is the server API for PrivacyProfileComposer service.
// All implementations must embed UnimplementedPrivacyProfileComposerServer
// for forward compatibility
type PrivacyProfileComposerServer interface {
	// Sends a greeting
	PostObservedProfile(context.Context, *SvcObservedProfile) (*emptypb.Empty, error)
	// Sends another greeting
	GetSystemWideProfile(context.Context, *emptypb.Empty) (*SystemwideObservedProfile, error)
	mustEmbedUnimplementedPrivacyProfileComposerServer()
}

// UnimplementedPrivacyProfileComposerServer must be embedded to have forward compatible implementations.
type UnimplementedPrivacyProfileComposerServer struct {
}

func (UnimplementedPrivacyProfileComposerServer) PostObservedProfile(context.Context, *SvcObservedProfile) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PostObservedProfile not implemented")
}
func (UnimplementedPrivacyProfileComposerServer) GetSystemWideProfile(context.Context, *emptypb.Empty) (*SystemwideObservedProfile, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetSystemWideProfile not implemented")
}
func (UnimplementedPrivacyProfileComposerServer) mustEmbedUnimplementedPrivacyProfileComposerServer() {
}

// UnsafePrivacyProfileComposerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to PrivacyProfileComposerServer will
// result in compilation errors.
type UnsafePrivacyProfileComposerServer interface {
	mustEmbedUnimplementedPrivacyProfileComposerServer()
}

func RegisterPrivacyProfileComposerServer(s grpc.ServiceRegistrar, srv PrivacyProfileComposerServer) {
	s.RegisterService(&PrivacyProfileComposer_ServiceDesc, srv)
}

func _PrivacyProfileComposer_PostObservedProfile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SvcObservedProfile)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PrivacyProfileComposerServer).PostObservedProfile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/privacy_profiles.PrivacyProfileComposer/PostObservedProfile",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PrivacyProfileComposerServer).PostObservedProfile(ctx, req.(*SvcObservedProfile))
	}
	return interceptor(ctx, in, info, handler)
}

func _PrivacyProfileComposer_GetSystemWideProfile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PrivacyProfileComposerServer).GetSystemWideProfile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/privacy_profiles.PrivacyProfileComposer/GetSystemWideProfile",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PrivacyProfileComposerServer).GetSystemWideProfile(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

// PrivacyProfileComposer_ServiceDesc is the grpc.ServiceDesc for PrivacyProfileComposer service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var PrivacyProfileComposer_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "privacy_profiles.PrivacyProfileComposer",
	HandlerType: (*PrivacyProfileComposerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "PostObservedProfile",
			Handler:    _PrivacyProfileComposer_PostObservedProfile_Handler,
		},
		{
			MethodName: "GetSystemWideProfile",
			Handler:    _PrivacyProfileComposer_GetSystemWideProfile_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "pkg/proto/privacy_profiles.proto",
}