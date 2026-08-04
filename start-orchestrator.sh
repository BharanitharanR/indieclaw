#!/bin/bash
cd /Users/bharani/Desktop/aiAgentCompaction/indieclaw/gateway-service
export PERSONA_NAME=executive_coach
export LOG_PATH=/tmp/orchestrator.log
echo "🚀 Starting Orchestrator (logs: /tmp/orchestrator.log)"
go run ./cmd/orchestrator/main.go
