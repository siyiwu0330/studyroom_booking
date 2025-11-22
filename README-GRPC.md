# StudyRoom - gRPC Distributed Version

This is the distributed gRPC version of the StudyRoom project, featuring the following advanced capabilities:

## 🚀 New Features

### 1. gRPC API
- All REST APIs have been converted to gRPC services
- More efficient binary protocol
- Support for streaming

### 2. Raft Consensus Algorithm
- **Heartbeat Timeout**: Leader periodically sends heartbeats to maintain leadership
- **Leader Re-election**: Automatically elects a new Leader when the current Leader fails
- **Log Replication**: Leader replicates operation logs to all Followers

### 3. 2PC Distributed Transactions
- **Prepare Phase**: Coordinator asks all participants if they can commit
- **Commit Phase**: If all participants agree, execute commit
- **Abort Phase**: If any participant rejects, rollback the transaction

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

## 🛠️ Building and Running

### Prerequisites

1. **Protocol Buffers Compiler**
```bash
# Ubuntu/Debian
sudo apt-get install protobuf-compiler

# macOS
brew install protobuf
```

2. **Go Plugins**
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Generate Proto Files

```bash
make proto
```

Or manually:
```bash
protoc --go_out=api/proto --go_opt=paths=source_relative \
       --go-grpc_out=api/proto --go-grpc_opt=paths=source_relative \
       api/proto/*.proto
```

### Local Run (Single Node)

```bash
# Start MongoDB and Redis
docker compose up -d mongo redis

# Run single node
go run ./cmd/server-grpc
```

### Docker Multi-Node Deployment

```bash
# Start 5-node cluster (as required by assignment Q1-Q4)
docker compose -f docker-compose-grpc.yml up -d --build

# View node status
docker compose -f docker-compose-grpc.yml ps

# View logs for a specific node
docker logs studyroom_booking-app-node1-1
docker logs studyroom_booking-app-node2-1
# ... etc for nodes 3, 4, 5

# Stop the cluster
docker compose -f docker-compose-grpc.yml down
```

## 🔧 Configuration

### Environment Variables

- `NODE_ID`: Node ID (e.g., node1, node2, node3)
- `GRPC_PORT`: gRPC service port (default: 50051)
- `RAFT_PORT`: Raft communication port (default: 50052)
- `PEERS`: Node list, format: `node1:host:port,node2:host:port`
- `MONGODB_URI`: MongoDB connection string
- `REDIS_ADDR`: Redis address

### Example Configuration

```bash
export NODE_ID=node1
export GRPC_PORT=50051
export RAFT_PORT=50052
export PEERS="node1:localhost:50052,node2:localhost:50053,node3:localhost:50054"
```

## 📡 gRPC Services

### AuthService
- `Register`: User registration
- `Login`: User login
- `Logout`: User logout
- `Me`: Get current user information

### BookingService
- `CreateBooking`: Create booking (using 2PC)
- `CancelBooking`: Cancel booking
- `JoinWaitlist`: Join waitlist

### SearchService
- `SearchRooms`: Search available rooms

### AdminService
- `CreateRoom`: Create room
- `ListRooms`: List all rooms
- `SetRoomSchedule`: Set room schedule

## 🧪 Testing

### Docker Testing (Recommended)

Run all tests using Docker, including 2PC and Raft tests:

```bash
# Run all tests
docker compose -f docker-compose.test.yml up --build

# Or use test script
docker compose -f docker-compose.test.yml run --rm test
```

The tests will automatically:
1. Generate Protocol Buffer code
2. Run all 2PC tests
3. Run all Raft tests
4. Display test results summary

### Local Testing

If your local environment has Go and protoc configured:

```bash
# Generate proto files first (if not already generated)
make proto

# Run all tests
go test ./test -v

# Run specific tests
go test ./test -v -run TestRaftLeaderElection
go test ./test -v -run Test2PC
```

### Testing gRPC Services with grpcurl

```bash
# Install grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# List services
grpcurl -plaintext localhost:50051 list

# Call Register
grpcurl -plaintext -d '{"email":"test@example.com","password":"test123"}' \
  localhost:50051 studyroom.AuthService/Register
```

### Using Go Client

```go
conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
client := pb.NewAuthServiceClient(conn)
resp, _ := client.Register(ctx, &pb.RegisterRequest{
    Email: "test@example.com",
    Password: "test123",
})
```

## 🔍 Raft Algorithm Details

### Node States
- **Follower**: Follower, receives logs from Leader
- **Candidate**: Candidate, participates in elections
- **Leader**: Leader, handles all write requests

### Election Process
1. Follower times out without receiving heartbeat
2. Transitions to Candidate, increments term
3. Requests votes from other nodes
4. Becomes Leader after receiving majority votes

### Log Replication
1. Leader receives client request
2. Appends operation to local log
3. Sends AppendEntries to all Followers in parallel
4. Commits log after receiving majority acknowledgments
5. Applies log to state machine

## 🔄 2PC Transaction Flow

### Normal Flow
```
Client → Coordinator → Prepare → All Participants
                              ↓
                         All Agree?
                              ↓
                         Commit → All Participants
```

### Failure Flow
```
Client → Coordinator → Prepare → Participant (Reject)
                              ↓
                         Abort → All Participants
```

## 📊 Monitoring and Debugging

### View Raft Status
```bash
# Check which node is Leader
docker logs app-node1 | grep "Leader"
```

### View 2PC Transactions
```bash
# View transaction logs
docker logs app-node1 | grep "2PC"
```

## ⚠️ Important Notes

1. **Raft Node Count**: It is recommended to use an odd number of nodes (3, 5, 7) to ensure majority voting
2. **Network Partition**: In case of network partition, only the majority partition can continue serving
3. **2PC Blocking**: If a participant fails, 2PC may block; timeout mechanisms are needed
4. **Data Consistency**: All write operations must go through the Leader node

## 🔮 Future Improvements

- [ ] Implement Raft snapshots
- [ ] Add 2PC timeout and recovery mechanisms
- [ ] Implement client load balancing
- [ ] Add monitoring and metrics collection
- [ ] Support dynamic node join/removal
