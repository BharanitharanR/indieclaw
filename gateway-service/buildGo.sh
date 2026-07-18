#!/bin/bash

# Exit immediately if a command fails

export PATH=$PATH:$(go env GOPATH)/bin
export TEXT_MODEL="qwen3:8b"
export VISION_MODEL="llava:7b"
# Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-grpc-go@latest
echo "🔨 Generating Protobuf stubs..."
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/gateway/v1/gateway.proto

echo "📦 Tidying Go modules..."
go mod tidy
# Configure the models



rm -Rf ./bin/orchestrator
echo "🏗️ Building Orchestrator..."

go build -o bin/orchestrator ./cmd/orchestrator


echo "✅ Build Successful!"

./bin/orchestrator