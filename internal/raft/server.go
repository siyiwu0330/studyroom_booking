package raft

import (
	"context"
	"fmt"

	pb "studyroom/api/proto"
)

// RaftServer implements the RaftService gRPC server
type RaftServer struct {
	pb.UnimplementedRaftServiceServer
	node *Node
}

// NewRaftServer creates a new Raft server
func NewRaftServer(node *Node) *RaftServer {
	return &RaftServer{node: node}
}

// RequestVote handles vote request from candidate
func (s *RaftServer) RequestVote(ctx context.Context, req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	// Print server-side log as required: Node <node_id> runs RPC <rpc_name> called by Node <node_id>
	fmt.Printf("Node %s runs RPC RequestVote called by Node %s\n", s.node.GetID(), req.CandidateId)
	
	voteGranted, currentTerm := s.node.HandleRequestVote(
		int(req.Term),
		req.CandidateId,
		int(req.LastLogIndex),
		int(req.LastLogTerm),
	)

	return &pb.RequestVoteResponse{
		Term:        int32(currentTerm),
		VoteGranted: voteGranted,
	}, nil
}

// AppendEntries handles append entries from leader
func (s *RaftServer) AppendEntries(ctx context.Context, req *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
	// Print server-side log as required: Node <node_id> runs RPC <rpc_name> called by Node <node_id>
	fmt.Printf("Node %s runs RPC AppendEntries called by Node %s\n", s.node.GetID(), req.LeaderId)
	
	entries := make([]LogEntry, len(req.Entries))
	for i, e := range req.Entries {
		entries[i] = LogEntry{
			Term:    int(e.Term),
			Index:   int(e.Index),
			Command: e.Command,
		}
	}

	success, currentTerm := s.node.HandleAppendEntries(
		int(req.Term),
		req.LeaderId,
		int(req.PrevLogIndex),
		int(req.PrevLogTerm),
		entries,
		int(req.LeaderCommit),
	)

	return &pb.AppendEntriesResponse{
		Term:    int32(currentTerm),
		Success: success,
	}, nil
}

// Heartbeat handles heartbeat from leader
func (s *RaftServer) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	// Print server-side log as required: Node <node_id> runs RPC <rpc_name> called by Node <node_id>
	fmt.Printf("Node %s runs RPC Heartbeat called by Node %s\n", s.node.GetID(), req.LeaderId)
	
	// Heartbeat is essentially an empty AppendEntries
	success, currentTerm := s.node.HandleAppendEntries(
		int(req.Term),
		req.LeaderId,
		0,
		0,
		[]LogEntry{},
		0,
	)

	return &pb.HeartbeatResponse{
		Term:    int32(currentTerm),
		Success: success,
	}, nil
}

