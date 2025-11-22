.PHONY: proto build run test clean

# Generate proto files
proto:
	@echo "Generating proto files..."
	@mkdir -p api/proto
	@protoc --go_out=api/proto --go_opt=paths=source_relative \
		--go-grpc_out=api/proto --go-grpc_opt=paths=source_relative \
		api/proto/*.proto
	@echo "Proto files generated!"

# Build the gRPC server
build:
	@echo "Building gRPC server..."
	@go build -o bin/server-grpc ./cmd/server-grpc
	@echo "Build complete!"

# Run the gRPC server (single node)
run:
	@go run ./cmd/server-grpc

# Run tests
test:
	@go test ./...

# Clean build artifacts
clean:
	@rm -rf bin/
	@echo "Clean complete!"



