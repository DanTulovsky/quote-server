package server

import (
	"context"

	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"google.golang.org/grpc/reflection"

	"google.golang.org/grpc"

	pb "github.com/DanTulovsky/quote-server/proto"
)

// server is used to implement quote.Quote
type server struct {
	pb.UnimplementedQuoteServer
}

func NewServer() *grpc.Server {
	s := grpc.NewServer()
	pb.RegisterQuoteServer(s, &server{})
	reflection.Register(s)

	healthServer := health.NewServer()
	healthServer.SetServingStatus("grpc.health.v1.quoteservice", 1)
	grpc_health_v1.RegisterHealthServer(s, healthServer)

	return s
}

// GetQuote implements quote.Quote
func (s *server) GetQuote(ctx context.Context, _ *pb.GetQuoteRequest) (*pb.GetQuoteResponse, error) {
	quote := TheySaidSoQuote(ctx)
	return &pb.GetQuoteResponse{QuoteText: quote}, nil
}
