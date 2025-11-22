package raft

import (
	"context"
	"fmt"

	pb "studyroom/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// RaftClient is a gRPC client for Raft communication
type RaftClient struct {
	conn   *grpc.ClientConn
	client pb.RaftServiceClient
}

// NewRaftClient creates a new Raft client
func NewRaftClient(address string) (*RaftClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %v", address, err)
	}

	return &RaftClient{
		conn:   conn,
		client: pb.NewRaftServiceClient(conn),
	}, nil
}

// Close closes the connection
func (c *RaftClient) Close() error {
	return c.conn.Close()
}

// RequestVote sends a vote request
func (c *RaftClient) RequestVote(ctx context.Context, term int, candidateID string, lastLogIndex, lastLogTerm int, targetNodeID string) (bool, int, error) {
	// Print client-side log as required: Node <node_id> sends RPC <rpc_name> to Node <node_id>
	fmt.Printf("Node %s sends RPC RequestVote to Node %s\n", candidateID, targetNodeID)
	
	req := &pb.RequestVoteRequest{
		Term:         int32(term),
		CandidateId:  candidateID,
		LastLogIndex: int32(lastLogIndex),
		LastLogTerm:  int32(lastLogTerm),
	}

	resp, err := c.client.RequestVote(ctx, req)
	if err != nil {
		return false, 0, err
	}

	return resp.VoteGranted, int(resp.Term), nil
}

// AppendEntries sends append entries request
func (c *RaftClient) AppendEntries(ctx context.Context, term int, leaderID string, prevLogIndex, prevLogTerm int, entries []LogEntry, leaderCommit int, targetNodeID string) (bool, int, error) {
	// Print client-side log as required: Node <node_id> sends RPC <rpc_name> to Node <node_id>
	fmt.Printf("Node %s sends RPC AppendEntries to Node %s\n", leaderID, targetNodeID)
	
	pbEntries := make([]*pb.LogEntry, len(entries))
	for i, e := range entries {
		pbEntries[i] = &pb.LogEntry{
			Term:    int32(e.Term),
			Index:   int32(e.Index),
			Command: e.Command,
		}
	}

	req := &pb.AppendEntriesRequest{
		Term:         int32(term),
		LeaderId:     leaderID,
		PrevLogIndex: int32(prevLogIndex),
		PrevLogTerm:  int32(prevLogTerm),
		Entries:      pbEntries,
		LeaderCommit: int32(leaderCommit),
	}

	resp, err := c.client.AppendEntries(ctx, req)
	if err != nil {
		return false, 0, err
	}

	return resp.Success, int(resp.Term), nil
}

// Heartbeat sends a heartbeat
func (c *RaftClient) Heartbeat(ctx context.Context, term int, leaderID string, targetNodeID string) (bool, int, error) {
	// Print client-side log as required: Node <node_id> sends RPC <rpc_name> to Node <node_id>
	fmt.Printf("Node %s sends RPC Heartbeat to Node %s\n", leaderID, targetNodeID)
	
	req := &pb.HeartbeatRequest{
		Term:     int32(term),
		LeaderId: leaderID,
	}

	resp, err := c.client.Heartbeat(ctx, req)
	if err != nil {
		return false, 0, err
	}

	return resp.Success, int(resp.Term), nil
}

