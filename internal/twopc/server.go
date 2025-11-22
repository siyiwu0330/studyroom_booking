package twopc

import (
	"context"

	pb "studyroom/api/proto"
)

// TwoPCServer implements the TwoPCService gRPC server
type TwoPCServer struct {
	pb.UnimplementedTwoPCServiceServer
	participant *ParticipantNode
}

// NewTwoPCServer creates a new 2PC server
func NewTwoPCServer(participant *ParticipantNode) *TwoPCServer {
	return &TwoPCServer{participant: participant}
}

// Prepare handles prepare request
func (s *TwoPCServer) Prepare(ctx context.Context, req *pb.PrepareRequest) (*pb.PrepareResponse, error) {
	return s.participant.Prepare(ctx, req)
}

// Commit handles commit request
func (s *TwoPCServer) Commit(ctx context.Context, req *pb.CommitRequest) (*pb.CommitResponse, error) {
	return s.participant.Commit(ctx, req)
}

// Abort handles abort request
func (s *TwoPCServer) Abort(ctx context.Context, req *pb.AbortRequest) (*pb.AbortResponse, error) {
	return s.participant.Abort(ctx, req)
}



