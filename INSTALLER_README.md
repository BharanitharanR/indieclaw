# Indieclaw Installer - Complete Documentation

## Overview

This package contains everything needed to install Indieclaw on a new macOS machine (Apple Silicon).

**Total Installation Time:** 45-75 minutes
**Final Setup Time:** 2-3 minutes per day to start services
**Disk Space Required:** 80GB (50GB for LLM models)
**RAM Required:** 16GB minimum (8GB with reduced performance)

---

## Files Included

### 1. **install.sh** ⭐ MAIN INSTALLER
The primary installation script that automates everything.

**Run once:**
```bash
bash install.sh
```

**What it does:**
- ✅ Checks system requirements (macOS, Apple Silicon, RAM, disk space)
- ✅ Installs Homebrew (if not present)
- ✅ Installs Xcode Command Line Tools (if not present)
- ✅ Installs Go 1.21+
- ✅ Installs Node.js 18 LTS
- ✅ Installs RabbitMQ & configures user
- ✅ Installs Qdrant vector database
- ✅ Installs Ollama runtime
- ✅ Downloads LLM models (qwen3:8b, gemma4:e2b, nomic-embed-text)
- ✅ Installs Python 3.11
- ✅ Creates configuration files
- ✅ Creates startup scripts
- ✅ Verifies all installations

**Output:**
- Installation logs → `~/.indieclaw/install.log`
- Configuration → `~/.indieclaw/indieclaw.env`
- Startup script → `~/.indieclaw/start-services.sh`

---

### 2. **INSTALLATION.md** 📖 DETAILED GUIDE
Comprehensive step-by-step installation and troubleshooting guide.

**Contents:**
- Quick start (3 simple steps)
- What gets installed (with sizes and times)
- System requirements check
- Ports used
- Environment configuration
- First-time setup checklist
- Troubleshooting section
- Service management
- Advanced configuration
- Uninstall instructions

**Read this if:**
- You want to understand what's being installed
- Installation fails and you need help
- You want to customize the setup
- You need to troubleshoot issues

---

### 3. **INDIECLAW_PREREQUISITES.md** 🔍 TECHNICAL SPECS
Deep technical documentation of all prerequisites.

**Contents:**
- System requirements breakdown
- Core installable components (languages, databases, LLMs)
- Component topology diagram
- Installation order (dependency chain)
- Disk space breakdown (60GB total)
- Memory requirements during runtime (~20GB peak)
- Ports used (5672, 6334, 9000, 11434, 3001, etc.)
- Required environment variables
- Installation checklist

**Read this if:**
- You want to understand the architecture
- You need to verify specific prerequisites
- You're debugging system issues
- You want to customize components

---

### 4. **QUICK_START.txt** ⚡ CHEAT SHEET
Quick reference guide for daily use and troubleshooting.

**Contents:**
- One-line installation
- Daily startup procedure
- Running all 3 components (3 terminals)
- Quick health checks
- Common commands
- Port reference
- Quick troubleshooting
- Testing your bot
- Persona editing

**Use this:**
- Every day to start services
- As a quick reference while running
- For common troubleshooting
- As a cheat sheet for commands

---

## Usage Flow

### Day 1 (Setup)
```
1. Read QUICK_START.txt (2 minutes)
2. Run install.sh (45-75 minutes)
3. Run ~/.indieclaw/start-services.sh
4. Clone indieclaw code
5. Start 3 components in terminals
6. Test with WhatsApp
```

### Every Other Day (Startup)
```
1. Run ~/.indieclaw/start-services.sh
2. Start 3 components (or just 2 if control plane not needed)
3. Begin using
```

### Customization
```
1. Open http://localhost:3001 (Control Plane)
2. Edit persona settings
3. Save changes
4. Restart orchestrator if needed
```

### Troubleshooting
```
1. Check QUICK_START.txt for quick fixes
2. Check INSTALLATION.md for detailed steps
3. Check ~/.indieclaw/install.log for error details
4. Check ~/. indieclaw/logs/orchestrator.log for runtime issues
```

---

## Architecture After Installation

```
User's MacBook (Apple Silicon)
│
├─ ~/.indieclaw/
│  ├─ install.log               (Installation log)
│  ├─ indieclaw.env             (Configuration)
│  ├─ start-services.sh         (Startup script)
│  └─ logs/
│     ├─ orchestrator.log       (Backend logs)
│     ├─ ollama.log             (LLM logs)
│     └─ ...
│
├─ Services (installed via Homebrew):
│  ├─ RabbitMQ (5672)           - Message queue
│  ├─ Qdrant (6334)             - Vector database
│  ├─ Ollama (11434)            - LLM runtime
│  └─ Models (~50GB)
│     ├─ qwen3:8b               - Text generation
│     ├─ gemma4:e2b             - Vision analysis
│     └─ nomic-embed-text        - Embeddings
│
└─ Applications (in ~/indieclaw/):
   ├─ gateway-service/          - Orchestrator (Go)
   ├─ approach-road/wa-echo-loop/ - WhatsApp Adapter (Node.js)
   └─ persona-control-plane/    - Configuration UI (Node.js)
```

---

## System Checklist

**Before running installer, verify:**
- [ ] macOS machine with Apple Silicon (M1/M2/M3+)
- [ ] At least 8GB RAM (16GB recommended)
- [ ] At least 80GB free disk space
- [ ] Internet connection (fast connection recommended)
- [ ] Ports 5672, 6333, 6334, 9000, 11434, 3001 available

**After running installer, verify:**
- [ ] All components installed (installer shows summary)
- [ ] ~/.indieclaw/ directory exists
- [ ] ~/.indieclaw/indieclaw.env exists
- [ ] ~/.indieclaw/start-services.sh exists
- [ ] Services start without errors
- [ ] Can access http://localhost:3001 (Control Plane)

---

## What's NOT Included

The installer does **NOT** include:

❌ Indieclaw source code (you download separately)
❌ Docker (not needed, runs natively)
❌ Cloud services (all local)
❌ Python packages (only Python runtime)
❌ npm packages (downloaded by each component)
❌ Brave or other browsers (use your existing browser)

---

## Common Issues & Quick Fixes

### "Permission denied: install.sh"
```bash
chmod +x install.sh
bash install.sh
```

### "Installer hangs on model download"
```bash
# Models take 30-60 minutes to download
# Don't interrupt! Let it finish.
# Or check progress: tail -f ~/.indieclaw/logs/ollama.log
```

### "Port already in use"
```bash
lsof -i :5672  # Check what's using port 5672
kill -9 <PID>   # Kill the process
```

### "Go version too old"
```bash
brew upgrade go
go version  # Verify
```

### "Node modules not found"
```bash
cd path/to/component
rm -rf node_modules package-lock.json
npm install
```

---

## Environment Variables

After installation, edit `~/.indieclaw/indieclaw.env`:

```bash
# Models (don't change unless you download new ones)
TEXT_MODEL=qwen3:8b
VISION_MODEL=gemma4:e2b
EMBEDDING_MODEL=nomic-embed-text

# Persona to load
PERSONA_NAME=executive_coach

# Service URLs
OLLAMA_HOST=http://localhost:11434
RABBITMQ_URL=amqp://indieclaw:secretpass@localhost:5672/

# Logging
LOG_PATH=$HOME/.indieclaw/logs/orchestrator.log

# WhatsApp
TRIGGER_PREFIX=Self
```

---

## Startup Sequence

Run in this order:

1. **Start services:**
   ```bash
   ~/.indieclaw/start-services.sh
   ```
   Wait 5 seconds for all services to start.

2. **Terminal 1 - Orchestrator:**
   ```bash
   cd ~/indieclaw/gateway-service
   source ~/.indieclaw/indieclaw.env
   go run ./cmd/orchestrator/main.go
   ```

3. **Terminal 2 - WhatsApp Adapter:**
   ```bash
   cd ~/indieclaw/approach-road/wa-echo-loop
   source ~/.indieclaw/indieclaw.env
   npm install  # First time only
   node app.js
   ```

4. **(Optional) Terminal 3 - Control Plane:**
   ```bash
   cd ~/indieclaw/persona-control-plane
   node server.js
   # Open http://localhost:3001
   ```

---

## Next Steps After Installation

1. **Clone Indieclaw code:**
   ```bash
   git clone https://github.com/your-repo/indieclaw.git ~/indieclaw
   ```

2. **Update phone numbers:**
   Edit `~/indieclaw/gateway-service/config/personas/executive_coach.toml`
   Add your WhatsApp numbers to `allowed_phone_numbers`

3. **Start services:**
   ```bash
   ~/.indieclaw/start-services.sh
   ```

4. **Start components:**
   Run 3 terminals with commands above

5. **Test:**
   Send WhatsApp message: "Self Hello"
   Bot should reply within 30 seconds

6. **Customize:**
   Open http://localhost:3001 to edit persona

---

## Support & Troubleshooting

**For installation issues:**
1. Read INSTALLATION.md (Troubleshooting section)
2. Check ~/.indieclaw/install.log
3. Run individual component tests from QUICK_START.txt

**For runtime issues:**
1. Check ~/. indieclaw/logs/orchestrator.log
2. Check service status: `brew services list`
3. Test individual services with curl commands in QUICK_START.txt

**For code issues:**
1. Check orchestrator logs
2. Check adapter logs (Terminal 2 console)
3. Check control plane logs (Terminal 3 console)

---

## Version Info

This installer package is for:
- **Indieclaw Version:** Latest
- **Target OS:** macOS with Apple Silicon
- **Go:** 1.21+
- **Node.js:** 18 LTS
- **RabbitMQ:** 3.12+
- **Qdrant:** 1.7+
- **Ollama:** Latest

---

## License & Attribution

Indieclaw Personal AI Assistant
© 2024

All installation scripts are provided as-is for personal use.

---

**Ready to install? Start with `bash install.sh`** 🚀
