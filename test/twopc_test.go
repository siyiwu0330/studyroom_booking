package test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	pb "studyroom/api/proto"
	"studyroom/internal/twopc"
)


// Test2PCBasicCommit tests basic 2PC commit flow
func Test2PCBasicCommit(t *testing.T) {
	participant := twopc.NewParticipantNode("test-participant")

	// Set up participant callbacks
	prepareCalled := false

	participant.SetPrepareFunc(func(operation string, data map[string]interface{}) error {
		prepareCalled = true
		t.Logf("Prepare called with operation: %s", operation)
		return nil
	})

	participant.SetCommitFunc(func(operation string, data map[string]interface{}) error {
		t.Logf("Commit called with operation: %s", operation)
		return nil
	})

	participant.SetAbortFunc(func(operation string, data map[string]interface{}) error {
		t.Logf("Abort called with operation: %s", operation)
		return nil
	})

	// Create test transaction
	txnID := "test-txn-1"
	operation := map[string]interface{}{
		"type":    "create_booking",
		"room_id": "room1",
		"user_id": "user1",
	}
	opJSON, _ := json.Marshal(operation)

	// Note: This test requires actual gRPC servers running
	// For unit testing, we test the coordinator logic
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test prepare phase
	req := &pb.PrepareRequest{
		TransactionId: txnID,
		Participants: []*pb.Participant{
			{NodeId: "node1", Address: "localhost:50051"},
		},
		Operation: string(opJSON),
	}

	resp, err := participant.Prepare(ctx, req)
	if err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}

	if !resp.CanCommit {
		t.Error("Expected CanCommit to be true")
	}

	if !prepareCalled {
		t.Error("Prepare function was not called")
	}

	t.Log("✓ Test2PCBasicCommit: Prepare phase passed")
}

// Test2PCAbortOnPrepareFailure tests 2PC abort when prepare fails
func Test2PCAbortOnPrepareFailure(t *testing.T) {
	participant := twopc.NewParticipantNode("test-participant")

	// Set up participant to reject prepare
	participant.SetPrepareFunc(func(operation string, data map[string]interface{}) error {
		return errors.New("prepare failed")
	})

	participant.SetAbortFunc(func(operation string, data map[string]interface{}) error {
		t.Log("Abort called as expected")
		return nil
	})

	ctx := context.Background()
	req := &pb.PrepareRequest{
		TransactionId: "test-txn-2",
		Participants: []*pb.Participant{
			{NodeId: "node1", Address: "localhost:50051"},
		},
		Operation: `{"type":"create_booking"}`,
	}

	resp, err := participant.Prepare(ctx, req)
	if err != nil {
		t.Logf("Prepare returned error as expected: %v", err)
	}

	if resp != nil && resp.CanCommit {
		t.Error("Expected CanCommit to be false when prepare fails")
	}

	t.Log("✓ Test2PCAbortOnPrepareFailure: Abort on prepare failure passed")
}

// Test2PCConcurrentTransactions tests concurrent 2PC transactions
func Test2PCConcurrentTransactions(t *testing.T) {
	coordinator := twopc.NewCoordinator(nil, "test-node", "localhost:50051")

	txn1 := "txn-concurrent-1"
	txn2 := "txn-concurrent-2"

	participants := []twopc.Participant{
		{NodeID: "node1", Address: "localhost:50051"},
	}

	// Start two transactions concurrently
	done1 := make(chan bool)
	done2 := make(chan bool)

	go func() {
		err := coordinator.StartTransaction(txn1, participants, `{"type":"booking1"}`)
		done1 <- (err == nil)
	}()

	go func() {
		err := coordinator.StartTransaction(txn2, participants, `{"type":"booking2"}`)
		done2 <- (err == nil)
	}()

	<-done1
	<-done2

	// Check both transactions exist
	state1, err1 := coordinator.GetTransactionState(txn1)
	state2, err2 := coordinator.GetTransactionState(txn2)

	if err1 != nil || err2 != nil {
		t.Errorf("Failed to get transaction states: %v, %v", err1, err2)
	}

	if state1 == twopc.Initial && state2 == twopc.Initial {
		t.Log("✓ Test2PCConcurrentTransactions: Both transactions started successfully")
	} else {
		t.Errorf("Unexpected transaction states: %v, %v", state1, state2)
	}
}

