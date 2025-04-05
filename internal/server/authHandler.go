package server

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"

	"github.com/assu-2000/StreamRPC/internal/auth"
	"github.com/assu-2000/StreamRPC/internal/pb"
)

type ChatServer struct {
	pb.UnimplementedChatServiceServer
	AuthService *auth.AuthService
}

func (s *ChatServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	err := s.AuthService.Register(req.Username, req.Password, req.Email)
	if err != nil {
		return &pb.RegisterResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.RegisterResponse{
		Success: true,
		Message: "User created successfully",
	}, nil
}

func (s *ChatServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	user, accessToken, refreshToken, err := s.AuthService.Login(req.Username, req.Password)
	if err != nil {
		return nil, err
	}

	return &pb.LoginResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		UserId:       user.ID.String(),
	}, nil
}

func (s *ChatServer) CheckAuth(ctx context.Context, _ *emptypb.Empty) (*pb.AuthResponse, error) {
	userID := ctx.Value("user_id")
	return &pb.AuthResponse{
		Message: fmt.Sprintf("Hello, %s! You're authenticated.", userID),
	}, nil
}

func (s *ChatServer) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	accessToken, refreshToken, err := s.AuthService.HandleRefresh(ctx, req.RefreshToken)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "failed to refresh tokens: %v", err)
	}

	return &pb.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *ChatServer) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	userID, ok := ctx.Value("user_id").(uuid.UUID)
	fmt.Println("User Id: ", userID)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	if err := s.AuthService.RevokeAllTokens(ctx, userID); err != nil {
		log.Printf("Failed to revoke tokens: %v", err)
		return nil, status.Error(codes.Internal, "failed to logout")
	}

	return &pb.LogoutResponse{Success: true}, nil
}
