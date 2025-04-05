package main

import (
	"github.com/assu-2000/StreamRPC/config"
	"github.com/assu-2000/StreamRPC/internal/server"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/assu-2000/StreamRPC/internal/auth"
	"github.com/assu-2000/StreamRPC/internal/database"
	"github.com/assu-2000/StreamRPC/internal/pb"
	"google.golang.org/grpc"
)

func main() {
	pgConfig := config.LoadPostgresConfig()
	jwtConfig := config.LoadJWTConfig()
	jwtService := auth.NewJWTService(jwtConfig)

	pgPool, err := database.NewPostgresConnection((*database.PostgresConfig)(pgConfig))
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgPool.Close()

	authRepo := auth.NewUserPostgresRepository(pgPool)
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
	pb.RegisterAuthGrpcServiceServer(s, &server.AuthHandler{
		AuthService: authService,
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
