# Persona Control Plane - Complete Setup Guide

A minimalistic, light-themed UI for managing all your persona configurations without touching code.

## 🎯 Overview

The control plane is a web-based management interface that lets you:
- 📝 Create, edit, delete personas
- 🔐 Manage WhatsApp phone number whitelists
- ✓ Validate TOML configuration in real-time
- 🎨 Minimize code changes - everything through the UI

## 🚀 Complete System Setup

### Terminal 1: Start Control Plane

```bash
cd persona-control-plane
npm install
npm start
```

Expected output:
```
📝 Persona Control Plane is running!
🌐 Open your browser: http://localhost:3000
📂 Managing personas in: /Users/bharani/Desktop/aiAgentCompaction/indieclaw/gateway-service/config/personas
```

### Terminal 2: Start Orchestrator

```bash
cd gateway-service
PERSONA_NAME=executive_coach go run ./cmd/orchestrator/main.go
```

Expected output:
```
✅ Persona loaded: Executive Coach Pro
🚀 Orchestrator running on :9000 [Persona: Executive Coach Pro, ...]
```

### Terminal 3: Start WhatsApp Bot

```bash
cd approach-road/wa-echo-loop
npm install
PERSONA_NAME=executive_coach node app.js
```

Expected output:
```
📝 Persona: Executive Coach Pro (v1.0) | Tone: coaching | Allowed Numbers: 2
⚡ Scan this QR Code with WhatsApp:
✅ WhatsApp Bot ready!
```

### Terminal 4: Open Browser

```
http://localhost:3000
```

## 🎨 The Control Plane UI

### Layout

```
┌─────────────────────────────────────────────────────────────┐
│                    Persona Control Plane                     │
├──────────────────┬────────────────────────────────────────┤
│                  │                                          │
│  Personas        │  Executive Coach Pro                     │
│  ═══════════════ │  Editing persona configuration          │
│                  │                                          │
│  • default       │  [persona]                               │
│  • executive_    │  name = "Executive Coach Pro"           │
│    coach (active)│  version = "1.0"                        │
│  • my_persona    │                                          │
│                  │  [models]                                │
│  + New           │  text_model = "qwen3:8b"                │
│  Delete          │  vision_model = "gemma4:e2b"            │
│                  │                                          │
│                  │  [personality]                           │
│                  │  prompt_template = """..."""             │
│                  │                                          │
│                  │  💾 Save    ✓ Validate                   │
│                  │                                          │
└──────────────────┴────────────────────────────────────────┘
```

### Features

#### 📋 Sidebar
- List of all personas
- Active persona highlighted
- "+ New" button to create personas
- "Delete" button to remove personas

#### ✏️ Main Editor
- Full TOML configuration editing
- Syntax highlighting (monospace font)
- Real-time validation feedback
- Save & Validate buttons

## 📝 Step-by-Step Usage

### Step 1: View Existing Personas

The sidebar shows all available personas:
- `default` - General-purpose assistant
- `executive_coach` - Coaching-focused persona

Click any persona to view/edit it.

### Step 2: Edit a Persona

1. Select a persona from sidebar
2. The TOML configuration appears in the editor
3. Make your changes
4. Click "Validate" to check syntax
5. Click "Save" to apply changes

### Step 3: Manage Phone Numbers

To control who can access the persona:

1. Find the `[whatsapp]` section in TOML
2. Edit `allowed_phone_numbers`:
   ```toml
   [whatsapp]
   allowed_phone_numbers = [
       "91-98765-43210",      # Client 1
       "91-87654-32109",      # Client 2
       "91-99999-99999",      # New client
   ]
   enabled = true
   ```
3. Save the persona

### Step 4: Customize Prompt

To change how the AI responds:

1. Find `[personality]` section
2. Edit `prompt_template`:
   ```toml
   [personality]
   prompt_template = """You are an expert business consultant...
   
   When answering:
   - Be strategic
   - Focus on ROI
   - Suggest frameworks
   
   Use tools when available: %s
   Action format: %s
   """
   ```
3. Save the persona

### Step 5: Create New Persona

1. Click "+ New" in sidebar
2. Enter name (e.g., `sales_coach`)
3. Click "Create"
4. Default template appears
5. Customize and save

### Step 6: Test Changes

After saving in control plane:

```bash
# Restart orchestrator with new persona
PERSONA_NAME=sales_coach go run ./cmd/orchestrator/main.go

# Restart bot with new persona
PERSONA_NAME=sales_coach node app.js
```

## 🎨 UI Design Details

### Color Scheme

The UI uses a warm, minimalist beige palette:

| Element | Color | Hex |
|---------|-------|-----|
| Background | Light Beige | `#f5f1e8` |
| Sidebar | Off-White | `#faf7f2` |
| Text | Dark Brown | `#5a5450` |
| Accents | Warm Beige | `#d9cdbf` |
| Borders | Light Beige | `#e8dfd6` |
| Buttons | White | `#ffffff` |

### Typography

- Font: Segoe UI, Tahoma, Geneva, Verdana, sans-serif
- Headings: 600 weight, larger size
- Body: 400 weight, 13-14px
- Code: Monaco, Menlo, Ubuntu Mono (12px)

### Responsive

- Desktop: Full sidebar + editor
- Tablet: Sidebar becomes horizontal
- Mobile: Stacked layout

## 🔧 API Reference

All operations are done through the browser, but here are the backend APIs:

### List Personas
```
GET /api/personas
```

### Get Persona
```
GET /api/personas/{name}
```

### Save Persona
```
POST /api/personas/{name}
Body: { "content": "..." }
```

### Delete Persona
```
DELETE /api/personas/{name}
```

### Validate TOML
```
POST /api/personas/{name}/validate
Body: { "content": "..." }
```

### Server Status
```
GET /api/status
```

## 💡 Tips & Tricks

### Backup Your Personas

TOML files are plain text - you can backup manually:

```bash
cp -r gateway-service/config/personas ~/backups/personas
```

### Copy a Persona

1. Click on persona A to view it
2. Select all (Ctrl+A / Cmd+A)
3. Copy (Ctrl+C / Cmd+C)
4. Click "+ New"
5. Create new persona
6. Paste content into editor
7. Modify as needed
8. Save

### Revert Changes

If you make a mistake:

1. Use browser back button won't help
2. Instead, manually restore from backup or:
   - Delete the persona
   - Recreate from template
   - Re-add your correct configuration

### Validate Before Saving

Always click "Validate" before "Save":
- Catches syntax errors
- Prevents corrupting TOML
- Shows exact error location

## 📊 Real-World Example

### Scenario: Add New Client

**Goal:** Create coaching persona for new client group

1. **Open Control Plane**
   ```
   http://localhost:3000
   ```

2. **Create New Persona**
   - Click "+ New"
   - Name: `client_group_acme`
   - Create

3. **Customize Prompt**
   - Edit `prompt_template`
   - Add company-specific context
   - Example:
     ```toml
     prompt_template = """You are a strategic advisor to ACME Corp.
     
     Focus areas:
     - Digital transformation
     - Supply chain optimization
     - Cost reduction
     
     Tools: %s
     """
     ```

4. **Add Client Numbers**
   - Update `allowed_phone_numbers`:
     ```toml
     [whatsapp]
     allowed_phone_numbers = [
         "91-88888-88888",  # Client 1
         "91-77777-77777",  # Client 2
     ]
     ```

5. **Validate & Save**
   - Click "Validate" → Success
   - Click "Save" → Saved

6. **Deploy**
   - Stop current bot/orchestrator
   - Start with new persona:
     ```bash
     PERSONA_NAME=client_group_acme node app.js
     PERSONA_NAME=client_group_acme go run ./cmd/orchestrator/main.go
     ```

7. **Test**
   - Scan WhatsApp QR
   - Send: `Self Can you help with supply chain?`
   - Bot responds with ACME-focused guidance

## 🚨 Troubleshooting

### Control Plane Won't Start

```bash
# Check if port 3000 is in use
lsof -i :3000

# Use different port
PORT=4000 npm start
```

### Cannot Find Personas Directory

Control plane logs the path:
```
📂 Using personas directory: ...
```

Ensure the path exists and contains TOML files.

### TOML Validation Error

Common issues:
- Missing quotes around strings
- Inconsistent indentation
- Invalid array syntax

Example error:
```
Invalid TOML: Unterminated string at line 5, column 10
```

Fix: Look at line 5 in editor, fix the syntax.

### Changes Don't Take Effect

Remember:
1. Save in control plane ✓
2. Restart orchestrator with `PERSONA_NAME` ✓
3. Restart bot with `PERSONA_NAME` ✓

Changes don't auto-apply - you must restart services.

## 📚 Documentation Map

- **This Guide**: `CONTROL_PLANE_GUIDE.md`
- **Control Plane README**: `persona-control-plane/README.md`
- **Persona Config**: `gateway-service/config/personas/README.md`
- **Quick Start**: `PERSONA_QUICKSTART.md`
- **WhatsApp Bot**: `approach-road/wa-echo-loop/PERSONA_INTEGRATION.md`

## 🎓 Learning Path

### Beginner
1. Start control plane
2. Open browser to http://localhost:3000
3. View `default` persona
4. See how TOML is structured
5. Make small change (e.g., tone: "friendly")
6. Validate & save
7. Restart orchestrator to see changes

### Intermediate
1. Create new persona from scratch
2. Customize prompt for specific use case
3. Add phone numbers for authorized users
4. Test with WhatsApp bot
5. Iterate on prompt based on responses

### Advanced
1. Manage multiple personas for different clients
2. Experiment with prompt templates
3. Use control plane as part of deployment workflow
4. Backup and version control personas
5. Build integrations with external systems

## ✅ Checklist

- [ ] Installed dependencies: `npm install` in persona-control-plane
- [ ] Started control plane: `npm start` (port 3000)
- [ ] Opened browser: http://localhost:3000
- [ ] Saw persona list in sidebar
- [ ] Clicked a persona and saw TOML editor
- [ ] Clicked "Validate" successfully
- [ ] Made a small change and saved
- [ ] Restarted orchestrator/bot with new persona
- [ ] Verified changes took effect
- [ ] Created new persona successfully
- [ ] Added phone numbers to whitelist
- [ ] Tested phone validation with WhatsApp bot

---

**Now you have a complete control plane for managing personas without writing any code!** 🎉

Just open http://localhost:3000 and start managing your personas.
