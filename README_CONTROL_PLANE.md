# 🎨 Persona Control Plane - Your AI Configuration Dashboard

A **light-beige minimalist web UI** for managing all Indieclaw persona configurations without touching code.

## ⚡ Quick Start (5 minutes)

### 1. Install & Start

```bash
cd persona-control-plane
npm install
npm start
```

### 2. Open Browser

```
http://localhost:3000
```

### 3. Start Managing

- 📝 Create new personas
- ✏️ Edit configurations
- 🔐 Manage phone whitelists
- ✅ Validate TOML
- 💾 Save changes

## 🎨 The Interface

```
┌─────────────────────────────────────────────────────────┐
│  Persona Control Plane  (Light Beige Theme)             │
├──────────────────┬────────────────────────────────────┤
│  Personas        │  Executive Coach Pro               │
│  ═════════════   │  Editing persona configuration    │
│                  │                                     │
│  • default       │  [persona]                         │
│  • executive_    │  name = "Executive Coach Pro"     │
│    coach (✓)     │  version = "1.0"                  │
│  • my_persona    │                                     │
│                  │  [models]                          │
│  + New  Delete   │  text_model = "qwen3:8b"          │
│                  │  vision_model = "gemma4:e2b"      │
│                  │                                     │
│                  │  [personality]                     │
│                  │  prompt_template = """..."""       │
│                  │  tone = "coaching"                │
│                  │                                     │
│                  │  [whatsapp]                        │
│                  │  allowed_phone_numbers = [         │
│                  │      "91-98765-43210",            │
│                  │      "91-87654-32109",            │
│                  │  ]                                 │
│                  │                                     │
│                  │  💾 Save    ✓ Validate             │
│                  │                                     │
└──────────────────┴────────────────────────────────────┘
```

## 📁 What's Included

```
persona-control-plane/
├── server.js              # Express backend with APIs
├── package.json           # Dependencies
├── public/
│   └── index.html         # Beautiful UI (all-in-one file)
├── README.md             # Technical docs
└── node_modules/         # (created after npm install)
```

## 🚀 Complete System Setup

Run everything together:

### Terminal 1: Control Plane (Port 3000)
```bash
cd persona-control-plane
npm install
npm start
```

### Terminal 2: Orchestrator (Port 9000)
```bash
cd gateway-service
PERSONA_NAME=executive_coach go run ./cmd/orchestrator/main.go
```

### Terminal 3: WhatsApp Bot
```bash
cd approach-road/wa-echo-loop
npm install
PERSONA_NAME=executive_coach node app.js
```

### Terminal 4: Browser
```
http://localhost:3000
```

## 🎯 Key Features

| Feature | What It Does |
|---------|-------------|
| **Persona List** | View all available personas in sidebar |
| **TOML Editor** | Edit configuration directly in browser |
| **Validation** | Real-time TOML syntax checking |
| **Create** | Generate new persona from template |
| **Delete** | Remove unwanted personas (not default) |
| **Save** | Apply changes immediately to disk |
| **Responsive** | Works on desktop, tablet, mobile |

## 🎨 Design Features

### Theme
- **Light Beige**: Warm, professional, minimalist
- **Color Palette**: Beige, white, subtle shadows
- **Typography**: Clean sans-serif, monospace for code
- **Spacing**: Generous padding, breathing room

### User Experience
- **Minimalistic**: No clutter, only essentials
- **Intuitive**: Click to select, type to edit
- **Responsive**: Adapts to screen size
- **Fast**: No frameworks, lightweight JavaScript

## 💡 Common Tasks

### Edit Phone Whitelist

1. Select persona from sidebar
2. Find `[whatsapp]` section
3. Edit `allowed_phone_numbers`:
   ```toml
   allowed_phone_numbers = [
       "91-98765-43210",      # Client 1
       "91-99999-99999",      # New client
   ]
   ```
4. Click "Validate" → Success ✓
5. Click "Save" → Saved 💾

### Create New Persona

1. Click "+ New" button
2. Enter name (e.g., `sales_coach`)
3. Click "Create"
4. Edit TOML template
5. Click "Validate"
6. Click "Save"
7. Deploy with `PERSONA_NAME=sales_coach`

### Customize AI Response

1. Select persona
2. Find `[personality]` section
3. Edit `prompt_template`:
   ```toml
   prompt_template = """You are an expert in X...
   
   When responding:
   - Do this
   - And this
   - Never do this
   
   Tools: %s
   """
   ```
4. Validate & Save
5. Restart orchestrator/bot

## 🔧 Technology Stack

- **Backend**: Node.js + Express.js
- **Frontend**: Vanilla JavaScript + HTML + CSS
- **Config**: TOML parsing & validation
- **API**: REST with JSON

**No external frameworks** - Single HTML file, lightweight backend.

## 📚 Documentation

- **This file**: Quick overview
- `CONTROL_PLANE_GUIDE.md`: Complete user guide
- `persona-control-plane/README.md`: Technical details
- `PERSONA_QUICKSTART.md`: Full system setup

## 🔐 Security

✅ **Good for**:
- Local networks
- Small teams
- Development/testing
- Self-hosted setups

⚠️ **Not suitable for**:
- Public internet exposure
- Multi-tenant environments
- High-security requirements

**Recommendation**: Run on internal network only, or add authentication proxy.

## 🛠️ Environment Variables

```bash
# Change server port
PORT=4000 npm start

# Change persona directory (if in custom location)
# Set in server.js possiblePaths array
```

## 📊 Real-World Example

**Scenario**: You're an executive coach managing 3 client groups with different coaching styles.

### Setup

1. **Control Plane** manages configurations
2. **Orchestrator** (Go) serves AI responses
3. **Bot** (Node) handles WhatsApp messages

### Workflow

```
Day 1: Setup
├─ Create persona: "fortune_500_coaching"
├─ Add prompt for C-level executives
└─ Add authorized client phone numbers

Day 2: Launch
├─ Start control plane (manage configs)
├─ Start orchestrator (AI engine)
├─ Start bot (WhatsApp interface)
└─ Clients start messaging

Day 3: Iterate
├─ Edit prompt based on feedback
├─ Add more clients to whitelist
├─ Validate before saving
├─ Restart orchestrator/bot
└─ Repeat as needed
```

## ✅ Quick Checklist

Before you start:

- [ ] Node.js installed: `node --version`
- [ ] npm installed: `npm --version`
- [ ] In correct directory: `pwd`
- [ ] Port 3000 available: `lsof -i :3000`

To get started:

- [ ] `npm install`
- [ ] `npm start`
- [ ] Open http://localhost:3000
- [ ] See persona list in sidebar
- [ ] Click a persona
- [ ] Edit TOML in textarea
- [ ] Click "Validate"
- [ ] Click "Save"
- [ ] Check console for success message

## 🎓 Learning Path

### Beginner (15 min)
1. Start control plane
2. Open http://localhost:3000
3. View existing personas
4. Make small change (change tone: "friendly")
5. Validate & save
6. Restart orchestrator to see effect

### Intermediate (1 hour)
1. Create new persona from scratch
2. Customize prompt for specific use case
3. Add authorized phone numbers
4. Test with orchestrator/bot
5. Iterate based on responses

### Advanced (2+ hours)
1. Manage multiple personas
2. Build deployment workflow
3. Backup & version control personas
4. Integrate with CI/CD
5. Add custom validation/hooks

## 🚨 Troubleshooting

### "Port 3000 already in use"
```bash
# Use different port
PORT=4000 npm start
```

### "Cannot find personas directory"
- Control plane logs the path on startup
- Ensure gateway-service is in expected location
- Verify `config/personas/` exists

### "TOML validation error"
- Check syntax (quotes, indentation)
- Look at error line number
- Use https://www.toml-lint.com/ to validate

### "Changes don't take effect"
- Save in UI ✓
- Restart orchestrator ✓
- Restart bot ✓

## 🔄 Integration with System

Control plane connects everything:

```
Browser (Port 3000)
    ↓
Express.js Server
    ↓
TOML Files (gateway-service/config/personas/)
    ↓
Orchestrator (Go)  +  Bot (Node.js)
    ↓
LLM + WhatsApp
```

## 📞 Support Resources

| Issue | Solution |
|-------|----------|
| Won't start | Check port availability, try PORT=4000 |
| Can't find personas | Verify directory path in logs |
| TOML invalid | Use TOML validator online |
| Changes don't apply | Remember to restart orchestrator/bot |

## 🎉 You're Ready!

```bash
# Just run these 3 commands:
cd persona-control-plane
npm install
npm start

# Then open: http://localhost:3000
```

---

## 📖 Complete Documentation Map

```
Main Entry Points:
├─ README_CONTROL_PLANE.md          ← You are here
├─ CONTROL_PLANE_GUIDE.md           ← Detailed user guide
└─ PERSONA_QUICKSTART.md            ← Full system setup

Control Plane Docs:
├─ persona-control-plane/README.md  ← Technical details
└─ persona-control-plane/server.js  ← API implementation

Backend Docs:
├─ IMPLEMENTATION_SUMMARY.md        ← Go orchestrator setup
└─ gateway-service/config/personas/README.md ← Persona format

Integration Docs:
├─ WHATSAPP_INTEGRATION_SUMMARY.md  ← Bot integration
└─ approach-road/wa-echo-loop/PERSONA_INTEGRATION.md
```

---

**Ready to manage personas without code? Start the control plane now!** 🚀

```bash
cd persona-control-plane && npm install && npm start
```

Then open: **http://localhost:3000** ✨
