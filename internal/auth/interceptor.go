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
		if info.FullMethod == "/chat.AuthGrpcService/Login" ||
			info.FullMethod == "/chat.AuthGrpcService/Register" {
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

func (s *AuthService) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return status.Error(codes.Unauthenticated, "metadata is not provided")
		}

		token, err := extractToken(md)
		if err != nil {
			return err
		}

		claims, err := s.jwtService.ValidateToken(token)
		if err != nil {
			return status.Error(codes.Unauthenticated, "invalid token")
		}

		// Inject user_id in stream context
		newCtx := context.WithValue(ss.Context(), "user_id", claims.UserID)

		// Wrap stream to override its context with the new one
		wrapped := &wrappedStream{ServerStream: ss, ctx: newCtx}

		fmt.Println("Stream Interceptor user id:", claims.UserID)

		return handler(srv, wrapped)
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

type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}
