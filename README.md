# StudyRoom – Layered Architecture with Distributed gRPC

A simple **Gin + MongoDB + Redis** application implementing layered architecture for room booking and scheduling, served via **Nginx reverse proxy**.

**Note**: This project now includes a **distributed gRPC version** with Raft consensus and 2PC transactions. See [README-GRPC.md](README-GRPC.md) for details.

**GitHub Repository**: [https://github.com/siyiwu0330/studyroom_booking](https://github.com/siyiwu0330/studyroom_booking)

---

##  Prerequisites
- Docker & Docker Compose
- `curl`, `jq`
- [`wrk`](https://github.com/wg/wrk) or use Docker image `williamyeh/wrk`

---

##  Run

### REST API Version (Original)

```bash
# from the monolith project root
docker compose up -d --build
docker compose ps
```

### gRPC Distributed Version

```bash
# Run 5-node cluster (as required by assignment)
docker compose -f docker-compose-grpc.yml up -d --build

# Check cluster status
docker compose -f docker-compose-grpc.yml ps

# View logs (example for node1)
docker logs studyroom_booking-app-node1-1

# See README-GRPC.md for more details
```

---

##  Testing

### Run All Tests with Docker

```bash
# Run comprehensive test suite (2PC + Raft)
docker compose -f docker-compose.test.yml up --build

# This will:
# - Generate Protocol Buffer code
# - Run all 2PC transaction tests
# - Run all Raft consensus algorithm tests
# - Display test results and summary
```

### Test Results

The test suite includes:
- **2PC Tests**: Basic commit, abort on failure, concurrent transactions
- **Raft Tests**: Leader election, timeout/re-election, log replication, new node join, split-brain prevention

See [TEST_REPORT.md](TEST_REPORT.md) for detailed test results and analysis.

**📖 For TA Demonstration**: See [DEMO_GUIDE.md](DEMO_GUIDE.md) for step-by-step instructions on running the system, understanding the implementation, and demonstrating features.

---

##  Project Structure

```
studyroom_booking/
├── cmd/
│   ├── server/          # REST API server
│   └── server-grpc/     # gRPC distributed server
├── internal/
│   ├── raft/            # Raft consensus implementation
│   ├── twopc/           # 2PC transaction implementation
│   ├── grpc/            # gRPC handlers
│   └── ...
├── test/                # Test suite
├── api/proto/           # Protocol Buffer definitions
└── docker-compose*.yml  # Docker configurations
```

