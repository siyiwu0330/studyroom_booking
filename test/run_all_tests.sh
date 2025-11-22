#!/bin/bash
set -e

echo "========================================="
echo "Running 2PC Tests"
echo "========================================="
go test ./test -v -run Test2PC 2>&1 | tee test_results_2pc.txt

echo ""
echo "========================================="
echo "Running Raft Tests"
echo "========================================="
go test ./test -v -run TestRaft 2>&1 | tee test_results_raft.txt

echo ""
echo "========================================="
echo "Test Summary"
echo "========================================="
echo "2PC Tests:"
grep -E "^(PASS|FAIL|✓|✗)" test_results_2pc.txt || echo "No results found"
echo ""
echo "Raft Tests:"
grep -E "^(PASS|FAIL|✓|✗)" test_results_raft.txt || echo "No results found"



