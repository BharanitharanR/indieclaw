#!/bin/bash

# Exit immediately if a command fails
	
export PERSONA_NAME="executive_coach"
export PATH=$PATH:$(go env GOPATH)/bin
export TEXT_MODEL="qwen3:8b"
export VISION_MODEL="llava:7b"
export EMBEDDING_MODEL="nomic-embed-text"
export MCP_CONFIG_PATH="/Users/bharani/Desktop/aiAgentCompaction/indieclaw/gateway-service/bin/mcp-servers-config.json"
export LOG_PATH="/Users/bharani/Desktop/aiAgentCompaction/indieclaw/gateway-service/bin/logs/orchestrator.log"
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



rm -Rf /Users/bharani/Desktop/aiAgentCompaction/indieclaw/gateway-service/bin/orchestrator
echo "🏗️ Building Orchestrator..."


go build -o /Users/bharani/Desktop/aiAgentCompaction/indieclaw/gateway-service/bin /Users/bharani/Desktop/aiAgentCompaction/indieclaw/gateway-service/cmd/orchestrator  


echo "✅ Build Successful!"

/Users/bharani/Desktop/aiAgentCompaction/indieclaw/gateway-service/bin/orchestrator