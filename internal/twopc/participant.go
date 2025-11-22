package twopc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	pb "studyroom/api/proto"
)

// ParticipantState represents the state of a participant in a transaction
type ParticipantState int

const (
	PInitial ParticipantState = iota
	PPrepared
	PCommitted
	PAborted
)

// ParticipantTransaction represents a transaction from participant's perspective
type ParticipantTransaction struct {
	ID        string
	State     ParticipantState
	Operation string
	Data      map[string]interface{}
	mu        sync.RWMutex
}

// ParticipantNode handles 2PC participant operations
type ParticipantNode struct {
	transactions map[string]*ParticipantTransaction
	mu           sync.RWMutex
	prepareFunc  func(operation string, data map[string]interface{}) error
	commitFunc   func(operation string, data map[string]interface{}) error
	abortFunc    func(operation string, data map[string]interface{}) error
	nodeID       string // Node ID for logging
}

// NewParticipantNode creates a new participant node
func NewParticipantNode(nodeID string) *ParticipantNode {
	return &ParticipantNode{
		transactions: make(map[string]*ParticipantTransaction),
		nodeID:      nodeID,
	}
}

// SetPrepareFunc sets the function to execute during prepare phase
func (p *ParticipantNode) SetPrepareFunc(fn func(operation string, data map[string]interface{}) error) {
	p.prepareFunc = fn
}

// SetCommitFunc sets the function to execute during commit phase
func (p *ParticipantNode) SetCommitFunc(fn func(operation string, data map[string]interface{}) error) {
	p.commitFunc = fn
}

// SetAbortFunc sets the function to execute during abort phase
func (p *ParticipantNode) SetAbortFunc(fn func(operation string, data map[string]interface{}) error) {
	p.abortFunc = fn
}

// Prepare handles prepare request from coordinator
// Q1: Voting Phase - receives vote-request, returns vote-commit or vote-abort
func (p *ParticipantNode) Prepare(ctx context.Context, req *pb.PrepareRequest) (*pb.PrepareResponse, error) {
	// Extract coordinator node ID from participants (first participant is usually the coordinator)
	coordinatorNodeID := "unknown"
	if len(req.Participants) > 0 {
		coordinatorNodeID = req.Participants[0].NodeId
	}
	
	// Print server-side log as required: Phase <phase_name> of Node <node_id> receives RPC <rpc_name> from Phase <phase_name> of Node <node_id>
	fmt.Printf("Phase Voting of Node %s receives RPC vote-request from Phase Voting of Node %s\n", p.nodeID, coordinatorNodeID)
	
	p.mu.Lock()
	defer p.mu.Unlock()

	// Parse operation
	var opData map[string]interface{}
	if err := json.Unmarshal([]byte(req.Operation), &opData); err != nil {
		// Print vote-abort response
		fmt.Printf("Phase Voting of Node %s sends RPC vote-abort to Phase Voting of Node %s\n", p.nodeID, coordinatorNodeID)
		return &pb.PrepareResponse{
			CanCommit: false,
			Error:     fmt.Sprintf("failed to parse operation: %v", err),
		}, nil
	}

	// Create or update transaction
	txn, exists := p.transactions[req.TransactionId]
	if !exists {
		txn = &ParticipantTransaction{
			ID:        req.TransactionId,
			State:     PInitial,
			Operation: req.Operation,
			Data:      opData,
		}
		p.transactions[req.TransactionId] = txn
	}

	txn.mu.Lock()
	defer txn.mu.Unlock()

	if txn.State != PInitial {
		return &pb.PrepareResponse{
			CanCommit: false,
			Error:     fmt.Sprintf("transaction %s is not in Initial state", req.TransactionId),
		}, nil
	}

	// Execute prepare function
	if p.prepareFunc != nil {
		if err := p.prepareFunc(req.Operation, opData); err != nil {
			log.Printf("[2PC Participant] Prepare failed for transaction %s: %v", req.TransactionId, err)
			// Print vote-abort response
			fmt.Printf("Phase Voting of Node %s sends RPC vote-abort to Phase Voting of Node %s\n", p.nodeID, coordinatorNodeID)
			return &pb.PrepareResponse{
				CanCommit: false,
				Error:     err.Error(),
			}, nil
		}
	}

	// Mark as prepared
	txn.State = PPrepared
	log.Printf("[2PC Participant] Prepared transaction %s", req.TransactionId)
	
	// Print vote-commit response
	fmt.Printf("Phase Voting of Node %s sends RPC vote-commit to Phase Voting of Node %s\n", p.nodeID, coordinatorNodeID)

	return &pb.PrepareResponse{
		CanCommit: true,
	}, nil
}

// Commit handles commit request from coordinator
// Q2: Decision Phase - receives global-commit
func (p *ParticipantNode) Commit(ctx context.Context, req *pb.CommitRequest) (*pb.CommitResponse, error) {
	// Note: We don't have coordinator node ID in CommitRequest, so we use "coordinator" as placeholder
	coordinatorNodeID := "coordinator"
	
	// Print server-side log as required: Phase <phase_name> of Node <node_id> receives RPC <rpc_name> from Phase <phase_name> of Node <node_id>
	fmt.Printf("Phase Decision of Node %s receives RPC global-commit from Phase Decision of Node %s\n", p.nodeID, coordinatorNodeID)
	
	p.mu.RLock()
	txn, exists := p.transactions[req.TransactionId]
	p.mu.RUnlock()

	if !exists {
		return &pb.CommitResponse{
			Success: false,
			Error:   fmt.Sprintf("transaction %s not found", req.TransactionId),
		}, nil
	}

	txn.mu.Lock()
	defer txn.mu.Unlock()

	if txn.State != PPrepared {
		return &pb.CommitResponse{
			Success: false,
			Error:   fmt.Sprintf("transaction %s is not in Prepared state", req.TransactionId),
		}, nil
	}

	// Execute commit function
	if p.commitFunc != nil {
		if err := p.commitFunc(txn.Operation, txn.Data); err != nil {
			log.Printf("[2PC Participant] Commit failed for transaction %s: %v", req.TransactionId, err)
			return &pb.CommitResponse{
				Success: false,
				Error:   err.Error(),
			}, nil
		}
	}

	// Mark as committed
	txn.State = PCommitted
	log.Printf("[2PC Participant] Committed transaction %s", req.TransactionId)

	return &pb.CommitResponse{
		Success: true,
	}, nil
}

// Abort handles abort request from coordinator
// Q2: Decision Phase - receives global-abort
func (p *ParticipantNode) Abort(ctx context.Context, req *pb.AbortRequest) (*pb.AbortResponse, error) {
	// Note: We don't have coordinator node ID in AbortRequest, so we use "coordinator" as placeholder
	coordinatorNodeID := "coordinator"
	
	// Print server-side log as required: Phase <phase_name> of Node <node_id> receives RPC <rpc_name> from Phase <phase_name> of Node <node_id>
	fmt.Printf("Phase Decision of Node %s receives RPC global-abort from Phase Decision of Node %s\n", p.nodeID, coordinatorNodeID)
	
	p.mu.RLock()
	txn, exists := p.transactions[req.TransactionId]
	p.mu.RUnlock()

	if !exists {
		return &pb.AbortResponse{
			Success: false,
			Error:   fmt.Sprintf("transaction %s not found", req.TransactionId),
		}, nil
	}

	txn.mu.Lock()
	defer txn.mu.Unlock()

	if txn.State == PCommitted {
		return &pb.AbortResponse{
			Success: false,
			Error:   fmt.Sprintf("cannot abort committed transaction %s", req.TransactionId),
		}, nil
	}

	// Execute abort function
	if p.abortFunc != nil {
		if err := p.abortFunc(txn.Operation, txn.Data); err != nil {
			log.Printf("[2PC Participant] Abort failed for transaction %s: %v", req.TransactionId, err)
		}
	}

	// Mark as aborted
	txn.State = PAborted
	log.Printf("[2PC Participant] Aborted transaction %s", req.TransactionId)

	return &pb.AbortResponse{
		Success: true,
	}, nil
}

