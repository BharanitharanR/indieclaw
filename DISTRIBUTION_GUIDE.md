# Indieclaw Distribution Package - For End Users

## For Distributors (You)

### What to Include in ZIP

```
indieclaw-installer.zip
│
├── install.sh                      ← Main installer (executable)
├── SETUP_INSTRUCTIONS.txt          ← Simple step-by-step guide
├── 00_START_HERE.md                ← Orientation
├── QUICK_START.txt                 ← Daily cheat sheet
├── INSTALLATION.md                 ← Detailed guide
├── OLLAMA_MODELS_GUIDE.md          ← Model reference
└── README.md                       ← Overview
```

**Total Size:** ~300KB (just scripts + docs, no code)
**No git clone needed!**
**No downloading code repositories!**

---

### Creating the Distribution ZIP

```bash
# From indieclaw root directory
cd /Users/bharani/Desktop/aiAgentCompaction/indieclaw

# Create distribution folder
mkdir -p dist/indieclaw-installer

# Copy installer files
cp install.sh dist/indieclaw-installer/
cp 00_START_HERE.md dist/indieclaw-installer/
cp QUICK_START.txt dist/indieclaw-installer/
cp INSTALLATION.md dist/indieclaw-installer/
cp OLLAMA_MODELS_GUIDE.md dist/indieclaw-installer/
cp DISTRIBUTION_GUIDE.md dist/indieclaw-installer/

# Create simple launcher
cat > dist/indieclaw-installer/SETUP_INSTRUCTIONS.txt << 'EOF'
================================================================================
                    INDIECLAW INSTALLATION - 3 STEPS
================================================================================

STEP 1: Open Terminal
────────────────────────────────────────────────────────────────────────────
1. Open "Terminal" app (search for it in Spotlight)
2. Drag this folder into Terminal window
3. Type: cd followed by space, then paste the path

STEP 2: Run Installer (one time, ~60 minutes)
────────────────────────────────────────────────────────────────────────────
Type this command exactly:

    bash install.sh

Press Enter and follow the prompts.
☕ The installer will download ~20GB of AI models (takes 15-20 minutes)

STEP 3: Daily Startup (takes 30 seconds)
────────────────────────────────────────────────────────────────────────────
After installation, every day you want to use Indieclaw:

    ~/.indieclaw/start-services.sh

Then open 3 Terminal windows and run:

Terminal 1 - Orchestrator:
    cd ~/indieclaw/gateway-service
    source ~/.indieclaw/indieclaw.env
    go run ./cmd/orchestrator/main.go

Terminal 2 - WhatsApp:
    cd ~/indieclaw/approach-road/wa-echo-loop
    source ~/.indieclaw/indieclaw.env
    npm install
    node app.js

Terminal 3 - Settings (optional):
    cd ~/indieclaw/persona-control-plane
    node server.js
    # Open http://localhost:3001 in your browser

================================================================================
                         TROUBLESHOOTING
================================================================================

Problem: "Command not found: bash"
Solution: You're in the wrong directory. Drag the folder into Terminal.

Problem: "Permission denied"
Solution: Run this first:
    chmod +x install.sh

Problem: Installation hangs
Solution: It's downloading AI models (~20GB). Don't interrupt. Wait 15-20 min.

Problem: Port already in use
Solution: Close other applications and try again.

For more help, see:
  • 00_START_HERE.md
  • INSTALLATION.md
  • QUICK_START.txt

================================================================================
                            WHAT YOU'RE GETTING
================================================================================

✅ Personal AI coaching assistant on your Mac
✅ Private (no cloud, data stays on your laptop)
✅ Runs via WhatsApp
✅ Customizable personality
✅ Web-based configuration

System Requirements:
  • macOS with Apple Silicon (M1/M2/M3+)
  • 16GB RAM minimum
  • 80GB free disk space
  • Internet connection for setup

================================================================================

Ready? Run: bash install.sh

Questions? See INSTALLATION.md or QUICK_START.txt in this folder.

================================================================================
EOF

# Make installer executable
chmod +x dist/indieclaw-installer/install.sh

# Create ZIP
cd dist
zip -r indieclaw-installer.zip indieclaw-installer/

# Show result
ls -lh indieclaw-installer.zip
```

---

## For End Users (Non-Technical People)

### Getting Started

1. **Download** the ZIP file
   - File: `indieclaw-installer.zip` (~300KB)
   - Location: Save to Downloads folder

2. **Unzip**
   - Double-click the file
   - A folder `indieclaw-installer` appears

3. **Follow SETUP_INSTRUCTIONS.txt**
   - Open the file in any text editor
   - Follow the 3 simple steps

That's it! No git, no cloning, no command-line complexity.

---

## File Descriptions for End Users

| File | What It Does |
|------|--------------|
| **SETUP_INSTRUCTIONS.txt** | Simple step-by-step guide (START HERE) |
| **00_START_HERE.md** | Quick orientation (2 min read) |
| **QUICK_START.txt** | Daily commands cheat sheet |
| **INSTALLATION.md** | Detailed guide if something goes wrong |
| **OLLAMA_MODELS_GUIDE.md** | Reference for AI models (if you want to customize) |
| **install.sh** | The actual installer (just run it!) |

---

## End-User Workflow

```
1. Download indieclaw-installer.zip
            ↓
2. Unzip folder
            ↓
3. Read SETUP_INSTRUCTIONS.txt
            ↓
4. Open Terminal
            ↓
5. cd into folder
            ↓
6. Run: bash install.sh
            ↓
7. Wait 60 minutes (downloads AI models)
            ↓
8. Follow printed instructions
            ↓
9. Done! 🎉
```

---

## Distribution Options

### Option A: Email Link
- Upload ZIP to cloud (Google Drive, Dropbox, etc.)
- Send download link to users
- Users download and unzip

### Option B: USB Drive
- Copy ZIP to USB
- Users unzip on their Mac

### Option C: Direct Download
- Host on your website
- Users download and unzip

### Option D: QR Code
- Create QR linking to download
- Users scan with phone

---

## What's NOT in the ZIP

❌ No Indieclaw source code
❌ No git repositories
❌ No node_modules
❌ No compiled binaries
❌ No large files

**Why?** Keeps the ZIP tiny (~300KB) and fresh. Users get latest code automatically.

---

## After Users Unzip

The installer automatically:
- ✅ Installs Homebrew (if needed)
- ✅ Installs Go, Node.js, RabbitMQ, Qdrant, Ollama
- ✅ Downloads AI models
- ✅ Creates configuration files
- ✅ Creates startup scripts
- ✅ Verifies everything works

**Users never need to touch code or Git!**

---

## Support Resources in ZIP

For users who get stuck:
1. Read **SETUP_INSTRUCTIONS.txt** → solves 90% of issues
2. Read **INSTALLATION.md** → troubleshooting section
3. Check **QUICK_START.txt** → common commands
4. Email the files to support with their logs

---

## Customization for Your Brand

Before creating ZIP, optionally update:

1. **SETUP_INSTRUCTIONS.txt**
   - Add your company name
   - Add support email
   - Add your logo (as ASCII art)

2. **00_START_HERE.md**
   - Add welcome message
   - Add support contact info
   - Add any company-specific notes

3. **install.sh**
   - Update final success message
   - Add company branding in print statements

---

## Version Management

For new versions:
1. Update installer and docs
2. Create new ZIP: `indieclaw-installer-v2.0.zip`
3. Send to users with release notes

Users with old version:
- Don't need to uninstall
- Can run new installer over old install
- Installer updates everything safely

---

## Checklist Before Distribution

- [ ] Tested installation on clean Mac
- [ ] All docs included in ZIP
- [ ] install.sh is executable (chmod +x)
- [ ] No large files (should be <500KB)
- [ ] No git repos in ZIP
- [ ] SETUP_INSTRUCTIONS.txt is clear
- [ ] ZIP creates single folder when unzipped
- [ ] No passwords or secrets in files
- [ ] Tested on Mac with Apple Silicon

---

## Security & Privacy

The ZIP contains:
- ✅ Open-source installation scripts
- ✅ Public documentation
- ✅ No credentials
- ✅ No API keys
- ✅ No private code

Safe to distribute to anyone!

---

## Example Distribution Message

```
Subject: Indieclaw Personal AI Assistant - Installation Package

Hi [User],

Your personal AI coaching assistant is ready!

Attached: indieclaw-installer.zip

To get started:
1. Unzip the folder
2. Open SETUP_INSTRUCTIONS.txt
3. Follow the 3 simple steps

Installation takes about 60 minutes (mostly automated).
No coding experience needed!

If you have questions, see INSTALLATION.md in the folder.

Questions? Reply to this email.

Enjoy! 🚀
```

---

## Summary for Distributors

✅ **Tiny ZIP** (~300KB)
✅ **Self-contained** (no git clone)
✅ **Non-technical friendly** (step-by-step guide)
✅ **Full documentation** (for edge cases)
✅ **Automatic setup** (installer does the work)
✅ **Secure** (no passwords or keys)
✅ **Brandable** (customize docs before distribution)

**Users just unzip and run `bash install.sh`** - that's it!
