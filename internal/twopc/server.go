package twopc

import (
	"context"

	pb "studyroom/api/proto"
)

// TwoPCServer implements the TwoPCService gRPC server
type TwoPCServer struct {
	pb.UnimplementedTwoPCServiceServer
	participant *ParticipantNode
	coordinator *Coordinator  // Coordinator for phase-to-phase gRPC
}

// NewTwoPCServer creates a new 2PC server
func NewTwoPCServer(participant *ParticipantNode) *TwoPCServer {
	return &TwoPCServer{participant: participant}
}

// NewTwoPCServerWithCoordinator creates a new 2PC server with coordinator support
func NewTwoPCServerWithCoordinator(participant *ParticipantNode, coordinator *Coordinator) *TwoPCServer {
	return &TwoPCServer{
		participant: participant,
		coordinator: coordinator,
	}
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

// StartDecision handles StartDecision request (phase-to-phase gRPC)
func (s *TwoPCServer) StartDecision(ctx context.Context, req *pb.StartDecisionRequest) (*pb.StartDecisionResponse, error) {
	if s.coordinator == nil {
		return &pb.StartDecisionResponse{
			Success: false,
			Error:   "coordinator not available",
		}, nil
	}
	return s.coordinator.StartDecision(ctx, req)
}



