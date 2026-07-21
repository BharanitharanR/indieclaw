# 📁 Gateway Service Codebase

This is a Go (Golang) microservices gateway with the following key components:

## 🧩 Project Structure
```
gateway-service/
├── cmd/              # Entry points for services
│   ├── init_db       # Database initialization
│   └── orchestrator  # Main service entry point
├── internal/         # Core business logic
│   ├── chat          # Chat-related functionality
│   ├── config        # Configuration management
│   ├── contracts     # Workflow definitions
│   ├── mcp           # Microservice communication
│   ├── memory        # Vector database/memory management
│   ├── orchestrator  # Workflow orchestration
│   ├── pkg           # Reusable utilities
│   └── service       # Service integrations (e.g., Ollama)
├── proto/            # gRPC definitions
│   └── gateway/v1    # API specifications
├── Makefile          # Build automation
├── buildGo.sh        # Bash build script
├── nginx/            # NGINX configuration (reverse proxy)
├── storage/          # Persistent storage for data
├── snapshots/        # Database snapshots
└── bin/              # Compiled binaries
```

## 🧠 Key Components

### 1. **Orchestrator Service**
- **Entry Point**: `cmd/orchestrator/main.go`
- **Core Logic**: 
  - Manages workflow execution (`internal/orchestrator/workflow.go`)
  - Integrates with Oll, LLMs, and vector databases
  - Handles request routing and response aggregation

### 2. **Configuration System**
- **File**: `internal/config/config.go`
- Manages environment variables and service-specific settings

### 3. **gRPC API**
- **Definitions**: `proto/gateway/v1/*.proto`
- **Generated Code**: `proto/gateway/v1/*.pb.go`
- Exposes endpoints for:
  - Chat interactions
  - Model inference
  - Vector database queries

### 4. **Memory/Vector Database**
- **Implementation**: `internal/memory/vector.go`
- Handles vector embeddings for semantic searches

### 5. **Microservice Communication**
- **Client**: `internal/mcp/client.go`
- **Registry**: `internal/mcp/registry.go`
- Enables service discovery and inter-service communication

## ⚙️ Setup Instructions

### Prerequisites
- [Go 1.20+](https://go.dev/dl/)
- Docker (for database containers)
- Make (for build automation)

### Install Dependencies
```bash
go mod download
```

### Build the Service
```bash
make build
# Or use the bash script
./buildGo.sh
```

### Run the Service
```bash
./gateway-service
```

## 📡 NGINX Integration
- Configuration files in `nginx/` directory
- Set up reverse proxy for API endpoints
- Load balancing for microservices

## 📁 Storage & Snapshots
- `storage/` - Persistent data storage
- `snapshots/` - Database backups for recovery

## 🧾 Contribution Guidelines
1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Follow Go style guidelines (`gofmt -s`)
5. Update documentation

## 📝 Notes
- The `init_db` command sets up required databases
- Ollama integration is handled via `internal/service/ollama_service.go`
- The `vision` package handles image/audio processing