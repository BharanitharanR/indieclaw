# 🎯 Adiyan - Complete Coaching Platform Suite

Welcome to **Adiyan**, a self-contained, installable AI coaching platform.

## 🚀 Start Everything in One Command

```bash
./adiyan-start.sh
```

That's it! This single script launches the entire suite:
- **Nginx** (API Gateway)
- **Qdrant** (Vector Database)  
- **Orchestrator** (AI Coaching Engine)
- **User Context Service** (Knowledge Management)
- **WhatsApp Adapter** (Messaging)
- **Discord Adapter** (Chat)
- **Person Config** (Persona Management)

---

## 📋 Scripts Available

### `adiyan-start.sh`
Launches all services with automatic configuration.

```bash
./adiyan-start.sh
```

**Output:**
```
✨ ADIYAN SUITE STARTED SUCCESSFULLY ✨

Services running:
  ✓ Nginx Gateway              : http://localhost:80
  ✓ Orchestrator              : http://localhost:9000
  ✓ User Context Service      : http://localhost:8001
  ✓ Qdrant Vector DB          : http://localhost:6334
```

**With custom persona:**
```bash
PERSONA_NAME=executive_coach ./adiyan-start.sh
```

### `adiyan-stop.sh`
Gracefully stops all running services.

```bash
./adiyan-stop.sh
```

### `adiyan-logs.sh`
View logs and service status.

```bash
# Show all service status and ports
./adiyan-logs.sh status

# View last 50 lines of orchestrator logs
./adiyan-logs.sh tail orchestrator

# Follow logs in real-time
./adiyan-logs.sh follow orchestrator

# Stop watching (Ctrl+C)
```

---

## 📚 Documentation

- **[ADIYAN_QUICKSTART.md](./ADIYAN_QUICKSTART.md)** — Complete user guide
  - Prerequisites & installation
  - Configuration options
  - Troubleshooting
  - API endpoints

- **[IMPLEMENTATION_SUMMARY.md](./IMPLEMENTATION_SUMMARY.md)** — Technical details
  - Architecture overview
  - Service topology
  - Performance characteristics
  - Deployment options

---

## ⚡ Quick Start (5 minutes)

### 1. Prerequisites
```bash
# Check you have these installed
go version           # Go 1.21+
nginx -v             # Nginx
docker --version     # Docker (for Qdrant)

# Install Ollama models (if not done)
ollama pull qwen2:7b
ollama pull llava:7b
```

### 2. Build (one time)
```bash
cd gateway-service
go build -o ./bin/orchestrator ./cmd/orchestrator/main.go
go build -o ./bin/user-context-service ./cmd/user-context-service/main.go
cd ..
```

### 3. Start
```bash
./adiyan-start.sh
```

### 4. Test
```bash
# Health check
curl http://localhost/health

# Send coaching request
curl -X POST http://localhost:9000/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "How can I improve my leadership?",
    "phoneNumber": "919361315379"
  }'
```

---

## 🎮 Service Endpoints

| Service | URL | Purpose |
|---------|-----|---------|
| **Nginx Gateway** | `http://localhost` | Main API entry point |
| **Orchestrator** | `http://localhost:9000` | AI coaching engine (gRPC) |
| **User Context** | `http://localhost:8001` | Knowledge management (REST) |
| **Qdrant** | `http://localhost:6334` | Vector database |
| **WhatsApp** | `http://localhost:3000` | Messaging adapter |
| **Discord** | `http://localhost:3001` | Discord bot |
| **Person Config** | `http://localhost:8080` | Persona management |

---

## 🔧 Common Commands

```bash
# Start with executive coach persona
PERSONA_NAME=executive_coach ./adiyan-start.sh

# Start with custom models
TEXT_MODEL=mistral:7b ./adiyan-start.sh

# View orchestrator logs
./adiyan-logs.sh follow orchestrator

# Check service status
./adiyan-logs.sh status

# Stop all services
./adiyan-stop.sh

# Clear logs
./adiyan-logs.sh clear
```

---

## 📁 What's Where

```
indieclaw/
├── adiyan-start.sh           ← START HERE (main script)
├── adiyan-stop.sh            ← Stop services
├── adiyan-logs.sh            ← View logs
├── ADIYAN_QUICKSTART.md      ← User guide
├── IMPLEMENTATION_SUMMARY.md ← Technical details
├── README_STARTUP.md         ← This file
├── nginx.conf                ← Auto-generated gateway config
├── logs/                     ← Service logs
│   ├── orchestrator.log
│   ├── ucs.log
│   └── ...
└── gateway-service/
    ├── bin/
    │   ├── orchestrator       ← AI engine
    │   └── user-context-service ← Knowledge service
    └── ...
```

---

## 🆘 Troubleshooting

**Port already in use?**
```bash
# Find what's using the port
lsof -i :9000

# Kill it
kill -9 <PID>
```

**Services won't start?**
```bash
# Check logs
./adiyan-logs.sh follow orchestrator

# Common issues:
# - Ollama not running: ollama serve
# - Models not loaded: ollama pull qwen2:7b
# - Nginx already running: lsof -i :80
```

**WhatsApp not receiving messages?**
```bash
# Check logs
./adiyan-logs.sh follow whatsapp

# Rescan QR code in web.whatsapp.com
```

---

## 🎯 What Each Service Does

**Orchestrator** (AI Coaching Engine)
- Processes coaching requests
- Uses 7-stage pipeline for quality responses
- Maintains conversation context
- Integrates with persona rules

**User Context Service** (Knowledge Management)
- Stores user profiles and coaching history
- Tracks patterns, experiments, learnings
- Enables personalized coaching
- Zero database setup needed

**Nginx** (API Gateway)
- Routes requests to appropriate services
- Provides single entry point
- Rate limiting & security
- Request logging

**Qdrant** (Vector Database)
- Semantic search on conversation history
- Embedding storage
- Fast retrieval of relevant context

**Adapters** (WhatsApp, Discord)
- Connect Adiyan to messaging platforms
- Bidirectional message flow
- User authentication

---

## 📊 Architecture

```
User (WhatsApp/Discord/API)
        ↓
    Nginx Gateway (Port 80)
        ↓
  ┌─────┴──────┐
  ↓            ↓
Orchestrator  User Context Service
  ↓            ↓
Qdrant      User Profiles
  ↓            ↓
Ollama      ~/.adiyan/users/
(Models)    (JSON files)
```

---

## 💡 Key Features

✅ **Single Script Launch** - Start everything with `./adiyan-start.sh`

✅ **Zero External Setup** - User data stored locally, no cloud dependency

✅ **Persona-Based** - Swap personalities (executive coach, therapist, etc.)

✅ **Knowledge Tracking** - Remembers user patterns and learnings

✅ **Multi-Channel** - WhatsApp, Discord, HTTP API

✅ **Production Ready** - Logging, error handling, graceful shutdown

✅ **Cross-Platform** - Windows, macOS, Linux (same binary)

✅ **Offline Capable** - Everything runs locally

---

## 🚀 Deployment

### For End Users
```bash
# Download, build, start
./adiyan-start.sh
```

### For Developers
```bash
# Customize persona
# Edit gateway-service/config/personas/my_persona.toml
PERSONA_NAME=my_persona ./adiyan-start.sh
```

### For Enterprises
```bash
# Cross-compile for deployment
GOOS=linux GOARCH=amd64 go build -o orchestrator ./cmd/orchestrator/main.go

# Use Docker
docker build -t adiyan:latest .
docker run -p 80:80 adiyan:latest
```

---

## 📞 Next Steps

1. **Start the suite:** `./adiyan-start.sh`
2. **Check status:** `./adiyan-logs.sh status`
3. **View logs:** `./adiyan-logs.sh follow orchestrator`
4. **Test coaching:** Send a message via WhatsApp or API
5. **Check knowledge:** Look at `~/.adiyan/users/`

---

## 📖 Learn More

- **[ADIYAN_QUICKSTART.md](./ADIYAN_QUICKSTART.md)** — Comprehensive guide
- **[IMPLEMENTATION_SUMMARY.md](./IMPLEMENTATION_SUMMARY.md)** — Architecture & design
- **Logs:** `tail -f logs/orchestrator.log`

---

## ✨ Ready to Coach?

```bash
./adiyan-start.sh
```

Adiyan will be ready to coach in 30 seconds.

Happy coaching! 🎯

---

**Questions?** Check the guides above or review service logs.
