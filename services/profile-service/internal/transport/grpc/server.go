package grpc

import (
	"fmt"
	"net"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	grpcServer *grpc.Server
	listener   net.Listener
	port       string
}

func NewServer(port string, profileServer *ProfileGRPCServer) (*Server, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on grpc port %s: %w", port, err)
	}

	opts := []grpc.ServerOption{}
	grpcSrv := grpc.NewServer(opts...)

	// Enable reflection for grpcurl and dev inspection
	reflection.Register(grpcSrv)

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

func (s *Server) Stop() {
	log.Info().Msg("Gracefully stopping gRPC Server...")
	s.grpcServer.GracefulStop()
}
