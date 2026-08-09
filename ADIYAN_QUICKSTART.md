# 🚀 Adiyan Suite Quick Start Guide

**Adiyan** (Tamil: அடியான் - "servant") is a complete, installable AI coaching platform suite with zero technical setup.

## What's Included

The Adiyan suite launches 7 integrated services:

1. **Nginx** - API Gateway (port 80)
2. **Qdrant** - Vector database for semantic search (port 6334)
3. **Orchestrator** - AI coaching engine (port 9000)
4. **User Context Service** - User knowledge management (port 8001)
5. **WhatsApp Adapter** - Messaging integration (port 3000)
6. **Discord Adapter** - Discord bot integration (port 3001)
7. **Person Config** - Persona configuration node (port 8080)

---

## Prerequisites

### Required
- **Go** 1.21+ (for building orchestrator & UCS)
- **Nginx** (gateway)
- **Ollama** (LLM serving) — running with models
- **Docker** (optional, for Qdrant)

### Install Dependencies (macOS)

```bash
# Install Homebrew (if not installed)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install required tools
brew install go nginx docker

# Install Ollama (download from ollama.ai)
# After install: ollama pull qwen2:7b && ollama pull llava:7b
```

### Install Dependencies (Ubuntu/Debian)

```bash
sudo apt-get update
sudo apt-get install -y golang-go nginx docker.io

# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh
ollama pull qwen2:7b && ollama pull llava:7b
```

---

## Build Adiyan Binaries

Before starting, build the Go services:

```bash
cd /path/to/indieclaw/gateway-service

# Build Orchestrator
go build -o ./bin/orchestrator ./cmd/orchestrator/main.go

# Build User Context Service
go build -o ./bin/user-context-service ./cmd/user-context-service/main.go

# Verify binaries exist
ls -lh ./bin/orchestrator ./bin/user-context-service
```

---

## Quick Start (3 Steps)

### 1️⃣ Start Adiyan

```bash
cd /path/to/indieclaw

# Make script executable (first time only)
chmod +x adiyan-start.sh

# Start the suite
./adiyan-start.sh
```

You should see:

```
✨ ADIYAN SUITE STARTED SUCCESSFULLY ✨

Services running:
  ✓ Nginx Gateway              : http://localhost:80
  ✓ Orchestrator              : http://localhost:9000 (gRPC)
  ✓ User Context Service      : http://localhost:8001
  ✓ Qdrant Vector DB          : http://localhost:6334

Adiyan suite is running. Press Ctrl+C to stop.
```

### 2️⃣ Verify Services

```bash
# Check all services
./adiyan-logs.sh status

# Test health endpoint
curl http://localhost/health
```

### 3️⃣ Send a Coaching Request

**Via WhatsApp:**
- Open WhatsApp Web (`web.whatsapp.com`)
- Message the bot number
- Start with trigger word (default: "Self")

**Via API:**
```bash
curl -X POST http://localhost:9000/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "How can I improve my leadership?",
    "phoneNumber": "919361315379"
  }'
```

---

## Management Commands

### Start Services

```bash
# Start everything
./adiyan-start.sh

# With custom persona
PERSONA_NAME=executive_coach ./adiyan-start.sh

# With custom models
TEXT_MODEL=qwen2:7b VISION_MODEL=llava:7b ./adiyan-start.sh
```

### View Logs

```bash
# Show all service status
./adiyan-logs.sh status

# View last 50 lines of orchestrator logs
./adiyan-logs.sh tail orchestrator

# Follow orchestrator logs in real-time
./adiyan-logs.sh follow orchestrator

# Follow user context service logs
./adiyan-logs.sh follow ucs

# Clear all logs
./adiyan-logs.sh clear
```

### Stop Services

```bash
./adiyan-stop.sh
```

Or press **Ctrl+C** in the startup terminal.

---

## Configuration

### Environment Variables

Set these before running `./adiyan-start.sh`:

```bash
# Persona selection
export PERSONA_NAME=executive_coach

# LLM Models
export TEXT_MODEL=qwen2:7b
export VISION_MODEL=llava:7b
export EMBEDDING_MODEL=nomic-embed-text

# Service ports (optional)
export ORCHESTRATOR_PORT=9000
export UCS_PORT=8001
export NGINX_PORT=80

# User data directory
export USER_DATA_DIR=$HOME/.adiyan/users
```

### Available Personas

- `generic_assistant` (default)
- `executive_coach`
- `custom` (see config/personas/)

### Custom Persona

Create `gateway-service/config/personas/my_persona.toml`:

```toml
[persona]
name = "My Custom Persona"
version = "1.0"

[models]
text_model = "qwen2:7b"
vision_model = "llava:7b"

[intent_definitions]
# ... coaching directives
```

Then start with:
```bash
PERSONA_NAME=my_persona ./adiyan-start.sh
```

---

## Data Storage

### User Profiles

User context and knowledge graphs are stored in:

```
~/.adiyan/users/
├── user_1722950123456789.json    # User profile with coaching history
└── user_1722950234567890.json
```

Each file contains:
- User metadata (role, permissions, goals)
- Coaching history
- Patterns, experiments, learnings
- Last session date

### Logs

Service logs are in:

```
./logs/
├── orchestrator.log      # AI coaching engine
├── ucs.log               # User context service
├── nginx.log             # API gateway
├── qdrant.log            # Vector database
├── whatsapp.log          # WhatsApp adapter
├── discord.log           # Discord adapter
└── person-config.log     # Persona management
```

### Vector Database

Qdrant stores conversation vectors in:

```
./qdrant_storage/          # Persistent storage (Docker volume)
```

---

## Troubleshooting

### ❌ "Port X already in use"

```bash
# Find process using port
lsof -i :9000

# Kill process
kill -9 <PID>
```

### ❌ Orchestrator won't start

Check logs:
```bash
./adiyan-logs.sh follow orchestrator
```

Common issues:
- Ollama not running: `ollama serve`
- Models not loaded: `ollama pull qwen2:7b`
- Qdrant not ready: `docker ps | grep qdrant`

### ❌ User Context Service fails

Check logs:
```bash
./adiyan-logs.sh follow ucs
```

Ensure directory exists:
```bash
mkdir -p ~/.adiyan/users
chmod 755 ~/.adiyan/users
```

### ❌ WhatsApp adapter not receiving messages

1. Open `web.whatsapp.com` in a browser
2. Scan QR code in first terminal
3. Check logs: `./adiyan-logs.sh follow whatsapp`

---

## API Endpoints

### Orchestrator (gRPC)

```
grpc://localhost:9000
```

### User Context Service (REST)

```
POST http://localhost:8001/context/lookup?phoneNumber=919361315379
POST http://localhost:8001/context/update
GET  http://localhost:8001/health
```

### Gateway (Nginx)

```
GET  http://localhost/health
GET  http://localhost/orchestrator/*
GET  http://localhost/context/*
```

---

## Performance Tips

1. **For Large Audiences:**
   - Use a load balancer in front of Nginx
   - Scale orchestrator with multiple instances
   - Use Redis for session caching

2. **For Real-time Response:**
   - Keep text models small: `qwen2:7b` or `mistral:7b`
   - Pre-load models: `ollama pull` before startup
   - Monitor: `./adiyan-logs.sh follow orchestrator`

3. **For Knowledge Management:**
   - Archive old user profiles monthly
   - Use Qdrant backups
   - Monitor user data size: `du -sh ~/.adiyan/users`

---

## Development

### Add a New Persona

1. Create `gateway-service/config/personas/my_persona.toml`
2. Define coaching directives, response rules
3. Start with: `PERSONA_NAME=my_persona ./adiyan-start.sh`

### Add a New Adapter

1. Create `approach-road/my-adapter/`
2. Implement adapter interface
3. Update `adiyan-start.sh` to launch it

### Debug Individual Service

```bash
cd gateway-service
./bin/orchestrator    # Run directly (see all logs)
# Ctrl+C to stop
```

---

## Support & Feedback

- 📧 Issues: GitHub issues
- 💬 Questions: Discussion forums
- 🐛 Bugs: File issue with logs

---

## License

Adiyan is provided as-is for personal use and coaching.

---

**Start coaching:**

```bash
./adiyan-start.sh
# Then visit http://localhost/health
```

Happy coaching! 🎯
