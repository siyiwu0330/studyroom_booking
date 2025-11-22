package raft

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

// NodeState represents the state of a Raft node
type NodeState int

const (
	Follower NodeState = iota
	Candidate
	Leader
)

// LogEntry represents a single log entry
type LogEntry struct {
	Term    int
	Index   int
	Command string
}

// Node represents a Raft node
type Node struct {
	mu sync.RWMutex

	// Persistent state
	id      string
	address string
	state   NodeState
	term    int
	votedFor string
	log     []LogEntry

	// Volatile state
	commitIndex int
	lastApplied int

	// Leader state
	nextIndex  map[string]int
	matchIndex map[string]int

	// Configuration
	peers      map[string]string // node_id -> address
	electionTimeout time.Duration
	heartbeatInterval time.Duration

	// Channels
	electionTimer  *time.Timer
	heartbeatTimer *time.Timer
	stopCh         chan struct{}

	// Callbacks
	applyFunc func(command string) error

	// gRPC clients for peers (lazy initialization)
	clients map[string]*RaftClient
	clientMu sync.RWMutex
}

// NewNode creates a new Raft node
func NewNode(id, address string, peers map[string]string) *Node {
	node := &Node{
		id:               id,
		address:          address,
		state:            Follower,
		term:             0,
		votedFor:         "",
		log:              []LogEntry{{Term: 0, Index: 0, Command: ""}}, // dummy entry at index 0
		commitIndex:      0,
		lastApplied:      0,
		nextIndex:        make(map[string]int),
		matchIndex:       make(map[string]int),
		peers:            peers,
		electionTimeout:  1500 * time.Millisecond, // Base 1.5 seconds
		heartbeatInterval: 1 * time.Second,        // 1 second as required
		stopCh:           make(chan struct{}),
		clients:          make(map[string]*RaftClient),
	}

	// Initialize nextIndex and matchIndex for each peer
	for peerID := range peers {
		node.nextIndex[peerID] = len(node.log)
		node.matchIndex[peerID] = 0
	}

	return node
}

// SetApplyFunc sets the function to apply committed log entries
func (n *Node) SetApplyFunc(fn func(command string) error) {
	n.applyFunc = fn
}

// Start starts the Raft node
func (n *Node) Start() {
	log.Printf("[Raft %s] Starting node", n.id)
	n.resetElectionTimer()
	go n.run()
}

// Stop stops the Raft node
func (n *Node) Stop() {
	close(n.stopCh)
	if n.electionTimer != nil {
		n.electionTimer.Stop()
	}
	if n.heartbeatTimer != nil {
		n.heartbeatTimer.Stop()
	}

	// Close all gRPC clients
	n.clientMu.Lock()
	for _, client := range n.clients {
		client.Close()
	}
	n.clients = make(map[string]*RaftClient)
	n.clientMu.Unlock()
}

// GetState returns the current state of the node
func (n *Node) GetState() (NodeState, int) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.state, n.term
}

// IsLeader returns true if the node is the leader
func (n *Node) IsLeader() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.state == Leader
}

// GetID returns the node ID
func (n *Node) GetID() string {
	return n.id
}

// GetAddress returns the node address
func (n *Node) GetAddress() string {
	return n.address
}

// AppendCommand appends a command to the log (only if leader)
func (n *Node) AppendCommand(command string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.state != Leader {
		return fmt.Errorf("not leader")
	}

	entry := LogEntry{
		Term:    n.term,
		Index:   len(n.log),
		Command: command,
	}
	n.log = append(n.log, entry)

	// Replicate to followers asynchronously
	go n.replicateLog()

	return nil
}

// run is the main event loop
func (n *Node) run() {
	for {
		select {
		case <-n.stopCh:
			return
		case <-n.electionTimer.C:
			if n.electionTimer != nil {
				n.handleElectionTimeout()
			}
		case <-n.heartbeatTimer.C:
			if n.heartbeatTimer != nil {
				n.handleHeartbeatTimeout()
			}
		}
	}
}

// resetElectionTimer resets the election timer with random timeout
// Election timeout should be randomly chosen from [1.5 seconds, 3 seconds]
func (n *Node) resetElectionTimer() {
	if n.electionTimer != nil {
		n.electionTimer.Stop()
	}
	// Random timeout between 1.5s and 3.0s (1500ms + random 0-1500ms)
	randomOffset := time.Duration(rand.Intn(1500)) * time.Millisecond
	timeout := n.electionTimeout + randomOffset
	n.electionTimer = time.NewTimer(timeout)
	if n.heartbeatTimer == nil {
		// Initialize heartbeat timer (will be used when becoming leader)
		n.heartbeatTimer = time.NewTimer(24 * time.Hour) // Long timeout, will be reset when needed
		n.heartbeatTimer.Stop()
	}
}

// handleElectionTimeout handles election timeout
func (n *Node) handleElectionTimeout() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.state == Leader {
		return
	}

	log.Printf("[Raft %s] Election timeout, starting election", n.id)
	n.startElection()
}

// startElection starts a new election
func (n *Node) startElection() {
	n.state = Candidate
	n.term++
	n.votedFor = n.id
	votes := 1 // vote for self

	log.Printf("[Raft %s] Starting election for term %d", n.id, n.term)

	// Request votes from all peers
	var wg sync.WaitGroup
	var mu sync.Mutex

	for peerID, peerAddr := range n.peers {
		wg.Add(1)
		go func(id, addr string) {
			defer wg.Done()
			lastLogIndex := len(n.log) - 1
			lastLogTerm := 0
			if lastLogIndex >= 0 {
				lastLogTerm = n.log[lastLogIndex].Term
			}

			// Use gRPC client to request vote
			client, err := n.getClient(id, addr)
			if err != nil {
				log.Printf("[Raft %s] Error getting client for %s: %v", n.id, id, err)
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			voted, respTerm, err := client.RequestVote(ctx, n.term, n.id, lastLogIndex, lastLogTerm, id)
			cancel()

			if err != nil {
				log.Printf("[Raft %s] Error requesting vote from %s: %v", n.id, id, err)
				return
			}

			// Update term if we see a higher term
			if respTerm > n.term {
				n.mu.Lock()
				n.term = respTerm
				n.state = Follower
				n.votedFor = ""
				n.mu.Unlock()
				return
			}

			if voted {
				mu.Lock()
				votes++
				mu.Unlock()
			}
		}(peerID, peerAddr)
	}

	wg.Wait()

	// Check if we got majority
	majority := len(n.peers)/2 + 1
	if votes >= majority {
		log.Printf("[Raft %s] Won election with %d votes, becoming leader", n.id, votes)
		n.becomeLeader()
	} else {
		log.Printf("[Raft %s] Lost election with %d votes", n.id, votes)
		n.state = Follower
		n.resetElectionTimer()
	}
}

// becomeLeader transitions to leader state
func (n *Node) becomeLeader() {
	n.state = Leader
	n.votedFor = ""

	// Initialize nextIndex and matchIndex
	for peerID := range n.peers {
		n.nextIndex[peerID] = len(n.log)
		n.matchIndex[peerID] = 0
	}

	// Start sending heartbeats
	n.resetHeartbeatTimer()
	n.sendHeartbeats()
}

// resetHeartbeatTimer resets the heartbeat timer
func (n *Node) resetHeartbeatTimer() {
	if n.heartbeatTimer != nil {
		n.heartbeatTimer.Stop()
	}
	n.heartbeatTimer = time.NewTimer(n.heartbeatInterval)
}

// handleHeartbeatTimeout handles heartbeat timeout (only for leader)
func (n *Node) handleHeartbeatTimeout() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.state == Leader {
		n.sendHeartbeats()
		n.resetHeartbeatTimer()
	}
}

// sendHeartbeats sends heartbeats to all followers
func (n *Node) sendHeartbeats() {
	for peerID, peerAddr := range n.peers {
		go func(id, addr string) {
			n.mu.RLock()
			term := n.term
			n.mu.RUnlock()
			n.sendAppendEntries(id, addr, true, []LogEntry{}, 0, 0, 0, term)
		}(peerID, peerAddr)
	}
}

// replicateLog replicates log entries to followers
func (n *Node) replicateLog() {
	for peerID, peerAddr := range n.peers {
		go func(id, addr string) {
			n.mu.RLock()
			nextIdx := n.nextIndex[id]
			prevLogIndex := nextIdx - 1
			prevLogTerm := 0
			if prevLogIndex >= 0 && prevLogIndex < len(n.log) {
				prevLogTerm = n.log[prevLogIndex].Term
			}
			entries := n.log[nextIdx:]
			leaderCommit := n.commitIndex
			term := n.term
			n.mu.RUnlock()

			success := n.sendAppendEntries(id, addr, false, entries, prevLogIndex, prevLogTerm, leaderCommit, term)
			if success {
				n.mu.Lock()
				n.nextIndex[id] = len(n.log)
				n.matchIndex[id] = len(n.log) - 1
				n.updateCommitIndex()
				n.mu.Unlock()
			} else {
				n.mu.Lock()
				if n.nextIndex[id] > 0 {
					n.nextIndex[id]--
				}
				n.mu.Unlock()
			}
		}(peerID, peerAddr)
	}
}

// updateCommitIndex updates the commit index
func (n *Node) updateCommitIndex() {
	for N := len(n.log) - 1; N > n.commitIndex; N-- {
		count := 1 // leader
		for peerID := range n.peers {
			if n.matchIndex[peerID] >= N {
				count++
			}
		}
		majority := len(n.peers)/2 + 1
		if count >= majority && n.log[N].Term == n.term {
			n.commitIndex = N
			break
		}
	}

	// Apply committed entries
	for n.lastApplied < n.commitIndex {
		n.lastApplied++
		if n.applyFunc != nil && n.lastApplied < len(n.log) {
			if err := n.applyFunc(n.log[n.lastApplied].Command); err != nil {
				log.Printf("[Raft %s] Error applying command: %v", n.id, err)
			}
		}
	}
}

// HandleRequestVote handles vote request from candidate
func (n *Node) HandleRequestVote(term int, candidateID string, lastLogIndex, lastLogTerm int) (bool, int) {
	n.mu.Lock()
	defer n.mu.Unlock()

	currentTerm := n.term

	if term < currentTerm {
		return false, currentTerm
	}

	if term > currentTerm {
		n.term = term
		n.state = Follower
		n.votedFor = ""
	}

	// Check if candidate's log is at least as up-to-date
	lastLogIdx := len(n.log) - 1
	lastLogT := 0
	if lastLogIdx >= 0 {
		lastLogT = n.log[lastLogIdx].Term
	}

	voteGranted := false
	if (n.votedFor == "" || n.votedFor == candidateID) &&
		(lastLogTerm > lastLogT || (lastLogTerm == lastLogT && lastLogIndex >= lastLogIdx)) {
		voteGranted = true
		n.votedFor = candidateID
		n.resetElectionTimer()
	}

	return voteGranted, n.term
}

// HandleAppendEntries handles append entries from leader
func (n *Node) HandleAppendEntries(term int, leaderID string, prevLogIndex, prevLogTerm int, entries []LogEntry, leaderCommit int) (bool, int) {
	n.mu.Lock()
	defer n.mu.Unlock()

	currentTerm := n.term

	if term < currentTerm {
		return false, currentTerm
	}

	// Reset election timer on valid append entries
	n.resetElectionTimer()

	if term > currentTerm {
		n.term = term
		n.state = Follower
		n.votedFor = ""
	}

	// Check if previous log entry matches
	if prevLogIndex >= len(n.log) || (prevLogIndex > 0 && n.log[prevLogIndex].Term != prevLogTerm) {
		return false, n.term
	}

	// Append new entries
	if len(entries) > 0 {
		// Delete conflicting entries
		if prevLogIndex+1 < len(n.log) {
			n.log = n.log[:prevLogIndex+1]
		}
		// Append new entries
		n.log = append(n.log, entries...)
	}

	// Update commit index
	if leaderCommit > n.commitIndex {
		n.commitIndex = min(leaderCommit, len(n.log)-1)
	}

	// Apply committed entries
	for n.lastApplied < n.commitIndex {
		n.lastApplied++
		if n.applyFunc != nil && n.lastApplied < len(n.log) {
			if err := n.applyFunc(n.log[n.lastApplied].Command); err != nil {
				log.Printf("[Raft %s] Error applying command: %v", n.id, err)
			}
		}
	}

	return true, n.term
}

// Helper functions
func (n *Node) requestVote(peerID, peerAddr string, term, lastLogIndex, lastLogTerm int) (bool, error) {
	// This would make an RPC call in real implementation
	// For now, we'll use a mock implementation
	// In production, this would call the RaftService.RequestVote gRPC method
	return false, fmt.Errorf("not implemented: use gRPC client")
}

func (n *Node) sendAppendEntries(peerID, peerAddr string, isHeartbeat bool, entries []LogEntry, prevLogIndex, prevLogTerm, leaderCommit, term int) bool {
	client, err := n.getClient(peerID, peerAddr)
	if err != nil {
		log.Printf("[Raft %s] Error getting client for %s: %v", n.id, peerID, err)
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if isHeartbeat {
		success, respTerm, err := client.Heartbeat(ctx, term, n.id, peerID)
		if err != nil {
			return false
		}
		if respTerm > term {
			n.mu.Lock()
			if respTerm > n.term {
				n.term = respTerm
				n.state = Follower
				n.votedFor = ""
			}
			n.mu.Unlock()
		}
		return success
	}

	success, respTerm, err := client.AppendEntries(ctx, term, n.id, prevLogIndex, prevLogTerm, entries, leaderCommit, peerID)
	if err != nil {
		return false
	}
	if respTerm > term {
		n.mu.Lock()
		if respTerm > n.term {
			n.term = respTerm
			n.state = Follower
			n.votedFor = ""
		}
		n.mu.Unlock()
	}
	return success
}

// getClient gets or creates a gRPC client for a peer
func (n *Node) getClient(peerID, peerAddr string) (*RaftClient, error) {
	n.clientMu.RLock()
	if client, exists := n.clients[peerID]; exists {
		n.clientMu.RUnlock()
		return client, nil
	}
	n.clientMu.RUnlock()

	n.clientMu.Lock()
	defer n.clientMu.Unlock()

	// Double check
	if client, exists := n.clients[peerID]; exists {
		return client, nil
	}

	client, err := NewRaftClient(peerAddr)
	if err != nil {
		return nil, err
	}

	n.clients[peerID] = client
	return client, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

