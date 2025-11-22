# StudyRoom gRPC Distributed System - Demo Guide

This guide provides step-by-step instructions for demonstrating the project to TAs, including how to run the system, test it, and understand the implementation.

---

## 📋 Table of Contents

1. [Quick Start - Running the 5-Node Cluster](#1-quick-start---running-the-5-node-cluster)
2. [Running Tests](#2-running-tests)
3. [2PC Implementation Details](#3-2pc-implementation-details)
4. [Raft Implementation Details](#4-raft-implementation-details)
5. [Test Setup and Coverage](#5-test-setup-and-coverage)

---

## 1. Quick Start - Running the 5-Node Cluster

### Step 1: Clean Up Any Existing Containers

```bash
# Stop and remove any existing containers
docker compose -f docker-compose-grpc.yml down
docker compose down  # Stop REST API version if running
```

### Step 2: Start the 5-Node Cluster

```bash
# Build and start all 5 nodes + MongoDB + Redis
docker compose -f docker-compose-grpc.yml up -d --build

# This will:
# - Build the gRPC server image for all 5 nodes
# - Start MongoDB (port 27017)
# - Start Redis (port 6379)
# - Start 5 gRPC nodes (node1-node5)
```

### Step 3: Check Cluster Status

```bash
# Check if all containers are running
docker compose -f docker-compose-grpc.yml ps

# Expected output: 7 containers (5 nodes + mongo + redis)
# All should show "Up" status
```

### Step 4: View Node Logs

```bash
# View logs for node1 (usually becomes leader)
docker logs studyroom_booking-app-node1-1 -f

# In another terminal, view logs for other nodes
docker logs studyroom_booking-app-node2-1 -f
docker logs studyroom_booking-app-node3-1 -f

# Look for:
# - "gRPC server listening on :50051"
# - "Raft node X started"
# - "Node X sends RPC RequestVote to Node Y" (Raft election)
# - "Node X runs RPC RequestVote called by Node Y" (Raft server)
# - "Phase Voting of Node X sends RPC vote-request..." (2PC)
```

### Step 5: Verify Leader Election

```bash
# Check which node is the leader (look for "Leader" in logs)
docker logs studyroom_booking-app-node1-1 2>&1 | grep -i leader
docker logs studyroom_booking-app-node2-1 2>&1 | grep -i leader
docker logs studyroom_booking-app-node3-1 2>&1 | grep -i leader

# Or check all nodes at once
for i in 1 2 3 4 5; do
  echo "=== Node $i ==="
  docker logs studyroom_booking-app-node${i}-1 2>&1 | grep -E "(Leader|Follower|Candidate)" | head -3
done
```

### Step 6: Stop the Cluster

```bash
# Stop all containers
docker compose -f docker-compose-grpc.yml down

# Or stop and remove volumes (clean slate)
docker compose -f docker-compose-grpc.yml down -v
```

---

## 2. Running Tests

### Option A: Run All Tests with Docker (Recommended)

```bash
# Run comprehensive test suite
docker compose -f docker-compose.test.yml up --build

# This will:
# 1. Generate Protocol Buffer code
# 2. Run all 2PC tests (3 tests)
# 3. Run all Raft tests (5 tests)
# 4. Display test results

# Expected output:
# - Test2PCBasicCommit: PASS
# - Test2PCAbortOnPrepareFailure: PASS
# - Test2PCConcurrentTransactions: PASS
# - TestRaftLeaderElection: PASS
# - TestRaftLeaderTimeout: PASS
# - TestRaftLogReplication: PASS
# - TestRaftNewNodeJoin: PASS
# - TestRaftSplitBrainPrevention: PASS
```

### Option B: Run Tests Locally (if Go and protoc are installed)

```bash
# Generate proto files first
make proto

# Run all tests
go test ./test -v

# Run only 2PC tests
go test ./test -v -run Test2PC

# Run only Raft tests
go test ./test -v -run TestRaft

# Run specific test
go test ./test -v -run Test2PCBasicCommit
```

### View Test Output with Logs

The tests will show formatted logs as required by the assignment:

**2PC Logs:**
```
Phase Voting of Node test-participant receives RPC vote-request from Phase Voting of Node node1
Phase Voting of Node test-participant sends RPC vote-commit to Phase Voting of Node node1
```

**Raft Logs:**
```
Node node1 sends RPC RequestVote to Node node2
Node node2 runs RPC RequestVote called by Node node1
```

---

## 3. 2PC Implementation Details

### 3.1 Architecture Overview

2PC (Two-Phase Commit) is implemented in two phases:
- **Q1: Voting Phase** - Coordinator sends vote-request, participants respond with vote-commit or vote-abort
- **Q2: Decision Phase** - Coordinator sends global-commit or global-abort based on votes

### 3.2 Key Code Locations

#### Protocol Definition
- **File**: `api/proto/twopc.proto`
- **Purpose**: Defines gRPC service and messages for 2PC
- **Key RPCs**:
  - `Prepare` (vote-request in Q1)
  - `Commit` (global-commit in Q2)
  - `Abort` (global-abort in Q2)

#### Coordinator (Voting & Decision Phase)
- **File**: `internal/twopc/coordinator.go`
- **Key Functions**:
  - `PreparePhase()` (line ~82): Sends vote-request to all participants
    - Calls `sendPrepare()` which prints: `Phase Voting of Node X sends RPC vote-request...`
  - `CommitPhase()` (line ~154): Sends global-commit after all vote-commit
    - Calls `sendCommit()` which prints: `Phase Decision of Node X sends RPC global-commit...`
  - `AbortPhase()` (line ~204): Sends global-abort if any vote-abort
    - Calls `sendAbort()` which prints: `Phase Decision of Node X sends RPC global-abort...`

#### Participant (Voting & Decision Phase)
- **File**: `internal/twopc/participant.go`
- **Key Functions**:
  - `Prepare()` (line ~64): Receives vote-request, responds with vote-commit or vote-abort
    - Prints: `Phase Voting of Node X receives RPC vote-request...`
    - Prints: `Phase Voting of Node X sends RPC vote-commit/vote-abort...`
  - `Commit()` (line ~119): Receives global-commit, executes transaction
    - Prints: `Phase Decision of Node X receives RPC global-commit...`
  - `Abort()` (line ~163): Receives global-abort, rolls back transaction
    - Prints: `Phase Decision of Node X receives RPC global-abort...`

#### gRPC Server for 2PC
- **File**: `internal/twopc/server.go`
- **Purpose**: Implements gRPC server that delegates to ParticipantNode

#### Integration with Booking Handler
- **File**: `internal/grpc/handler/booking_handler.go`
- **Function**: `createBookingWith2PC()` (line ~64)
- **Purpose**: Uses 2PC coordinator to create bookings atomically across nodes

### 3.3 2PC Flow Example

```
1. Client requests booking via gRPC
   ↓
2. BookingHandler.createBookingWith2PC()
   ↓
3. Coordinator.PreparePhase() [Q1: Voting]
   - Coordinator sends vote-request to all participants
   - Each participant responds: vote-commit or vote-abort
   ↓
4. If all vote-commit:
   - Coordinator.CommitPhase() [Q2: Decision]
   - Coordinator sends global-commit to all participants
   - Participants execute the transaction
   ↓
5. If any vote-abort:
   - Coordinator.AbortPhase() [Q2: Decision]
   - Coordinator sends global-abort to all participants
   - Participants roll back the transaction
```

---

## 4. Raft Implementation Details

### 4.1 Architecture Overview

Raft consensus algorithm implements:
- **Q3: Leader Election** - Heartbeat timeout (1s), Election timeout (1.5-3s random)
- **Q4: Log Replication** - Leader replicates logs to followers, client request forwarding

### 4.2 Key Code Locations

#### Protocol Definition
- **File**: `api/proto/raft.proto`
- **Purpose**: Defines gRPC service and messages for Raft
- **Key RPCs**:
  - `RequestVote`: Used during election
  - `AppendEntries`: Used for log replication and heartbeats
  - `Heartbeat`: Empty AppendEntries for heartbeat

#### Core Raft Node
- **File**: `internal/raft/node.go`
- **Key Functions**:
  - `NewNode()` (line ~67): Creates node with timeout settings
    - `heartbeatInterval: 1 * time.Second` (Q3 requirement)
    - `electionTimeout: 1500 * time.Millisecond` (base 1.5s)
  - `resetElectionTimer()` (line ~192): Randomizes election timeout (1.5-3s)
    ```go
    randomOffset := time.Duration(rand.Intn(1500)) * time.Millisecond
    timeout := n.electionTimeout + randomOffset  // 1.5s + 0-1.5s = 1.5-3s
    ```
  - `startElection()` (line ~218): Q3 - Candidate requests votes
    - Increments term
    - Votes for self
    - Sends RequestVote RPCs to all peers
  - `becomeLeader()` (line ~291): Transitions to leader, starts heartbeats
  - `sendHeartbeats()` (line ~326): Leader sends periodic heartbeats (1s interval)
  - `AppendCommand()` (line ~150): Q4 - Leader appends command, replicates to followers
  - `HandleAppendEntries()` (line ~434): Q4 - Follower receives and applies log entries

#### Raft gRPC Client
- **File**: `internal/raft/client.go`
- **Key Functions**:
  - `RequestVote()` (line ~37): Sends vote request
    - Prints: `Node X sends RPC RequestVote to Node Y`
  - `AppendEntries()` (line ~57): Sends log entries
    - Prints: `Node X sends RPC AppendEntries to Node Y`
  - `Heartbeat()` (line ~88): Sends heartbeat
    - Prints: `Node X sends RPC Heartbeat to Node Y`

#### Raft gRPC Server
- **File**: `internal/raft/server.go`
- **Key Functions**:
  - `RequestVote()` (line ~21): Handles vote request
    - Prints: `Node X runs RPC RequestVote called by Node Y`
  - `AppendEntries()` (line ~36): Handles log replication
    - Prints: `Node X runs RPC AppendEntries called by Node Y`
  - `Heartbeat()` (line ~62): Handles heartbeat
    - Prints: `Node X runs RPC Heartbeat called by Node Y`

#### Client Request Forwarding (Q4)
- **File**: `internal/grpc/handler/booking_handler.go`
- **Function**: `forwardToLeader()` (line ~189)
- **Purpose**: If client connects to follower, forward request to leader
- **Implementation**:
  ```go
  if h.raftNode != nil && !h.raftNode.IsLeader() {
      return h.forwardToLeader(ctx, ...)
  }
  ```

### 4.3 Raft Flow Example

#### Leader Election (Q3)
```
1. All nodes start as Followers
   ↓
2. Follower doesn't receive heartbeat within election timeout (1.5-3s random)
   ↓
3. Follower becomes Candidate:
   - Increments term
   - Votes for self
   - Sends RequestVote to all peers
   ↓
4. If receives majority votes:
   - Becomes Leader
   - Starts sending heartbeats every 1 second
   ↓
5. If another candidate wins or timeout:
   - Returns to Follower
```

#### Log Replication (Q4)
```
1. Client sends request to any node
   ↓
2. If node is Follower:
   - Forwards request to Leader (forwardToLeader)
   ↓
3. Leader receives request:
   - Appends command to log
   - Sends AppendEntries to all Followers
   ↓
4. Followers receive AppendEntries:
   - Append to log
   - Return ACK
   ↓
5. Leader receives majority ACKs:
   - Commits log entry
   - Applies to state machine
   - Returns result to client
```

---

## 5. Test Setup and Coverage

### 5.1 Test Structure

Tests are located in `test/` directory:
- `test/twopc_test.go`: 2PC transaction tests
- `test/raft_test.go`: Raft consensus tests
- `test/integration_test.go`: Integration tests (requires running servers)
- `test/docker_test.sh`: Docker test orchestration script

### 5.2 Test Configuration

#### Docker Test Setup
- **File**: `docker-compose.test.yml`
- **Purpose**: Provides isolated test environment
- **Steps**:
  1. Builds test image with all dependencies
  2. Generates Protocol Buffer code
  3. Runs all test suites
  4. Displays results

#### Test Script
- **File**: `test/docker_test.sh`
- **Purpose**: Orchestrates test execution in Docker
- **Steps**:
  1. Installs dependencies
  2. Generates proto files
  3. Runs 2PC tests
  4. Runs Raft tests
  5. Displays summary

### 5.3 Test Coverage

#### 2PC Tests (3 tests)

1. **Test2PCBasicCommit** (`test/twopc_test.go:16`)
   - **Tests**: Basic commit flow
   - **Verifies**:
     - Prepare phase succeeds
     - Participant responds with vote-commit
     - Transaction can commit
   - **Log Output**: 
     ```
     Phase Voting of Node test-participant receives RPC vote-request...
     Phase Voting of Node test-participant sends RPC vote-commit...
     ```

2. **Test2PCAbortOnPrepareFailure** (`test/twopc_test.go:77`)
   - **Tests**: Abort when participant rejects
   - **Verifies**:
     - Prepare phase fails
     - Participant responds with vote-abort
     - Transaction is aborted
   - **Log Output**:
     ```
     Phase Voting of Node test-participant receives RPC vote-request...
     Phase Voting of Node test-participant sends RPC vote-abort...
     ```

3. **Test2PCConcurrentTransactions** (`test/twopc_test.go:113`)
   - **Tests**: Multiple concurrent transactions
   - **Verifies**:
     - Transactions don't interfere
     - Each transaction tracked independently
     - No race conditions

#### Raft Tests (5 tests)

1. **TestRaftLeaderElection** (`test/raft_test.go:13`)
   - **Tests**: Basic leader election
   - **Verifies**:
     - Nodes start as Followers
     - Election mechanism works
     - Valid states maintained

2. **TestRaftLeaderTimeout** (`test/raft_test.go:62`)
   - **Tests**: Leader failure and re-election
   - **Verifies**:
     - Leader failure detected
     - New leader elected
     - Cluster remains functional

3. **TestRaftLogReplication** (`test/raft_test.go:110`)
   - **Tests**: Log replication from leader
   - **Verifies**:
     - Only leader can append commands
     - Log replication works
     - Commands applied correctly

4. **TestRaftNewNodeJoin** (`test/raft_test.go:145`)
   - **Tests**: New node joining cluster
   - **Verifies**:
     - New node can join
     - Cluster stability maintained
     - Leader continues functioning

5. **TestRaftSplitBrainPrevention** (`test/raft_test.go:200`)
   - **Tests**: Split-brain prevention with 5 nodes
   - **Verifies**:
     - Only one leader exists
     - Majority voting prevents conflicts
     - No split-brain scenarios

### 5.4 Running Specific Tests

```bash
# Run only 2PC tests
go test ./test -v -run Test2PC

# Run only Raft tests
go test ./test -v -run TestRaft

# Run specific test
go test ./test -v -run Test2PCBasicCommit
go test ./test -v -run TestRaftLeaderElection

# Run with coverage
go test ./test -cover
```

### 5.5 Expected Test Output

When running tests, you should see:

**2PC Test Output:**
```
=== RUN   Test2PCBasicCommit
Phase Voting of Node test-participant receives RPC vote-request from Phase Voting of Node node1
Phase Voting of Node test-participant sends RPC vote-commit to Phase Voting of Node node1
--- PASS: Test2PCBasicCommit (0.00s)
```

**Raft Test Output:**
```
=== RUN   TestRaftLeaderElection
[Raft node1] Starting node
[Raft node2] Starting node
[Raft node3] Starting node
--- PASS: TestRaftLeaderElection (0.20s)
```

---

## 📝 Quick Reference Commands

### Start 5-Node Cluster
```bash
docker compose -f docker-compose-grpc.yml up -d --build
```

### View Logs
```bash
docker logs studyroom_booking-app-node1-1 -f
```

### Run Tests
```bash
docker compose -f docker-compose.test.yml up --build
```

### Stop Everything
```bash
docker compose -f docker-compose-grpc.yml down
```

---

## 🎯 Key Points for TA Demonstration

1. **5-Node Configuration**: All Q1-Q4 require at least 5 nodes - configured in `docker-compose-grpc.yml`

2. **Timeout Settings**: 
   - Heartbeat: 1 second (Q3 requirement)
   - Election: 1.5-3 seconds random (Q3 requirement)
   - See `internal/raft/node.go:80-81`

3. **Log Format Compliance**:
   - 2PC: `Phase <phase> of Node <id> sends/receives RPC <name>`
   - Raft: `Node <id> sends/runs RPC <name>`
   - All logs printed in client/server code

4. **Client Request Forwarding**: 
   - Implemented in `internal/grpc/handler/booking_handler.go:34`
   - Followers forward to leader automatically

5. **Test Coverage**: 
   - 8 tests total (3 2PC + 5 Raft)
   - All tests pass
   - Logs verified in test output

---

**End of Demo Guide**

