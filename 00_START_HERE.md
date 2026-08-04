# 🎯 INDIECLAW INSTALLER - START HERE

Welcome! This folder contains everything you need to install Indieclaw on your Mac.

---

## ⚡ Quick Start (3 Steps)

### Step 1: Run Installer (once, takes ~60 minutes)
```bash
bash install.sh
```
☕ Grab coffee - this downloads 50GB of LLM models

### Step 2: Start Services (daily)
```bash
~/.indieclaw/start-services.sh
```

### Step 3: Run Components (3 terminal windows)
See QUICK_START.txt for exact commands

---

## 📚 Documentation Files

Read these in order based on your needs:

### 🚀 **New User? Start Here:**
1. **00_START_HERE.md** ← You are here
2. **QUICK_START.txt** ← Daily cheat sheet
3. **INSTALLATION.md** ← Detailed guide
4. **install.sh** ← The actual installer

### 🔧 **Need Technical Details?**
- **INDIECLAW_PREREQUISITES.md** ← What gets installed
- **INSTALLER_README.md** ← Full documentation

### 📋 **Files Summary:**

| File | Purpose | Read When |
|------|---------|-----------|
| **install.sh** | Main installer (executable) | First time setup |
| **INSTALLATION.md** | Step-by-step guide | Installing or troubleshooting |
| **QUICK_START.txt** | Daily commands cheat sheet | Daily use & quick fixes |
| **INDIECLAW_PREREQUISITES.md** | Technical specifications | Understanding architecture |
| **INSTALLER_README.md** | Complete documentation | Comprehensive reference |
| **00_START_HERE.md** | This file | Quick orientation |

---

## ✅ Requirements Check (Before Installing)

**Your Mac must have:**
- ✅ Apple Silicon (M1, M2, M3, M4, etc.)
- ✅ 16GB RAM minimum (8GB will work but slow)
- ✅ 80GB free disk space (for models + data)
- ✅ macOS Big Sur or newer
- ✅ Internet connection

**Unsure?** Run installer anyway - it checks everything for you.

---

## 🎯 Installation Path

```
START HERE
    ↓
bash install.sh (60 minutes)
    ↓
~/.indieclaw/start-services.sh
    ↓
Start 3 components
    ↓
Test with WhatsApp
    ↓
Done! 🎉
```

---

## 📖 Read These Next

### Just want to get started?
→ Read **QUICK_START.txt** (2 minutes)

### Want detailed setup instructions?
→ Read **INSTALLATION.md** (15 minutes)

### Need to understand what's being installed?
→ Read **INDIECLAW_PREREQUISITES.md** (10 minutes)

### Full documentation?
→ Read **INSTALLER_README.md** (20 minutes)

---

## 🚀 Quick Command Reference

```bash
# Install everything (one time, ~60 min)
bash install.sh

# Start services (every day)
~/.indieclaw/start-services.sh

# Start orchestrator (Terminal 1)
cd ~/indieclaw/gateway-service
source ~/.indieclaw/indieclaw.env
go run ./cmd/orchestrator/main.go

# Start WhatsApp (Terminal 2)
cd ~/indieclaw/approach-road/wa-echo-loop
source ~/.indieclaw/indieclaw.env
node app.js

# Open Control Plane (Terminal 3, optional)
cd ~/indieclaw/persona-control-plane
node server.js
# Visit http://localhost:3001

# Check services running
ps aux | grep -E 'rabbitmq|qdrant|ollama'

# View logs
tail -f ~/.indieclaw/logs/orchestrator.log
```

---

## 🐛 Something Wrong?

1. **Installation fails?**
   - Check: `~/.indieclaw/install.log`
   - Read: INSTALLATION.md → Troubleshooting section

2. **Component won't start?**
   - Check QUICK_START.txt → Common Commands
   - Run health checks listed there

3. **Still stuck?**
   - Read INSTALLER_README.md → Support & Troubleshooting
   - Check ~/. indieclaw/logs/ directory

---

## 📋 What You'll Get

After installation:

**Services:**
- 🐰 RabbitMQ (message queue)
- 🗄️ Qdrant (vector database)
- 🤖 Ollama (LLM runtime)
  - qwen3:8b (text)
  - gemma4:e2b (vision)
  - nomic-embed-text (embeddings)

**Components:**
- ⚙️ Orchestrator (Go backend)
- 💬 WhatsApp Adapter (Node.js)
- 🎛️ Control Plane (Config UI)

**Size:** ~60GB total
**RAM:** ~20GB peak usage
**Ports:** 5672, 6334, 9000, 11434, 3001, 15672

---

## 🆘 Key Ports

If any of these ports show "in use", close the app:
- **5672** → RabbitMQ
- **6334** → Qdrant
- **11434** → Ollama
- **9000** → Orchestrator
- **3001** → Control Plane
- **15672** → RabbitMQ Admin

Check with: `lsof -i :PORT`

---

## 🎯 Next: Run the Installer

Ready? Just run:

```bash
bash install.sh
```

The installer will guide you through everything!

**Questions while installing?** Check INSTALLATION.md in this same folder.

---

## 📞 File Reference

All files in this folder:

- **install.sh** - Automated installer (executable)
- **INSTALLATION.md** - Detailed setup guide + troubleshooting
- **QUICK_START.txt** - Daily commands cheat sheet
- **INDIECLAW_PREREQUISITES.md** - Technical specifications
- **INSTALLER_README.md** - Complete documentation
- **00_START_HERE.md** - This file

---

## ✨ Features After Installation

✅ Personal AI assistant on your laptop
✅ No cloud services (all local)
✅ Privacy-first (data stays on your Mac)
✅ Customizable personas
✅ WhatsApp integration
✅ Web UI for configuration
✅ Semantic search with vector DB
✅ Async processing with queues

---

**Let's go! Run:** `bash install.sh` 🚀

Questions? Check the documentation files above.

Happy coding! 🎉
