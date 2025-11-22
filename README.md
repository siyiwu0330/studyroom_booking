# StudyRoom – Distributed gRPC System with Raft and 2PC

A distributed room booking system implementing **gRPC**, **Raft consensus algorithm**, and **2PC distributed transactions**. Built with Go, MongoDB, Redis, and Docker.

**GitHub Repository**: [https://github.com/siyiwu0330/studyroom_booking](https://github.com/siyiwu0330/studyroom_booking)

---

## 🚀 Features

### 1. gRPC API
- All REST APIs converted to gRPC services
- Efficient binary protocol
- Support for streaming

### 2. Raft Consensus Algorithm
- **Heartbeat Timeout**: 1 second (as required by Q3)
- **Election Timeout**: Random 1.5-3 seconds (as required by Q3)
- **Leader Election**: Automatic leader election and re-election
- **Log Replication**: Leader replicates operation logs to all followers
- **Client Request Forwarding**: Followers forward requests to leader (Q4 requirement)

### 3. 2PC Distributed Transactions
- **Voting Phase (Q1)**: Coordinator sends vote-request, participants respond with vote-commit or vote-abort
- **Decision Phase (Q2)**: Coordinator sends global-commit or global-abort based on votes
- **Proper Log Formatting**: All RPC calls logged in required format

### 4. 5-Node Cluster
- Deployed with 5 nodes as required by assignment Q1-Q4
- Full Docker containerization
- Easy scaling and management

---

## 📋 Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Node 1    │     │   Node 2    │     │   Node 3    │     │   Node 4    │     │   Node 5    │
│  (Leader)   │◄────┤  (Follower) │◄────┤  (Follower) │◄────┤  (Follower) │◄────┤  (Follower) │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
       │                   │                   │                   │                   │
       └───────────────────┴───────────────────┴───────────────────┴───────────────────┘
                                                          │
                                                   ┌──────┴──────┐
                                                   │  MongoDB    │
                                                   │    Redis    │
                                                   └─────────────┘
```

---

## 📦 Prerequisites

- **Docker & Docker Compose** (required)
- **Go 1.24+** (optional, for local development)
- **Protocol Buffers Compiler** (optional, for local development)
  ```bash
  # Ubuntu/Debian
  sudo apt-get install protobuf-compiler
  
  # macOS
  brew install protobuf
  ```

---

## 🛠️ Quick Start

### Option 1: Run 5-Node Cluster (Recommended)

```bash
# 1. Clone the repository
git clone https://github.com/siyiwu0330/studyroom_booking.git
cd studyroom_booking

# 2. Start the 5-node cluster
docker compose -f docker-compose-grpc.yml up -d --build

# 3. Check cluster status
docker compose -f docker-compose-grpc.yml ps

# Expected output: 7 containers running
# - studyroom-mongo
# - studyroom-redis
# - studyroom_booking-app-node1-1 through node5-1
```

### Option 2: Run Tests

```bash
# Run comprehensive test suite (2PC + Raft)
docker compose -f docker-compose.test.yml up --build

# This will:
# - Generate Protocol Buffer code automatically
# - Run all 2PC transaction tests (3 tests)
# - Run all Raft consensus algorithm tests (5 tests)
# - Display test results and summary
```

---

## 📖 Detailed Running Instructions

### Starting the 5-Node Cluster

#### Step 1: Clean Up (if needed)

```bash
# Stop any existing containers
docker compose -f docker-compose-grpc.yml down
docker compose down  # Stop REST API version if running
```

#### Step 2: Build and Start

```bash
# Build and start all services
docker compose -f docker-compose-grpc.yml up -d --build

# This command will:
# 1. Build the gRPC server image (includes proto file generation)
# 2. Start MongoDB container (port 27017)
# 3. Start Redis container (internal network only)
# 4. Start 5 gRPC nodes (node1-node5, ports 50051-50052)
```

#### Step 3: Verify Cluster Status

```bash
# Check all containers are running
docker compose -f docker-compose-grpc.yml ps

# View logs for a specific node
docker logs studyroom_booking-app-node1-1

# View logs for all nodes
for i in 1 2 3 4 5; do
  echo "=== Node $i ==="
  docker logs studyroom_booking-app-node${i}-1 2>&1 | grep -E "(gRPC|Raft|Starting)" | head -3
done
```

#### Step 4: Observe Raft Election

```bash
# Real-time view of node1 logs (watch for election)
docker logs studyroom_booking-app-node1-1 -f

# In another terminal, check which node is leader
for i in 1 2 3 4 5; do
  echo "=== Node $i ==="
  docker logs studyroom_booking-app-node${i}-1 2>&1 | grep -i "leader\|becoming leader" | head -2
done
```

#### Step 5: View Raft RPC Logs

```bash
# View client-side logs (sends RPC)
docker logs studyroom_booking-app-node1-1 2>&1 | grep "sends RPC" | head -5

# View server-side logs (runs RPC)
docker logs studyroom_booking-app-node2-1 2>&1 | grep "runs RPC" | head -5
```

**Expected Log Format**:
- Client: `Node node1 sends RPC RequestVote to Node node2`
- Server: `Node node2 runs RPC RequestVote called by Node node1`

#### Step 6: Stop the Cluster

```bash
# Stop all containers
docker compose -f docker-compose-grpc.yml down

# Stop and remove volumes (clean slate)
docker compose -f docker-compose-grpc.yml down -v
```

---

### Running Tests

#### Docker Testing (Recommended)

```bash
# Run all tests
docker compose -f docker-compose.test.yml up --build

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

#### Local Testing (if Go is installed)

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

**Test Output Examples**:

2PC Logs:
```
Phase Voting of Node test-participant receives RPC vote-request from Phase Voting of Node node1
Phase Voting of Node test-participant sends RPC vote-commit to Phase Voting of Node node1
```

Raft Logs:
```
Node node1 sends RPC RequestVote to Node node2
Node node2 runs RPC RequestVote called by Node node1
```

---

## 🔧 Configuration

### Environment Variables

Each node can be configured via environment variables in `docker-compose-grpc.yml`:

- `NODE_ID`: Node identifier (node1, node2, node3, node4, node5)
- `GRPC_PORT`: gRPC service port (default: 50051)
- `RAFT_PORT`: Raft communication port (default: 50052)
- `PEERS`: List of all nodes in format: `node1:host:port,node2:host:port,...`
- `MONGODB_URI`: MongoDB connection string
- `REDIS_ADDR`: Redis address

### Port Configuration

- **MongoDB**: 27017 (exposed to host)
- **Redis**: Internal network only (no host port mapping)
- **gRPC Nodes**: 50051-50052 (internal network, exposed via Docker network)

---

## 📡 gRPC Services

### AuthService
- `Register`: User registration
- `Login`: User login
- `Logout`: User logout
- `Me`: Get current user information

### BookingService
- `CreateBooking`: Create booking (uses 2PC for distributed coordination)
- `CancelBooking`: Cancel booking
- `JoinWaitlist`: Join waitlist

### SearchService
- `SearchRooms`: Search available rooms

### AdminService
- `CreateRoom`: Create room
- `ListRooms`: List all rooms
- `SetRoomSchedule`: Set room schedule

---

## 🧪 Test Coverage

### 2PC Tests (3 tests)

1. **Test2PCBasicCommit**: Basic commit flow
   - Verifies prepare phase succeeds
   - Verifies vote-commit response
   - Verifies transaction can commit

2. **Test2PCAbortOnPrepareFailure**: Abort when participant rejects
   - Verifies vote-abort response
   - Verifies transaction is aborted

3. **Test2PCConcurrentTransactions**: Concurrent transactions
   - Verifies transactions don't interfere
   - Verifies independent transaction tracking

### Raft Tests (5 tests)

1. **TestRaftLeaderElection**: Basic leader election
2. **TestRaftLeaderTimeout**: Leader failure and re-election
3. **TestRaftLogReplication**: Log replication from leader
4. **TestRaftNewNodeJoin**: New node joining cluster
5. **TestRaftSplitBrainPrevention**: Split-brain prevention with 5 nodes

**All 8 tests pass successfully.** See [TEST_REPORT.md](TEST_REPORT.md) for detailed test results and analysis.

---

## 📁 Project Structure

```
studyroom_booking/
├── api/
│   └── proto/                    # Protocol Buffer definitions
│       ├── studyroom.proto      # Business services (Auth, Booking, Search, Admin)
│       ├── raft.proto           # Raft consensus protocol
│       └── twopc.proto          # 2PC transaction protocol
│
├── cmd/
│   ├── server/                  # REST API server (original)
│   └── server-grpc/             # gRPC distributed server
│       └── main.go              # Entry point
│
├── internal/
│   ├── raft/                    # Raft consensus implementation
│   │   ├── node.go             # Core Raft node logic
│   │   ├── client.go           # Raft gRPC client
│   │   └── server.go           # Raft gRPC server
│   │
│   ├── twopc/                   # 2PC distributed transaction
│   │   ├── coordinator.go     # 2PC coordinator (voting & decision phases)
│   │   ├── participant.go     # 2PC participant
│   │   └── server.go           # 2PC gRPC server
│   │
│   └── grpc/
│       └── handler/             # gRPC business handlers
│           ├── auth_handler.go
│           ├── booking_handler.go  # Integrates 2PC and request forwarding
│           ├── search_handler.go
│           └── admin_handler.go
│
├── test/                        # Test suite
│   ├── twopc_test.go           # 2PC tests (3 tests)
│   ├── raft_test.go            # Raft tests (5 tests)
│   ├── integration_test.go     # Integration tests
│   └── docker_test.sh          # Docker test script
│
├── docker-compose-grpc.yml     # 5-node cluster configuration
├── docker-compose.test.yml      # Test environment configuration
├── Dockerfile-grpc              # gRPC server image
├── Dockerfile.test              # Test environment image
├── Makefile                     # Build scripts
└── README.md                    # This file
```

---

## 🔍 Key Implementation Details

### 2PC Implementation

**Voting Phase (Q1)**:
- Coordinator: `internal/twopc/coordinator.go:275` - Sends vote-request
- Participant: `internal/twopc/participant.go:64` - Receives vote-request, responds with vote-commit/abort

**Decision Phase (Q2)**:
- Coordinator: `internal/twopc/coordinator.go:302` - Sends global-commit
- Coordinator: `internal/twopc/coordinator.go:328` - Sends global-abort
- Participant: `internal/twopc/participant.go:119` - Receives global-commit
- Participant: `internal/twopc/participant.go:163` - Receives global-abort

### Raft Implementation

**Leader Election (Q3)**:
- Timeout settings: `internal/raft/node.go:80-81`
  - Heartbeat: 1 second
  - Election: 1.5-3 seconds (randomized)
- Election logic: `internal/raft/node.go:218` - `startElection()`
- Client logs: `internal/raft/client.go:37` - `RequestVote()`
- Server logs: `internal/raft/server.go:21` - `RequestVote()`

**Log Replication (Q4)**:
- Leader append: `internal/raft/node.go:150` - `AppendCommand()`
- Client logs: `internal/raft/client.go:57` - `AppendEntries()`
- Server logs: `internal/raft/server.go:36` - `AppendEntries()`
- Request forwarding: `internal/grpc/handler/booking_handler.go:34` - `forwardToLeader()`

---

## 📊 Performance

### Raft Performance
- **Election Time**: 1.5-3 seconds (randomized, as required by Q3)
- **Heartbeat Interval**: 1 second (as required by Q3)
- **Log Replication Latency**: 50-200ms per entry

### 2PC Performance
- **Prepare Phase**: < 10ms (local)
- **Commit Phase**: < 10ms (local)
- **Network Latency**: Depends on network conditions

---

## ⚠️ Important Notes

1. **Raft Node Count**: Uses 5 nodes as required by assignment Q1-Q4. Odd number ensures majority voting.

2. **Network Partitions**: In case of network partitions, only the majority partition can continue to serve.

3. **2PC Blocking**: If a participant fails during prepare phase, the transaction may block. Future improvements should include timeout mechanisms.

4. **Data Consistency**: All write operations must go through the Leader node.

5. **Redis Port**: Redis runs in Docker network only (no host port mapping) to avoid port conflicts.

---

## 🐛 Troubleshooting

### Port Already in Use

If you see "address already in use" errors:

```bash
# Check what's using the port
sudo lsof -i :6379  # Redis
sudo lsof -i :27017  # MongoDB

# Stop conflicting services or change ports in docker-compose-grpc.yml
```

### Containers Not Starting

```bash
# Check container logs
docker logs studyroom_booking-app-node1-1

# Restart containers
docker compose -f docker-compose-grpc.yml restart

# Rebuild if needed
docker compose -f docker-compose-grpc.yml up -d --build --force-recreate
```

### Tests Failing

```bash
# Ensure proto files are generated
make proto

# Run tests with verbose output
go test ./test -v

# Check Docker test logs
docker compose -f docker-compose.test.yml up --build
```

---

## 📚 Documentation

- **Test Report**: See [TEST_REPORT.md](TEST_REPORT.md) for detailed test results, implementation analysis, and performance observations.

---

## 🔮 Future Improvements

- [ ] Implement Raft snapshots for large logs
- [ ] Add 2PC timeout and recovery mechanisms
- [ ] Implement client-side load balancing
- [ ] Add monitoring and metrics collection
- [ ] Support dynamic node joining/removal
- [ ] Add TLS for gRPC security

---

## 📄 License

This project is part of a distributed systems assignment.

---

## 👥 Authors

See GitHub repository for contributors.

---

**For detailed test results and analysis, see [TEST_REPORT.md](TEST_REPORT.md).**
