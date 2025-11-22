// +build integration

package test

import (
	"context"
	"testing"
	"time"

	pb "studyroom/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Test2PCIntegration tests 2PC with actual gRPC calls
func Test2PCIntegration(t *testing.T) {
	// This test requires running gRPC servers
	// Skip if servers are not available
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Skip("gRPC server not available, skipping integration test")
		return
	}
	defer conn.Close()

	client := pb.NewTwoPCServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test prepare
	req := &pb.PrepareRequest{
		TransactionId: "integration-test-1",
		Participants: []*pb.Participant{
			{NodeId: "node1", Address: "localhost:50051"},
		},
		Operation: `{"type":"create_booking","room_id":"room1"}`,
	}

	resp, err := client.Prepare(ctx, req)
	if err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}

	if resp.CanCommit {
		t.Log("✓ Test2PCIntegration: Prepare phase successful")
	} else {
		t.Errorf("Prepare failed: %s", resp.Error)
	}
}

