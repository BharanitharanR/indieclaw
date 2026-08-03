# WhatsApp Integration with Persona Configuration

This WhatsApp bot now loads its configuration from the persona TOML files created in `gateway-service/config/personas/`.

## 🚀 Quick Start

### 1. Install Dependencies

```bash
npm install
# This will install the new 'toml' package
```

### 2. Run with Default Persona

```bash
node app.js
```

The bot will:
- Load `config/personas/default.toml` from the gateway-service directory
- Use the default trigger prefix "Self"
- Accept messages from any sender (if no phone numbers configured)

### 3. Run with Executive Coach Persona

```bash
export PERSONA_NAME=executive_coach
node app.js
```

The bot will:
- Load the executive coach persona
- Only accept messages from configured client phone numbers
- Use the configured trigger prefix

## 📋 Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `PERSONA_NAME` | `"default"` | Which persona TOML file to load |
| `TRIGGER_PREFIX` | `"Self"` | Message prefix to trigger bot |

### Examples

```bash
# Use default persona, default prefix
node app.js

# Use executive coach persona
PERSONA_NAME=executive_coach node app.js

# Use custom persona with custom prefix
PERSONA_NAME=executive_coach TRIGGER_PREFIX="Coach::" node app.js
```

## 🔧 How It Works

### Startup Flow

```
1. App starts
   ↓
2. Read PERSONA_NAME env var (default: "default")
   ↓
3. Load persona TOML file from gateway-service/config/personas/
   ↓
4. Log persona info: name, tone, allowed numbers
   ↓
5. Connect to WhatsApp
   ↓
6. Listen for incoming messages
```

### Message Handling Flow

```
1. WhatsApp message arrives from a user
   ↓
2. Skip if status message or broadcast
   ↓
3. Skip if group message (@g.us)
   ↓
4. Extract sender's phone number from JID
   ↓
5. Validate against persona's allowed_phone_numbers
   ├─ If configured and not allowed → Reject
   └─ If not configured → Accept
   ↓
6. Check message starts with trigger prefix
   ├─ If not → Skip
   └─ If yes → Process
   ↓
7. Extract prompt (remove prefix)
   ↓
8. Download media if attached (image → use vision model)
   ↓
9. Send to orchestrator via gRPC with location context
   ↓
10. Reply with AI response
```

## 📝 Configuration

### Phone Number Validation

The persona's `[whatsapp]` section controls which phone numbers can interact:

```toml
[whatsapp]
allowed_phone_numbers = [
    "91-98765-43210",    # Format: country code + number
    "91-87654-32109",
]
enabled = true
```

Phone numbers are normalized for comparison (removes `-`, spaces, parentheses, `+`).

**Examples:**
- `919361315379` ✅ Matches
- `91-9361315379` ✅ Matches
- `+91 93613 15379` ✅ Matches
- `+91-9361-315379` ✅ Matches

### Trigger Prefix

By default, messages must start with "Self":
```
User: Self What's the weather?
Bot: <response>
```

To use a custom prefix per persona, set environment variable:
```bash
TRIGGER_PREFIX="Coach::" node app.js
```

Now messages must start with "Coach::":
```
User: Coach:: What's the weather?
Bot: <response>
```

## 🔐 Security

### Phone Number Whitelist

Only configured phone numbers can send messages:
- If persona has `allowed_phone_numbers`: Only those numbers work
- If list is empty: All numbers are accepted (no restriction)
- If `enabled: false`: All messages rejected

### Message Filtering

Additional safeguards:
- Group messages (@g.us) are always rejected
- Status messages are ignored
- Broadcast messages (@broadcast) are ignored
- Empty messages are skipped

## 📊 Logging

The bot logs all important events:

```
📝 Persona: Executive Coach Pro (v1.0) | Tone: coaching | Allowed Numbers: 2
⚡ Scan this QR Code with WhatsApp:
✅ WhatsApp Bot ready!
📍 Current Location: Hyderabad, Telangana, India

DEBUG: Message received from: 919361315379@c.us Body: Self Hello
✅ Phone number 919361315379 is allowed
Processing message...
✨ Replied via qwen3:8b
```

## ⚠️ Error Handling

### Persona Not Found

```
❌ Persona file not found for 'my_persona' (tried: ...)
⚠️  No phone number validation configured. Accepting from: 919361315379
```

The bot falls back to accepting all numbers if persona not found.

### gRPC Connection Error

```
❌ gRPC Error: ...
Bot sends: "⚠️ Error communicating with the backend."
```

### Image Processing Error

If image download fails, the bot skips media and processes text only.

## 🛠️ Troubleshooting

### "Persona file not found"

**Cause:** PERSONA_NAME doesn't match a TOML file in the personas directory

**Fix:** 
1. Check `gateway-service/config/personas/` has the file
2. Verify filename (case-sensitive on Linux/Mac)
3. Ensure TOML file is valid TOML syntax

### Phone number not working

**Cause:** Number not in persona's `allowed_phone_numbers` list

**Fix:**
1. Add your number to `allowed_phone_numbers` in the TOML file
2. Ensure format matches (can use separators: `91-9876543210`)
3. Restart bot after changing TOML file

### Bot doesn't respond to "Self" messages

**Cause:** Either phone not allowed OR message format wrong

**Fix:**
1. Check logs for "✅ Phone number X is allowed"
2. Ensure message format is exactly: `Self <your prompt>`
3. No extra characters before "Self"

### gRPC connection fails

**Cause:** Orchestrator not running or wrong gateway URL

**Fix:**
1. Start orchestrator: `go run ./cmd/orchestrator/main.go`
2. Check `grpcClient.js` has correct URL
3. Verify network connection between bot and orchestrator

## 🔄 Changing Personas

To switch personas:

1. **Edit the TOML file** (add/remove phone numbers, change settings)
2. **Restart the bot**: `Ctrl+C` then `node app.js`
3. **Or use environment variable:**
   ```bash
   PERSONA_NAME=executive_coach node app.js
   ```

Changes take effect immediately after restart.

## 📖 Persona File Structure

See `gateway-service/config/personas/README.md` for complete persona file documentation.

Quick reference:
```toml
[persona]
name = "Display Name"
version = "1.0"

[models]
text_model = "qwen3:8b"
vision_model = "gemma4:e2b"

[personality]
prompt_template = "..."  # Not used yet in this adapter
tone = "professional"
# ... other settings

[whatsapp]
allowed_phone_numbers = ["91-9876543210"]
enabled = true
```

## 🚀 Next Steps

1. Create custom persona for your use case
2. Add authorized client phone numbers
3. Customize trigger prefix if needed
4. Test messaging from different numbers
5. Monitor logs for authorization/errors

## 📞 Example Setup

**For Executive Coach Use Case:**

1. Create/edit persona:
   ```bash
   # Already exists at: gateway-service/config/personas/executive_coach.toml
   ```

2. Add client phone numbers to TOML file

3. Start bot:
   ```bash
   PERSONA_NAME=executive_coach node app.js
   ```

4. Clients can now message:
   ```
   Self <coaching question>
   ```

5. Bot responds with coaching-focused responses (from orchestrator's persona system)

---

**Questions?** Check `gateway-service/config/personas/README.md` for persona configuration details.
