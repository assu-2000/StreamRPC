package main

import (
	"context"
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
	user, err := s.authService.Login(req.Username, req.Password)
	if err != nil {
		return nil, err
	}

	// TODO: Générer un vrai JWT
	token := "generated-jwt-token-for-" + user.ID

	return &pb.LoginResponse{
		Token:  token,
		UserId: user.ID,
	}, nil
}

func main() {
	pgConfig := database.PostgresConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "postgres",
		DBName:   "chatdb",
	}

	pgPool, err := database.NewPostgresConnection(pgConfig)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgPool.Close()

	authRepo := auth.NewPostgresRepository(pgPool)
	authService := auth.NewAuthService(authRepo)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
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
