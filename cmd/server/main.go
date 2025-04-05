package main

import (
	"context"
	"github.com/assu-2000/StreamRPC/config"
	"github.com/assu-2000/StreamRPC/internal/auth"
	"github.com/assu-2000/StreamRPC/internal/database"
	"github.com/assu-2000/StreamRPC/internal/pb"
	"github.com/assu-2000/StreamRPC/internal/room"
	"github.com/assu-2000/StreamRPC/internal/server"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"os/signal"
	"time"
)

func main() {
	pgConfig := config.LoadPostgresConfig()
	jwtConfig := config.LoadJWTConfig()
	jwtService := auth.NewJWTService(jwtConfig)

	redisClient := initRedis()

	// RoomService
	roomRepo := room.NewRedisRepository(redisClient)
	roomService := room.NewRoomService(roomRepo)
	roomHandler := room.NewGRPCHandler(roomService)

	//
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
	pb.RegisterRoomGrpcServiceServer(s, roomHandler)

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

func initRedis() *redis.Client {
	redisClient := redis.NewClient(&redis.Options{
		Addr:            os.Getenv("REDIS_URL"),
		Password:        "",
		DB:              0,
		MaxRetries:      3,
		DialTimeout:     5 * time.Second,
		MinRetryBackoff: 300 * time.Millisecond,
		MaxRetryBackoff: 500 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	return redisClient
}
