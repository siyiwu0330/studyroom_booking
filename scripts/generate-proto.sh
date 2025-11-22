#!/bin/bash
set -e

PROTO_DIR="api/proto"
OUT_DIR="api/proto"

# Generate Go code from proto files
protoc --go_out=$OUT_DIR --go_opt=paths=source_relative \
  --go-grpc_out=$OUT_DIR --go-grpc_opt=paths=source_relative \
  $PROTO_DIR/*.proto

echo "Proto files generated successfully!"



