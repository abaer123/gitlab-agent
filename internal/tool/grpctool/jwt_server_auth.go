package grpctool

import (
	"context"
	"fmt"

	"github.com/dgrijalva/jwt-go/v4"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type JWTAuther struct {
	Secret   []byte
	Audience string
}

// UnaryServerInterceptor returns a new unary server interceptors that performs per-request JWT auth.
func (a *JWTAuther) UnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if err := a.doAuth(ctx); err != nil {
		return nil, err
	}
	return handler(ctx, req)
}

// StreamServerInterceptor returns a new stream server interceptors that performs per-request JWT auth.
func (a *JWTAuther) StreamServerInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if err := a.doAuth(stream.Context()); err != nil {
		return err
	}
	return handler(srv, stream)
}

func (a *JWTAuther) doAuth(ctx context.Context) error {
	token, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return err // returns gRPC status error
	}
	_, err = jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.Secret, nil
	}, jwt.WithAudience(a.Audience))
	if err != nil {
		LoggerFromContext(ctx).Debug("JWT auth failed", zap.Error(err))
		return status.Error(codes.Unauthenticated, "JWT validation failed")
	}
	return nil
}
