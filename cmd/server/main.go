package main

import (
	"context"
	"fmt"
	"github.com/assu-2000/StreamRPC/config"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/assu-2000/StreamRPC/internal/auth"
	"github.com/assu-2000/StreamRPC/internal/database"
	"github.com/assu-2000/StreamRPC/internal/pb"
	"google.golang.org/grpc"
)

type chatServer struct {
	pb.UnimplementedChatServiceServer
	authService *auth.AuthService
}

func (s *chatServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	err := s.authService.Register(req.Username, req.Password, req.Email)
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

func (s *chatServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	user, accessToken, refreshToken, err := s.authService.Login(req.Username, req.Password)
	if err != nil {
		return nil, err
	}

	return &pb.LoginResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		UserId:       user.ID.String(),
	}, nil
}

func (s *chatServer) CheckAuth(ctx context.Context, _ *emptypb.Empty) (*pb.AuthResponse, error) {
	userID := ctx.Value("user_id")
	return &pb.AuthResponse{
		Message: fmt.Sprintf("Hello, %s! You're authenticated.", userID),
	}, nil
}

func (s *chatServer) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	accessToken, refreshToken, err := s.authService.HandleRefresh(ctx, req.RefreshToken)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "failed to refresh tokens: %v", err)
	}

	return &pb.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *chatServer) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	userID, ok := ctx.Value("user_id").(uuid.UUID)
	fmt.Println("User Id: ", userID)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	if err := s.authService.RevokeAllTokens(ctx, userID); err != nil {
		log.Printf("Failed to revoke tokens: %v", err)
		return nil, status.Error(codes.Internal, "failed to logout")
	}

	return &pb.LogoutResponse{Success: true}, nil
}

func main() {
	pgConfig := config.LoadPostgresConfig()
	jwtConfig := config.LoadJWTConfig()
	jwtService := auth.NewJWTService(jwtConfig)

	pgPool, err := database.NewPostgresConnection((*database.PostgresConfig)(pgConfig))
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgPool.Close()

	authRepo := auth.NewPostgresRepository(pgPool)
	tokenRepo := auth.NewPostgresTokenRepository(pgPool)
	tokenService := auth.NewTokenService(tokenRepo, jwtService, jwtConfig.AccessDuration, jwtConfig.RefreshDuration)

	authService := auth.NewAuthService(authRepo, jwtService, tokenService)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(
		grpc.UnaryInterceptor(authService.UnaryInterceptor()),
	)
	pb.RegisterChatServiceServer(s, &chatServer{
		authService: authService,
	})

	go func() {
		log.Println("Server starting on port 50051...")
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Safe Server ShutDown
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch
	log.Println("Stopping the server...")
	s.GracefulStop()
	log.Println("Server stopped")
}
