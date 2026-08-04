# 🎯 Indieclaw - Personal AI Assistant

Welcome! This is your all-in-one installation package for Indieclaw.

**No coding experience needed.** Everything is automated.

---

## ⚡ Quick Start (3 Simple Steps)

### Step 1: Open Terminal
- Open the "Terminal" app on your Mac
- Drag this folder into the Terminal window
- Type: `cd ` then paste the path

### Step 2: Run Installer (once, takes ~60 minutes)
```bash
bash install.sh
```
☕ Grab coffee - it's downloading AI models

### Step 3: Start Using (daily startup takes 30 seconds)
After installation, follow the colorful instructions that appear.

**That's it!** No git, no complicated commands, no technical knowledge needed.

---

## 📚 Documentation Files

**Choose your reading level:**

| File | For Whom | Time |
|------|----------|------|
| **SETUP_INSTRUCTIONS.txt** | Non-technical users | 5 min |
| **00_START_HERE.md** | Quick overview | 2 min |
| **QUICK_START.txt** | Daily reference | 3 min |
| **INSTALLATION.md** | Troubleshooting | 15 min |
| **OLLAMA_MODELS_GUIDE.md** | Advanced customization | 10 min |

**Start with: SETUP_INSTRUCTIONS.txt**

---

## ✅ What You Get

✨ **Your own personal AI assistant**
- Private (all on your Mac, no cloud)
- Customizable personality
- Accessible via WhatsApp
- Web-based settings panel

💻 **Everything is local**
- No subscriptions
- No API keys
- No cloud services
- Runs on your laptop

🔒 **Secure & Private**
- Your data never leaves your computer
- No monitoring or tracking
- No telemetry

---

## 🎯 What Happens When You Run install.sh

The installer automatically:

1. ✅ Checks if your Mac is compatible
2. ✅ Installs required tools (Go, Node.js, RabbitMQ, etc.)
3. ✅ Downloads AI models (~20GB)
4. ✅ Configures everything
5. ✅ Creates startup scripts
6. ✅ Verifies everything works

**No manual steps. No configuration. Just run and wait.**

---

## 📋 System Requirements

Your Mac needs:
- ✅ Apple Silicon (M1, M2, M3, M4, etc.)
- ✅ 16GB RAM (8GB minimum, but will be slow)
- ✅ 80GB free disk space
- ✅ Internet connection

**Unsure?** The installer checks everything for you!

---

## 🚀 After Installation

You'll have:
- **~/.indieclaw/** folder with all configuration
- **Startup scripts** for easy daily use
- **Web UI** to customize your AI personality
- **WhatsApp integration** to chat with your AI

Every day you want to use it:
1. Open Terminal
2. Run one command
3. 3 windows open automatically
4. Start chatting!

---

## ❓ Have Questions?

- **Before installing:** Read SETUP_INSTRUCTIONS.txt
- **During installation:** The script shows you what it's doing
- **After installation:** Read QUICK_START.txt
- **Troubleshooting:** See INSTALLATION.md

**No questions to worry about!** This is designed to be foolproof.

---

## 🎁 What's Included in This Package

Small & clean distribution (~300KB):
- ✅ Installer script
- ✅ All documentation
- ✅ No code (downloads fresh automatically)
- ✅ No git needed
- ✅ No cloning required

**Just unzip and run!**

---

## 🔄 Keep It Updated

Want a newer version later?
1. Download the new package
2. Run `bash install.sh` again
3. Everything updates safely
4. No reinstall needed!

---

## 🎉 Ready?

**Just run this and follow along:**

```bash
bash install.sh
```

The installer will:
- Guide you through each step
- Show colorful progress
- Print clear instructions
- Tell you when it's done

**No surprises. No confusion.**

---

## 📞 Support

If something goes wrong:

1. **Check the logs:**
   ```bash
   tail -f ~/.indieclaw/install.log
   ```

2. **Read the troubleshooting guide:**
   - See INSTALLATION.md → Troubleshooting section

3. **Verify services are running:**
   ```bash
   ps aux | grep -E 'rabbitmq|qdrant|ollama'
   ```

4. **Check RabbitMQ is healthy:**
   ```
   http://localhost:15672
   (username: guest, password: guest)
   ```

---

## 💡 Pro Tips

- **First run takes 60 minutes** (downloading models)
- **Requires 16GB RAM** during peak usage
- **Needs 80GB disk space** for models + data
- **WiFi recommended** for faster setup
- **Keep Terminal windows open** while using

---

## 🎯 Next Steps

1. **Read:** SETUP_INSTRUCTIONS.txt
2. **Run:** `bash install.sh`
3. **Follow:** The colorful instructions
4. **Enjoy:** Your personal AI assistant!

---

**Questions?** Everything is explained in the documentation files.

**Ready?** Just run `bash install.sh`

**No coding needed. No confusion. Just install and enjoy!** 🚀
