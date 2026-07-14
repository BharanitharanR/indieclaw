.PHONY: proto build run clean

# Generate Go code from .proto files
proto:
	protoc --proto_path=. \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/gateway/v1/gateway.proto

# Build the binary
build:
	go build -o bin/gateway cmd/gateway/main.go

# Run the gateway service
run: build
	./bin/gateway

# Clean generated binaries
clean:
	rm -rf bin/