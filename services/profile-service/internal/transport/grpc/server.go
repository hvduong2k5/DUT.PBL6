package grpc

import (
	"context"
	"crypto/subtle"
	"fmt"
	profilev1 "github.com/omamx/profile-service/internal/gen/profile/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"net"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

type Server struct {
	grpcServer *grpc.Server
	listener   net.Listener
	port       string
}

func NewServer(port string, profileServer *ProfileGRPCServer, token string) (*Server, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on grpc port %s: %w", port, err)
	}

	opts := []grpc.ServerOption{grpc.UnaryInterceptor(func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		values := md.Get("x-internal-token")
		if token == "" || len(values) != 1 || subtle.ConstantTimeCompare([]byte(values[0]), []byte(token)) != 1 {
			return nil, status.Error(codes.Unauthenticated, "authenticated internal caller required")
		}
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return handler(ctx, req)
	})}
	grpcSrv := grpc.NewServer(opts...)

	profilev1.RegisterProfileServiceServer(grpcSrv, profileServer)

	return &Server{
		grpcServer: grpcSrv,
		listener:   lis,
		port:       port,
	}, nil
}

func (s *Server) Start() error {
	log.Info().Str("port", s.port).Msg("gRPC Server listening")
	return s.grpcServer.Serve(s.listener)
}

func (s *Server) Addr() net.Addr { return s.listener.Addr() }

func (s *Server) Stop() {
	log.Info().Msg("Gracefully stopping gRPC Server...")
	done := make(chan struct{})
	go func() { s.grpcServer.GracefulStop(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		s.grpcServer.Stop()
		<-done
	}
}
