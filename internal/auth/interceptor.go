package auth

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	authorizationHeader = "authorization"
	bearerPrefix        = "Bearer "
)

func (s *AuthService) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Skip auth for login and register
		if info.FullMethod == "/chat.ChatService/Login" ||
			info.FullMethod == "/chat.ChatService/Register" {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "metadata is not provided")
		}

		token, err := extractToken(md)
		if err != nil {
			return nil, err
		}

		claims, err := s.jwtService.ValidateToken(token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}
		// Adding claims to the contexte (user_id)
		newCtx := context.WithValue(ctx, "user_id", claims.UserID)
		fmt.Println("Interceptor user id:", claims.UserID)
		return handler(newCtx, req)
	}
}

func extractToken(md metadata.MD) (string, error) {
	authHeaders := md.Get(authorizationHeader)
	if len(authHeaders) == 0 {
		return "", status.Error(codes.Unauthenticated, "authorization token is not provided")
	}

	token := authHeaders[0]
	if !strings.HasPrefix(token, bearerPrefix) {
		return "", status.Error(codes.Unauthenticated, "invalid authorization format")
	}

	return token[len(bearerPrefix):], nil
}
