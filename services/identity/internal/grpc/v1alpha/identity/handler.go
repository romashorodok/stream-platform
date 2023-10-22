package identity

import (
	"context"

	identitypb "github.com/romashorodok/stream-platform/gen/golang/identity/v1alpha"
	"github.com/romashorodok/stream-platform/services/identity/internal/security"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PublicKeyService struct {
	identitypb.UnimplementedPublicKeyServiceServer

	securityService *security.SecurityServiceImpl
}

func (s PublicKeyService) PublicKeyList(ctx context.Context, _ *identitypb.PublicKeyListRequest) (*identitypb.PublicKeyListResponse, error) {

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "Unable parse metadata for request.")
	}

	rawToken := md.Get("authorization")
	if rawToken == nil {
		return nil, status.Error(codes.FailedPrecondition, "The route require authorization metadata")
	}

	result, err := s.securityService.GetPublciKeys(rawToken[0])
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &identitypb.PublicKeyListResponse{
		Result:      result,
		GeneratedAt: timestamppb.Now(),
	}, nil
}

type PublicKeyGRPCServiceParams struct {
	fx.In

	Server          *grpc.Server
	SecurityService *security.SecurityServiceImpl
}

func NewPublicKeyGRPCService(params PublicKeyGRPCServiceParams) {
	identitypb.RegisterPublicKeyServiceServer(params.Server, &PublicKeyService{
		securityService: params.SecurityService,
	})
}

func requireAuthMetaOnRequest(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	meta, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "Unable parse metadata for request.")
	}

	// TODO: old grpc generators don't have that var
	// if info.FullMethod != identitypb.PublicKeyService_PublicKeyList_FullMethodName {
	// 	return handler(ctx, req)
	// }

	if auth := meta.Get("authorization"); auth == nil || auth[0] == "" {
		return nil, status.Error(codes.FailedPrecondition, "The route require authorization metadata")
	}

	return handler(ctx, req)
}

func NewPublicKeyListInterceptor() grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(requireAuthMetaOnRequest)
}
