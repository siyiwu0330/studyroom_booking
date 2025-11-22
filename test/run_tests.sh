#!/bin/bash
set -e

echo "========================================="
echo "StudyRoom Distributed System - Test Suite"
echo "========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
PASSED=0
FAILED=0
TOTAL=0

# Function to run a test and count results
run_test() {
    local test_name=$1
    local test_pattern=$2
    
    echo "----------------------------------------"
    echo "Running: $test_name"
    echo "----------------------------------------"
    
    TOTAL=$((TOTAL + 1))
    
    if go test ./test -v -run "$test_pattern" -count=1 2>&1 | tee /tmp/test_output.log; then
        echo -e "${GREEN}✓ $test_name PASSED${NC}"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}✗ $test_name FAILED${NC}"
        FAILED=$((FAILED + 1))
    fi
    echo ""
}

# Generate proto files first
echo "Generating Protocol Buffer files..."
if protoc --version > /dev/null 2>&1; then
    protoc --go_out=api/proto --go_opt=paths=source_relative \
           --go-grpc_out=api/proto --go-grpc_opt=paths=source_relative \
           api/proto/*.proto 2>/dev/null || echo "Proto files already generated"
else
    echo "Warning: protoc not found, skipping proto generation"
fi
echo ""

# Run 2PC tests
echo "========================================="
echo "2PC Transaction Tests"
echo "========================================="
run_test "2PC Basic Commit" "Test2PCBasicCommit"
run_test "2PC Abort on Failure" "Test2PCAbortOnPrepareFailure"
run_test "2PC Concurrent Transactions" "Test2PCConcurrentTransactions"

# Run Raft tests
echo "========================================="
echo "Raft Consensus Algorithm Tests"
echo "========================================="
run_test "Raft Leader Election" "TestRaftLeaderElection"
run_test "Raft Leader Timeout" "TestRaftLeaderTimeout"
run_test "Raft Log Replication" "TestRaftLogReplication"
run_test "Raft New Node Join" "TestRaftNewNodeJoin"
run_test "Raft Split-Brain Prevention" "TestRaftSplitBrainPrevention"

# Final summary
echo "========================================="
echo "Test Summary"
echo "========================================="
echo "Total Tests: $TOTAL"
echo -e "${GREEN}Passed: $PASSED${NC}"
if [ $FAILED -gt 0 ]; then
    echo -e "${RED}Failed: $FAILED${NC}"
else
    echo -e "${GREEN}Failed: $FAILED${NC}"
fi

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed! ✓${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed! ✗${NC}"
    exit 1
fi



