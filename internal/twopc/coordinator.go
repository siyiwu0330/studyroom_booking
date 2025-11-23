package twopc

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	pb "studyroom/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TransactionState represents the state of a transaction
type TransactionState int

const (
	Initial TransactionState = iota
	Prepared
	Committed
	Aborted
)

// Transaction represents a 2PC transaction
type Transaction struct {
	ID          string
	State       TransactionState
	Participants []Participant
	Operation   string
	StartTime   time.Time
	mu          sync.RWMutex
}

// Participant represents a transaction participant
type Participant struct {
	NodeID  string
	Address string
}

// Coordinator manages 2PC transactions
type Coordinator struct {
	transactions map[string]*Transaction
	mu           sync.RWMutex
	raftNode     interface{ IsLeader() bool } // Interface to check if Raft leader
	nodeID       string                        // Node ID for logging
	address      string                        // Coordinator's own address for phase-to-phase gRPC
}

// NewCoordinator creates a new 2PC coordinator
func NewCoordinator(raftNode interface{ IsLeader() bool }, nodeID string, address string) *Coordinator {
	return &Coordinator{
		transactions: make(map[string]*Transaction),
		raftNode:    raftNode,
		nodeID:     nodeID,
		address:    address,
	}
}

// StartTransaction starts a new 2PC transaction
func (c *Coordinator) StartTransaction(transactionID string, participants []Participant, operation string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Only leader can coordinate transactions
	if c.raftNode != nil && !c.raftNode.IsLeader() {
		return fmt.Errorf("only leader can coordinate transactions")
	}

	if _, exists := c.transactions[transactionID]; exists {
		return fmt.Errorf("transaction %s already exists", transactionID)
	}

	c.transactions[transactionID] = &Transaction{
		ID:           transactionID,
		State:        Initial,
		Participants: participants,
		Operation:    operation,
		StartTime:    time.Now(),
	}

	log.Printf("[2PC] Started transaction %s with %d participants", transactionID, len(participants))
	return nil
}

// PreparePhase executes the prepare phase of 2PC
func (c *Coordinator) PreparePhase(ctx context.Context, transactionID string) (bool, error) {
	c.mu.RLock()
	txn, exists := c.transactions[transactionID]
	c.mu.RUnlock()

	if !exists {
		return false, fmt.Errorf("transaction %s not found", transactionID)
	}

	txn.mu.Lock()
	if txn.State != Initial {
		txn.mu.Unlock()
		return false, fmt.Errorf("transaction %s is not in Initial state", transactionID)
	}
	txn.State = Prepared
	txn.mu.Unlock()

	log.Printf("[2PC] Starting prepare phase for transaction %s", transactionID)

	// Send prepare requests to all participants
	var wg sync.WaitGroup
	results := make(chan bool, len(txn.Participants))
	errors := make(chan error, len(txn.Participants))

	for _, participant := range txn.Participants {
		wg.Add(1)
		go func(p Participant) {
			defer wg.Done()
			canCommit, err := c.sendPrepare(ctx, p, transactionID, txn.Operation, c.nodeID)
			if err != nil {
				errors <- err
				results <- false
				return
			}
			results <- canCommit
		}(participant)
	}

	wg.Wait()
	close(results)
	close(errors)

	// Check for errors
	hasErrors := false
	for err := range errors {
		if err != nil {
			log.Printf("[2PC] Error in prepare phase: %v", err)
			hasErrors = true
		}
	}

	// Check if all participants can commit
	allCanCommit := true
	for canCommit := range results {
		if !canCommit {
			allCanCommit = false
			break
		}
	}

	if hasErrors || !allCanCommit {
		log.Printf("[2PC] Prepare phase failed for transaction %s", transactionID)
		// Abort transaction
		go c.AbortPhase(ctx, transactionID)
		return false, fmt.Errorf("prepare phase failed")
	}

	log.Printf("[2PC] Prepare phase succeeded for transaction %s", transactionID)
	return true, nil
}

// CommitPhase executes the commit phase of 2PC
func (c *Coordinator) CommitPhase(ctx context.Context, transactionID string) error {
	c.mu.RLock()
	txn, exists := c.transactions[transactionID]
	c.mu.RUnlock()

	if !exists {
		return fmt.Errorf("transaction %s not found", transactionID)
	}

	txn.mu.Lock()
	if txn.State != Prepared {
		txn.mu.Unlock()
		return fmt.Errorf("transaction %s is not in Prepared state", transactionID)
	}
	txn.State = Committed
	txn.mu.Unlock()

	log.Printf("[2PC] Starting commit phase for transaction %s", transactionID)

	// Send commit requests to all participants
	var wg sync.WaitGroup
	errors := make(chan error, len(txn.Participants))

	for _, participant := range txn.Participants {
		wg.Add(1)
		go func(p Participant) {
			defer wg.Done()
			if err := c.sendCommit(ctx, p, transactionID); err != nil {
				errors <- err
			}
		}(participant)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		if err != nil {
			log.Printf("[2PC] Error in commit phase: %v", err)
			// Even if some commits fail, we consider the transaction committed
			// In production, you might want to implement retry logic
		}
	}

	log.Printf("[2PC] Commit phase completed for transaction %s", transactionID)
	return nil
}

// AbortPhase executes the abort phase of 2PC
func (c *Coordinator) AbortPhase(ctx context.Context, transactionID string) error {
	c.mu.RLock()
	txn, exists := c.transactions[transactionID]
	c.mu.RUnlock()

	if !exists {
		return fmt.Errorf("transaction %s not found", transactionID)
	}

	txn.mu.Lock()
	if txn.State == Committed {
		txn.mu.Unlock()
		return fmt.Errorf("cannot abort committed transaction %s", transactionID)
	}
	txn.State = Aborted
	txn.mu.Unlock()

	log.Printf("[2PC] Starting abort phase for transaction %s", transactionID)

	// Send abort requests to all participants
	var wg sync.WaitGroup
	errors := make(chan error, len(txn.Participants))

	for _, participant := range txn.Participants {
		wg.Add(1)
		go func(p Participant) {
			defer wg.Done()
			if err := c.sendAbort(ctx, p, transactionID); err != nil {
				errors <- err
			}
		}(participant)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		if err != nil {
			log.Printf("[2PC] Error in abort phase: %v", err)
		}
	}

	log.Printf("[2PC] Abort phase completed for transaction %s", transactionID)
	return nil
}

// ExecuteTransaction executes a complete 2PC transaction
func (c *Coordinator) ExecuteTransaction(ctx context.Context, transactionID string, participants []Participant, operation string) error {
	// Start transaction
	if err := c.StartTransaction(transactionID, participants, operation); err != nil {
		return err
	}

	// Prepare phase (Voting Phase)
	canCommit, err := c.PreparePhase(ctx, transactionID)
	if err != nil || !canCommit {
		// If prepare fails, call decision phase via gRPC to abort
		if err := c.callDecisionPhaseViaGRPC(ctx, transactionID, false); err != nil {
			return fmt.Errorf("prepare phase failed and decision phase call failed: %v", err)
		}
		return fmt.Errorf("prepare phase failed: %v", err)
	}

	// Decision phase: Call via gRPC (phase-to-phase communication)
	if err := c.callDecisionPhaseViaGRPC(ctx, transactionID, true); err != nil {
		return fmt.Errorf("decision phase failed: %v", err)
	}

	return nil
}

// callDecisionPhaseViaGRPC calls the decision phase via gRPC (phase-to-phase communication)
func (c *Coordinator) callDecisionPhaseViaGRPC(ctx context.Context, transactionID string, allVotedCommit bool) error {
	// Log: Phase Voting of Node <coordinator> sends RPC StartDecision to Phase Decision of Node <coordinator>
	fmt.Printf("Phase Voting of Node %s sends RPC StartDecision to Phase Decision of Node %s\n", c.nodeID, c.nodeID)
	
	// Connect to decision phase via localhost gRPC
	conn, err := grpc.Dial(c.address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to decision phase at %s: %v", c.address, err)
	}
	defer conn.Close()

	client := pb.NewTwoPCServiceClient(conn)

	req := &pb.StartDecisionRequest{
		TransactionId:  transactionID,
		AllVotedCommit: allVotedCommit,
	}

	resp, err := client.StartDecision(ctx, req)
	if err != nil {
		return fmt.Errorf("StartDecision RPC failed: %v", err)
	}

	if !resp.Success {
		return fmt.Errorf("decision phase failed: %s", resp.Error)
	}

	return nil
}

// sendPrepare sends a prepare request to a participant
// Q1: Voting Phase - vote-request
func (c *Coordinator) sendPrepare(ctx context.Context, participant Participant, transactionID, operation string, coordinatorNodeID string) (bool, error) {
	// Print client-side log as required: Phase <phase_name> of Node <node_id> sends RPC <rpc_name> to Phase <phase_name> of Node <node_id>
	fmt.Printf("Phase Voting of Node %s sends RPC vote-request to Phase Voting of Node %s\n", coordinatorNodeID, participant.NodeID)
	
	conn, err := grpc.Dial(participant.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return false, fmt.Errorf("failed to connect to %s: %v", participant.Address, err)
	}
	defer conn.Close()

	client := pb.NewTwoPCServiceClient(conn)

	participants := []*pb.Participant{
		{NodeId: participant.NodeID, Address: participant.Address},
	}

	req := &pb.PrepareRequest{
		TransactionId: transactionID,
		Participants:  participants,
		Operation:     operation,
	}

	resp, err := client.Prepare(ctx, req)
	if err != nil {
		return false, err
	}

	return resp.CanCommit, nil
}

// sendCommit sends a commit request to a participant
// Q2: Decision Phase - global-commit
func (c *Coordinator) sendCommit(ctx context.Context, participant Participant, transactionID string) error {
	// Print client-side log as required: Phase <phase_name> of Node <node_id> sends RPC <rpc_name> to Phase <phase_name> of Node <node_id>
	fmt.Printf("Phase Decision of Node %s sends RPC global-commit to Phase Decision of Node %s\n", c.nodeID, participant.NodeID)
	
	conn, err := grpc.Dial(participant.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", participant.Address, err)
	}
	defer conn.Close()

	client := pb.NewTwoPCServiceClient(conn)

	req := &pb.CommitRequest{
		TransactionId: transactionID,
	}

	resp, err := client.Commit(ctx, req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("commit failed: %s", resp.Error)
	}

	return nil
}

// sendAbort sends an abort request to a participant
// Q2: Decision Phase - global-abort
func (c *Coordinator) sendAbort(ctx context.Context, participant Participant, transactionID string) error {
	// Print client-side log as required: Phase <phase_name> of Node <node_id> sends RPC <rpc_name> to Phase <phase_name> of Node <node_id>
	fmt.Printf("Phase Decision of Node %s sends RPC global-abort to Phase Decision of Node %s\n", c.nodeID, participant.NodeID)
	
	conn, err := grpc.Dial(participant.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", participant.Address, err)
	}
	defer conn.Close()

	client := pb.NewTwoPCServiceClient(conn)

	req := &pb.AbortRequest{
		TransactionId: transactionID,
	}

	resp, err := client.Abort(ctx, req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("abort failed: %s", resp.Error)
	}

	return nil
}

// GetTransactionState returns the state of a transaction
func (c *Coordinator) GetTransactionState(transactionID string) (TransactionState, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	txn, exists := c.transactions[transactionID]
	if !exists {
		return Initial, fmt.Errorf("transaction %s not found", transactionID)
	}

	txn.mu.RLock()
	defer txn.mu.RUnlock()
	return txn.State, nil
}

// StartDecision handles StartDecision RPC from voting phase (phase-to-phase gRPC)
// Q2: Decision Phase - receives StartDecision from voting phase
func (c *Coordinator) StartDecision(ctx context.Context, req *pb.StartDecisionRequest) (*pb.StartDecisionResponse, error) {
	// Log: Phase Decision of Node <coordinator> runs RPC StartDecision called by Phase Voting of Node <coordinator>
	fmt.Printf("Phase Decision of Node %s runs RPC StartDecision called by Phase Voting of Node %s\n", c.nodeID, c.nodeID)
	
	c.mu.RLock()
	_, exists := c.transactions[req.TransactionId]
	c.mu.RUnlock()

	if !exists {
		return &pb.StartDecisionResponse{
			Success: false,
			Error:   fmt.Sprintf("transaction %s not found", req.TransactionId),
		}, nil
	}

	// Execute decision phase based on voting results
	if req.AllVotedCommit {
		// All participants voted commit → send global-commit
		if err := c.CommitPhase(ctx, req.TransactionId); err != nil {
			return &pb.StartDecisionResponse{
				Success: false,
				Error:   fmt.Sprintf("commit phase failed: %v", err),
			}, nil
		}
	} else {
		// At least one participant voted abort → send global-abort
		if err := c.AbortPhase(ctx, req.TransactionId); err != nil {
			return &pb.StartDecisionResponse{
				Success: false,
				Error:   fmt.Sprintf("abort phase failed: %v", err),
			}, nil
		}
	}

	return &pb.StartDecisionResponse{
		Success: true,
	}, nil
}

