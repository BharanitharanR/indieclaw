# Indieclaw Personas - Quick Start Guide

Everything is now configured to work with persona TOML files!

## 🎯 What Changed

Both the **Go Orchestrator** and **Node.js WhatsApp Bot** now load configuration from the same persona TOML files.

### Backend (Go Orchestrator)
- **Location:** `gateway-service/internal/orchestrator/persona.go`
- **Config Files:** `gateway-service/config/personas/*.toml`
- **Loading:** Automatic at startup via `PERSONA_NAME` env var

### Frontend (WhatsApp Bot)
- **Location:** `approach-road/wa-echo-loop/personaConfig.js`
- **Config Files:** Loads from `gateway-service/config/personas/*.toml`
- **Loading:** Automatic at startup via `PERSONA_NAME` env var

---

## 🚀 Running the Complete System

### Terminal 1: Start the Orchestrator

```bash
cd gateway-service

# With default persona
go run ./cmd/orchestrator/main.go

# Or with executive coach persona
PERSONA_NAME=executive_coach go run ./cmd/orchestrator/main.go
```

You should see:
```
✅ Persona loaded: Default Assistant
🚀 Orchestrator running on :9000 [Persona: Default Assistant, ...]
```

### Terminal 2: Start the WhatsApp Bot

```bash
cd approach-road/wa-echo-loop

# Install dependencies (first time only)
npm install

# With default persona
node app.js

# Or with executive coach persona
PERSONA_NAME=executive_coach node app.js
```

You should see:
```
📝 Persona: Default Assistant (v1.0) | Tone: professional | Allowed Numbers: 0
⚡ Scan this QR Code with WhatsApp:
✅ WhatsApp Bot ready!
```

---

## 📝 Persona Files

Both applications read from the same directory:

```
gateway-service/config/personas/
├── default.toml                    # General-purpose assistant
├── executive_coach.toml            # For coaching use case
└── README.md                       # Complete configuration guide
```

### Example: executive_coach.toml

```toml
[persona]
name = "Executive Coach Pro"
version = "1.0"

[models]
text_model = "qwen3:8b"
vision_model = "gemma4:e2b"

[personality]
prompt_template = "You are an elite executive coach..."
tone = "coaching"
response_style = "strategic_guidance"

[whatsapp]
allowed_phone_numbers = [
    "91-98765-43210",  # Client 1
    "91-87654-32109",  # Client 2
]
enabled = true
```

---

## 🔄 The Complete Flow

```
User sends WhatsApp message
    ↓
WhatsApp Bot (Node.js)
├─ Loads: persona config from TOML
├─ Validates: sender's phone number
├─ Checks: message trigger prefix
└─ Extracts: prompt + media
    ↓
Sends via gRPC to Orchestrator
    ↓
Orchestrator (Go)
├─ Loads: persona config from TOML
├─ Gets: persona's prompt template
├─ Applies: tone & style guidelines
├─ Calls: LLM with personalized prompt
└─ Returns: AI response
    ↓
WhatsApp Bot receives response
    ↓
Bot replies to user on WhatsApp
```

---

## 🔐 Security & Validation

### Phone Number Whitelist

The `allowed_phone_numbers` in the persona controls access:

```toml
[whatsapp]
allowed_phone_numbers = [
    "91-98765-43210",      # Only these numbers can send
    "91-87654-32109",      # Messages to the bot
]
enabled = true
```

**Phone number formats supported:**
- `919876543210` ✅
- `91-9876543210` ✅
- `+91 98765 43210` ✅
- `+91-9876-543210` ✅

All formats are normalized and compared.

### Unauthorized Access

If a number isn't in the whitelist:

```
❌ Phone number 919111111111 is NOT allowed
```

Bot silently ignores the message.

---

## 🛠️ Environment Variables

### Orchestrator

```bash
# Which persona to load
export PERSONA_NAME=executive_coach

# Optional: Override persona's models
export TEXT_MODEL=mistral:latest
export VISION_MODEL=llava:latest

# Start
go run ./cmd/orchestrator/main.go
```

### WhatsApp Bot

```bash
# Which persona to load
export PERSONA_NAME=executive_coach

# Which message prefix triggers the bot
export TRIGGER_PREFIX="Coach::"

# Start
node app.js
```

---

## 📋 Quick Commands

### Create New Persona

1. Copy template:
   ```bash
   cp gateway-service/config/personas/default.toml \
      gateway-service/config/personas/my_persona.toml
   ```

2. Edit the file with your settings:
   ```bash
   nano gateway-service/config/personas/my_persona.toml
   ```

3. Use it:
   ```bash
   PERSONA_NAME=my_persona node app.js
   ```

### Add Client Phone Number

Edit the TOML file:

```toml
[whatsapp]
allowed_phone_numbers = [
    "91-98765-43210",      # Existing
    "91-99999-99999",      # New client
]
enabled = true
```

Then restart bot:
```bash
PERSONA_NAME=executive_coach node app.js
```

### Test Phone Validation

1. Start bot with persona that has phone numbers:
   ```bash
   PERSONA_NAME=executive_coach node app.js
   ```

2. Send message from authorized number:
   ```
   ✅ Phone number 919876543210 is allowed
   Processing message...
   ```

3. Send message from unauthorized number:
   ```
   ❌ Phone number 919111111111 is NOT allowed
   ```

---

## 📊 Real-World Example: Executive Coach

### Setup

1. **Persona file** (`executive_coach.toml`):
   - Prompt focused on coaching
   - Tone: professional, empathetic
   - Clients: 2 authorized numbers

2. **Orchestrator** starts:
   ```bash
   PERSONA_NAME=executive_coach go run ./cmd/orchestrator/main.go
   ```
   - Loads coaching prompt
   - Uses coaching tone in responses

3. **Bot** starts:
   ```bash
   PERSONA_NAME=executive_coach node app.js
   ```
   - Loads same persona config
   - Only allows 2 client numbers
   - Uses trigger prefix "Self"

### Client Interaction

**Client 1 (authorized):**
```
Client: Self What's my Q3 strategy?
Bot: ✅ Authorized → Sends to orchestrator
Orchestrator: (uses coaching prompt) → Crafts response
Bot: <coaching-focused response>
```

**Unknown Number (unauthorized):**
```
Unknown: Self Hello?
Bot: ❌ Not in allowed list → Silently ignores
```

---

## 🎓 Learning Path

1. **Start with defaults:**
   ```bash
   # Terminal 1
   cd gateway-service && go run ./cmd/orchestrator/main.go
   
   # Terminal 2
   cd approach-road/wa-echo-loop && npm install && node app.js
   ```

2. **Edit default persona:**
   - Change prompt, tone, guidelines
   - Restart both apps
   - See differences in responses

3. **Create custom persona:**
   - Copy default.toml to my_persona.toml
   - Customize prompt + phone numbers
   - Test with `PERSONA_NAME=my_persona`

4. **Understand phone validation:**
   - Add your number to `allowed_phone_numbers`
   - Send message → See ✅ "allowed" in logs
   - Remove number → See ❌ "not allowed"

5. **Advanced: Multiple personas:**
   - Different coaching styles
   - Different client groups
   - Switch with env var (no code changes)

---

## 🔍 Debugging

### Check What Persona Loaded

Look for these logs:

**Orchestrator:**
```
✅ Persona loaded: Executive Coach Pro | Models: qwen3:8b (text), gemma4:e2b (vision) | Allowed numbers: 2
🚀 Orchestrator running on :9000 [Persona: Executive Coach Pro, ...]
```

**WhatsApp Bot:**
```
📝 Persona: Executive Coach Pro (v1.0) | Tone: coaching | Allowed Numbers: 2
```

### Check Phone Validation

Look for logs in WhatsApp bot:

```
✅ Phone number 919876543210 is allowed      ← Message accepted
❌ Phone number 919111111111 is NOT allowed  ← Message rejected
⚠️  No phone numbers configured              ← All accepted
```

### TOML Parse Error

If persona file is invalid:

```
❌ Persona file not found for 'bad_persona'
⚠️  No phone number validation configured
```

**Fix:** Validate TOML at https://www.toml-lint.com/

---

## 📚 Documentation

- **Persona Configuration:** `gateway-service/config/personas/README.md`
- **Orchestrator Changes:** `IMPLEMENTATION_SUMMARY.md`
- **WhatsApp Bot Changes:** `WHATSAPP_INTEGRATION_SUMMARY.md`
- **WhatsApp Bot Detailed:** `approach-road/wa-echo-loop/PERSONA_INTEGRATION.md`

---

## ✅ Checklist

- [ ] Backend compiles: `cd gateway-service && go build -v ./cmd/orchestrator/`
- [ ] WhatsApp has Node modules: `cd approach-road/wa-echo-loop && npm install`
- [ ] Persona files exist: `ls gateway-service/config/personas/`
- [ ] Start orchestrator: `PERSONA_NAME=executive_coach go run ./cmd/orchestrator/main.go`
- [ ] Start bot: `PERSONA_NAME=executive_coach node app.js`
- [ ] Scan WhatsApp QR code
- [ ] Send message from authorized number: `Self <prompt>`
- [ ] Verify response uses executive coach tone

---

## 🎉 You're All Set!

Both the orchestrator and WhatsApp bot are now fully configured to use personas. You can:

✅ Create multiple personas for different use cases
✅ Control who can access each persona via phone whitelist
✅ Change personas by setting `PERSONA_NAME` env var
✅ Customize AI behavior without code changes
✅ Scale to support multiple coaches/clients

**Next:** Explore creating a Web UI command center for managing personas without editing files!
