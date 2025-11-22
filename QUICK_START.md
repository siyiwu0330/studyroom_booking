# Quick Start Guide for TA Demonstration

## 🚀 Running the 5-Node Cluster

```bash
# 1. Clean up any existing containers
docker compose -f docker-compose-grpc.yml down

# 2. Start the 5-node cluster
docker compose -f docker-compose-grpc.yml up -d --build

# 3. Check status (should see 7 containers: 5 nodes + mongo + redis)
docker compose -f docker-compose-grpc.yml ps

# 4. View logs to see Raft election and 2PC logs
docker logs studyroom_booking-app-node1-1 -f
# Press Ctrl+C to exit, then check other nodes:
docker logs studyroom_booking-app-node2-1 | grep -E "(Node|Phase|Leader)" | head -10
```

## 🧪 Running Tests

```bash
# Run all tests (2PC + Raft)
docker compose -f docker-compose.test.yml up --build

# Expected: 8 tests pass (3 2PC + 5 Raft)
# Look for formatted logs in output:
# - "Phase Voting of Node X sends RPC vote-request..."
# - "Node X sends RPC RequestVote to Node Y"
```

## 📍 Key Code Locations

### 2PC Implementation
- **Proto**: `api/proto/twopc.proto` - gRPC service definition
- **Coordinator**: `internal/twopc/coordinator.go` - Voting & Decision phases
- **Participant**: `internal/twopc/participant.go` - Handles vote-request/commit/abort
- **Log Format**: Lines 275, 302, 328 in coordinator.go (sends), lines 64, 119, 163 in participant.go (receives)

### Raft Implementation
- **Proto**: `api/proto/raft.proto` - gRPC service definition
- **Node**: `internal/raft/node.go` - Core Raft logic (election, replication)
- **Client**: `internal/raft/client.go` - Sends RPCs (prints "sends" logs)
- **Server**: `internal/raft/server.go` - Receives RPCs (prints "runs" logs)
- **Timeout Settings**: `internal/raft/node.go:80-81` (heartbeat=1s, election=1.5-3s)
- **Request Forwarding**: `internal/grpc/handler/booking_handler.go:34` (Q4 requirement)

## ✅ Test Coverage

### 2PC Tests (3 tests in `test/twopc_test.go`)
1. Test2PCBasicCommit - Basic commit flow
2. Test2PCAbortOnPrepareFailure - Abort on rejection
3. Test2PCConcurrentTransactions - Concurrent transactions

### Raft Tests (5 tests in `test/raft_test.go`)
1. TestRaftLeaderElection - Basic election
2. TestRaftLeaderTimeout - Leader failure & re-election
3. TestRaftLogReplication - Log replication
4. TestRaftNewNodeJoin - New node joining
5. TestRaftSplitBrainPrevention - Split-brain prevention (5 nodes)

## 📝 For More Details

See [DEMO_GUIDE.md](DEMO_GUIDE.md) for comprehensive documentation.
