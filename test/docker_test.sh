#!/bin/sh
set -e

echo "========================================="
echo "Installing Dependencies"
echo "========================================="
go mod download
go get google.golang.org/grpc@latest || true
go get google.golang.org/protobuf/cmd/protoc-gen-go@latest || true
go get google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest || true

echo ""
echo "========================================="
echo "Generating Proto Files"
echo "========================================="
cd /app
export PATH=$PATH:/go/bin
if [ -f api/proto/studyroom.proto ]; then
    protoc --go_out=api/proto --go_opt=paths=source_relative \
           --go-grpc_out=api/proto --go-grpc_opt=paths=source_relative \
           api/proto/*.proto && echo "Proto files generated successfully"
else
    echo "Proto files not found"
fi

echo ""
echo "========================================="
echo "Running 2PC Tests"
echo "========================================="
go test ./test -v -run Test2PC -count=1 || true

echo ""
echo "========================================="
echo "Running Raft Tests"
echo "========================================="
go test ./test -v -run TestRaft -count=1 || true

echo ""
echo "========================================="
echo "Running All Tests (excluding integration)"
echo "========================================="
go test ./test -v -count=1 -tags=!integration

echo ""
echo "========================================="
echo "Test Summary"
echo "========================================="
go test ./test -v -count=1 2>&1 | grep -E "(PASS|FAIL|ok|FAIL)" | tail -10

