# Indieclaw Installation Guide

## Quick Start (5 minutes setup)

### Step 1: Run the Installer

```bash
bash install.sh
```

The installer will:
- ✅ Check your system requirements
- ✅ Install all dependencies via Homebrew
- ✅ Configure RabbitMQ
- ✅ Download LLM models (~50GB, takes 30-60 minutes)
- ✅ Create startup scripts and environment files

### Step 2: Start Services

```bash
~/.indieclaw/start-services.sh
```

This starts:
- 🐰 RabbitMQ (port 5672)
- 🗄️ Qdrant (port 6334)
- 🤖 Ollama (port 11434)

### Step 3: Run Indieclaw Components

**Terminal 1 - Orchestrator:**
```bash
cd ~/indieclaw/gateway-service
source ~/.indieclaw/indieclaw.env
go run ./cmd/orchestrator/main.go
```

**Terminal 2 - WhatsApp Adapter:**
```bash
cd ~/indieclaw/approach-road/wa-echo-loop
source ~/.indieclaw/indieclaw.env
npm install
node app.js
```

**Terminal 3 - Control Plane (optional):**
```bash
cd ~/indieclaw/persona-control-plane
node server.js
# Open http://localhost:3001 in browser
```

---

## What Gets Installed

### Automatically Installed by Installer

| Component | Size | Time | Purpose |
|-----------|------|------|---------|
| **Homebrew** | 500MB | 3 min | Package manager |
| **Go 1.21** | 400MB | 2 min | Build orchestrator |
| **Node.js 18 LTS** | 300MB | 1 min | Adapters & UI |
| **RabbitMQ** | 200MB | 1 min | Message queue |
| **Qdrant** | 200MB | 1 min | Vector database |
| **Ollama** | 1GB | 5 min | LLM runtime |
| **Python 3.11** | 100MB | 1 min | Web tools |
| **LLM Models** | ~50GB | 30-60 min | AI models |
| **Total** | **~60GB** | **45-75 min** | Full stack |

### Manual Setup (Indieclaw Code)

After installer completes:
```bash
git clone https://github.com/your-org/indieclaw.git ~/indieclaw
cd ~/indieclaw
# Everything else is pre-configured!
```

---

## System Requirements Check

The installer verifies:

✅ **macOS** (Apple Silicon M1/M2/M3+)
✅ **RAM:** 8GB minimum (16GB recommended)
✅ **Disk:** 80GB free space
✅ **Internet:** For downloads

If any checks fail, the installer will stop and tell you what's needed.

---

## Ports Used

After installation, these ports will be in use:

```
5672    → RabbitMQ (AMQP)
15672   → RabbitMQ Admin UI
6334    → Qdrant (gRPC)
6333    → Qdrant (HTTP)
11434   → Ollama API
9000    → Orchestrator (gRPC)
3001    → Control Plane (HTTP)
```

**Make sure these ports are not blocked by firewall or other applications.**

---

## Environment Configuration

After installation, edit `~/.indieclaw/indieclaw.env`:

```bash
# LLM Models (these must match downloaded models)
TEXT_MODEL=qwen3:8b
VISION_MODEL=gemma4:e2b
EMBEDDING_MODEL=nomic-embed-text

# Persona to use
PERSONA_NAME=executive_coach

# Queue configuration
RABBITMQ_URL=amqp://indieclaw:secretpass@localhost:5672/

# Logging
LOG_PATH=$HOME/.indieclaw/logs/orchestrator.log

# WhatsApp trigger prefix
TRIGGER_PREFIX=Self
```

---

## First-Time Setup Checklist

After installation:

- [ ] Run `~/.indieclaw/start-services.sh`
- [ ] Wait 5 seconds for all services to start
- [ ] Verify services with:
  ```bash
  curl http://localhost:11434/api/tags
  curl http://localhost:6333/health
  rabbitmqctl status
  ```
- [ ] Clone or copy Indieclaw code to `~/indieclaw`
- [ ] Update phone numbers in persona TOML files
- [ ] Start Orchestrator in Terminal 1
- [ ] Start WhatsApp Adapter in Terminal 2
- [ ] Open Control Plane at http://localhost:3001

---

## Troubleshooting

### "Port already in use"
```bash
# Find what's using the port (e.g., 5672)
lsof -i :5672
# Kill the process if needed
kill -9 <PID>
```

### "Models not downloaded"
```bash
# Models download on first Ollama startup
# If interrupted, manually download:
ollama pull qwen3:8b
ollama pull gemma4:e2b
ollama pull nomic-embed-text
```

### "RabbitMQ user already exists"
```bash
# Reset RabbitMQ user
sudo rabbitmqctl delete_user indieclaw
sudo rabbitmqctl add_user indieclaw secretpass
sudo rabbitmqctl set_permissions -p / indieclaw ".*" ".*" ".*"
```

### "Go build fails"
```bash
# Ensure Go is in PATH
go version
# If not found, add to ~/.zshrc:
export PATH="$(brew --prefix)/opt/go/bin:$PATH"
source ~/.zshrc
```

### "Node.js modules not found"
```bash
# Reinstall node modules
cd ~/indieclaw/approach-road/wa-echo-loop
npm install
cd ~/indieclaw/persona-control-plane
npm install
```

### Check installation logs
```bash
cat ~/.indieclaw/install.log
tail -f ~/.indieclaw/logs/orchestrator.log
tail -f ~/.indieclaw/logs/ollama.log
```

---

## Service Management

### Start all services
```bash
~/.indieclaw/start-services.sh
```

### Stop services
```bash
brew services stop rabbitmq
brew services stop qdrant
# For Ollama, kill the process:
pkill -f "ollama serve"
```

### Check service status
```bash
brew services list
ps aux | grep -E 'rabbitmq|qdrant|ollama'
```

### View RabbitMQ UI
```
http://localhost:15672
username: guest
password: guest
```

---

## Advanced Configuration

### Change LLM Models

Edit `~/.indieclaw/indieclaw.env`:
```bash
TEXT_MODEL=llama2:13b        # Larger model
VISION_MODEL=llava           # Different vision model
```

Then restart Ollama:
```bash
pkill -f "ollama serve"
ollama serve &
```

### Increase Resource Limits

In `~/.indieclaw/indieclaw.env`, add:
```bash
# Allocate more GPU VRAM (if using Metal acceleration)
OLLAMA_MEMORY_GB=20

# Increase model context window
OLLAMA_NUM_GPU=1
```

### Custom RabbitMQ Setup

```bash
# Change password
sudo rabbitmqctl change_password indieclaw newpassword

# Check permissions
sudo rabbitmqctl list_user_permissions indieclaw

# Reset everything
sudo rabbitmqctl delete_user indieclaw
sudo rabbitmqctl add_user indieclaw secretpass
sudo rabbitmqctl set_permissions -p / indieclaw ".*" ".*" ".*"
```

---

## Uninstall

To completely remove Indieclaw:

```bash
# Stop services
brew services stop rabbitmq
brew services stop qdrant
pkill -f "ollama serve"

# Remove user data
rm -rf ~/.indieclaw

# Optional: Remove installed packages (careful!)
# brew uninstall rabbitmq qdrant ollama python@3.11 node@18 go
```

---

## Next Steps

1. **Configure Personas** → Open http://localhost:3001
2. **Add Phone Numbers** → Whitelist your WhatsApp numbers
3. **Test Messages** → Send "Self help" to your bot
4. **Monitor Logs** → Check ~/. indieclaw/logs/

---

## Getting Help

- 📋 Check prerequisites: `INDIECLAW_PREREQUISITES.md`
- 📝 View installation log: `~/.indieclaw/install.log`
- 🔍 Search error message in logs
- 🐛 Report issues with full log output

---

## Performance Tips

- **16GB+ RAM:** All models can run concurrently
- **8GB RAM:** Unload unused models between requests
- **SSD Required:** Much faster model loading than HDD
- **WiFi:** Use ethernet for faster model downloads
- **Temperature:** Monitor Mac temps, Ollama may throttle

---

**Happy coding! Your personal AI assistant is ready.** 🚀
