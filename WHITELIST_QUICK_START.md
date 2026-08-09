# Whitelist Feature — Quick Start Guide

**Status:** ✅ Implemented and Ready to Test  
**Implementation Date:** 2026-08-06

---

## What's Been Implemented

### ✅ Files Created
1. **`approach-road/wa-echo-loop/whitelist.js`** (165 lines)
   - WhitelistManager class
   - In-memory + file-based storage
   - Format validation
   - Add/remove/check operations
   - Admin logging

### ✅ Files Modified
1. **`approach-road/wa-echo-loop/personaConfig.js`**
   - Added whitelist config reading from TOML
   - Added getter methods for whitelist settings
   - Updated toString() for status display

2. **`approach-road/wa-echo-loop/app.js`**
   - Imported WhitelistManager
   - Initialize whitelist on startup
   - Integrated contact name extraction
   - Added whitelist check logic
   - Added HTTP admin API (port 8003)

3. **`gateway-service/config/personas/default.toml`**
   - Added `[whitelist]` section
   - Configurable prefix, storage, path

### ✅ Documentation Created
- **`WHITELIST_IMPLEMENTATION.md`** (Full guide with examples)
- **`WHITELIST_QUICK_START.md`** (This file)

---

## How It Works (Simplified)

```
User sends WhatsApp message
      ↓
Extract contact_name from WhatsApp
      ↓
Check whitelist: Is contact_name registered?
      ↓
      YES → Route to orchestrator.requests (coaching)
      NO  → Route to registration.inbound (registration)
            (Accepts any contact_name format - no rejection based on prefix)
```

**Note:** Format validation is lenient. Even if the contact name doesn't match the configured prefix, users can still proceed with registration. The prefix is a guideline for coaches, not a hard requirement.

---

## Getting Started

### Step 1: Configure Prefix in Persona TOML

Edit `gateway-service/config/personas/default.toml`:

```toml
[whitelist]
enabled = true
contact_name_prefix = "CLAREO-STUDENT"    # ← Change this to your program
storage = "memory"                         # or "file" for persistent storage
storage_path = "./whitelist.json"
```

### Step 2: Coach Sets Contact Names in WhatsApp

In WhatsApp, edit contact names to match the pattern:

```
Example for CLAREO-STUDENT prefix:
- CLAREO-STUDENT-Alice
- CLAREO-STUDENT-Bob
- CLAREO-STUDENT-Jane
```

### Step 3: Start wa-echo-loop

```bash
cd approach-road/wa-echo-loop
npm start
```

You'll see:
```
📝 Persona: Default Assistant (v1.0) | Tone: professional | Whitelist: CLAREO-STUDENT-*

[Whitelist] Status: {
  "enabled": true,
  "initialized": true,
  "contactNamePrefix": "CLAREO-STUDENT",
  "storage": "memory",
  "count": 0,
  "entries": []
}

🔧 Admin API running on http://localhost:8003
   GET  /admin/whitelist - List all whitelisted contacts
   POST /admin/whitelist - Add contact to whitelist
   DELETE /admin/whitelist/:name - Remove contact
   GET  /admin/whitelist/check/:name - Check if whitelisted
   GET  /admin/health - Health check
```

### Step 4: Manage Whitelist via Admin API

**Add contacts:**
```bash
curl -X POST http://localhost:8003/admin/whitelist \
  -H "Content-Type: application/json" \
  -d '{"contact_name": "CLAREO-STUDENT-Alice"}'
```

**List all:**
```bash
curl http://localhost:8003/admin/whitelist
```

**Check if whitelisted:**
```bash
curl http://localhost:8003/admin/whitelist/check/CLAREO-STUDENT-Alice
```

---

## How Routing Works Now

### Scenario 1: New User (Not Whitelisted)

```
User WhatsApp contact: "CLAREO-STUDENT-Charlie"
User sends: "Register me in your coaching program"

wa-echo-loop decision:
  Contact: CLAREO-STUDENT-Charlie
  Format valid: ✓ YES
  On whitelist: ✗ NO
  → Route to adiyan.registration.inbound (registration service)
  
Console output:
  [Whitelist] [Routing] ❌ UNREGISTERED | CLAREO-STUDENT-Charlie | → adiyan.registration.inbound
```

### Scenario 2: Registered User

```
User WhatsApp contact: "CLAREO-STUDENT-Alice" (already whitelisted)
User sends: "Can you help me with my homework?"

wa-echo-loop decision:
  Contact: CLAREO-STUDENT-Alice
  On whitelist: ✓ YES
  → Route to orchestrator.requests (coaching flow)
  
Console output:
  [Whitelist] [Routing] ✅ REGISTERED | CLAREO-STUDENT-Alice | → orchestrator.requests
```

### Scenario 3: Non-Matching Format (Still Accepted)

```
User WhatsApp contact: "Alice" (doesn't match CLAREO-STUDENT- prefix)
User sends: "Register me in your coaching program"

wa-echo-loop decision:
  Contact: Alice
  Format valid: ✗ NO (expected: CLAREO-STUDENT-*)
  On whitelist: ✗ NO
  → Route to registration.inbound (STILL PROCEEDS)
  
Console output:
  [Whitelist] [Routing] ❌ UNREGISTERED | Alice | → adiyan.registration.inbound
  [Whitelist] ℹ️  Contact name format: "Alice" (expected: CLAREO-STUDENT-*)
  [Whitelist] ℹ️  Accepting anyway for registration flow

Result: User can still register with contact name "Alice"
         (prefix is advisory, not restrictive)
```

---

## Admin API Commands (Quick Reference)

| Operation | Command |
|-----------|---------|
| **List all** | `curl http://localhost:8003/admin/whitelist` |
| **Add contact** | `curl -X POST http://localhost:8003/admin/whitelist -H "Content-Type: application/json" -d '{"contact_name": "CLAREO-STUDENT-Name"}'` |
| **Check if whitelisted** | `curl http://localhost:8003/admin/whitelist/check/CLAREO-STUDENT-Name` |
| **Remove contact** | `curl -X DELETE http://localhost:8003/admin/whitelist/CLAREO-STUDENT-Name` |
| **Health check** | `curl http://localhost:8003/admin/health` |

---

## Storage Options

### In-Memory (Default - Fast)
```toml
storage = "memory"
```
- Data lost on restart
- Good for development
- Single instance only

### File-Based (Persistent)
```toml
storage = "file"
storage_path = "./whitelist.json"
```
- Survives restarts
- Slower than in-memory
- Easy to backup

---

## Testing the Feature

### Test 1: Add Contact via API

```bash
# 1. Check empty list
curl http://localhost:8003/admin/whitelist

# 2. Add contact
curl -X POST http://localhost:8003/admin/whitelist \
  -H "Content-Type: application/json" \
  -d '{"contact_name": "CLAREO-STUDENT-TestUser"}'

# 3. Verify added
curl http://localhost:8003/admin/whitelist
```

### Test 2: Format Validation (Lenient)

```bash
# Add contact WITHOUT matching prefix format (still succeeds!)
curl -X POST http://localhost:8003/admin/whitelist \
  -H "Content-Type: application/json" \
  -d '{"contact_name": "Alice"}'

# Expected response (success with warning):
# {
#   "success": true,
#   "message": "[Whitelist] ✅ Contact whitelisted: Alice",
#   "contactName": "Alice",
#   "formatWarning": "Expected format: CLAREO-STUDENT-*"
# }

# The contact is added successfully despite not matching the prefix!
```

### Test 3: Check Status

```bash
curl http://localhost:8003/admin/health | jq .
```

---

## Console Logging

All whitelist operations are logged to console:

```
[Whitelist] 📝 Using in-memory storage
[Whitelist] ✅ Added CLAREO-STUDENT-Alice to whitelist
[Whitelist] [Routing] ✅ REGISTERED | CLAREO-STUDENT-Alice | → orchestrator.requests
[Whitelist] ✅ Removed CLAREO-STUDENT-Bob from whitelist
```

---

## Configuration Examples

### For Student Coaching
```toml
[whitelist]
enabled = true
contact_name_prefix = "CLAREO-STUDENT"
storage = "file"
storage_path = "./whitelist-students.json"
```

### For Patient Monitoring
```toml
[whitelist]
enabled = true
contact_name_prefix = "APPROACH-PATIENT"
storage = "file"
storage_path = "./whitelist-patients.json"
```

### For Testing (Disabled)
```toml
[whitelist]
enabled = false
```

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Admin API not responding | Check `ADMIN_PORT` (default 8003), verify port not in use |
| Whitelist empty after restart | Using `memory` storage? Change to `file` for persistence |
| Contact not recognized | Verify WhatsApp contact name matches prefix exactly |
| Format validation not working | Check `contact_name_prefix` in TOML file |
| Logs not showing routing decisions | Verify whitelist is enabled in TOML |

---

## Next: Integration with Registration Service

Once whitelist is working, the next step is to:

1. **Build registration service** (Go) that:
   - Handles "Register me in your coaching program" message
   - Collects occupation, LinkedIn, consent
   - Calls Admin API to add contact to whitelist
   - Publishes enrollment event

2. **Update routing logic** to send unregistered users to registration service

3. **Add user profile creation** that uses contact_name as primary key

---

## Files Summary

| File | Purpose | Status |
|------|---------|--------|
| `whitelist.js` | Whitelist manager | ✅ Created |
| `personaConfig.js` | Config reading | ✅ Updated |
| `app.js` | Integration + API | ✅ Updated |
| `default.toml` | Whitelist config | ✅ Updated |
| `WHITELIST_IMPLEMENTATION.md` | Full documentation | ✅ Created |
| `WHITELIST_QUICK_START.md` | This guide | ✅ Created |

---

## Ready to Use! 🚀

The whitelist feature is fully implemented and ready to test. Start by configuring the contact name prefix in your persona TOML file and running the Admin API to add users to the whitelist.

**Next Steps:**
1. Configure prefix in `default.toml`
2. Start wa-echo-loop
3. Use Admin API to add whitelisted contacts
4. Observe routing decisions in console logs
5. Build registration service for user onboarding

For detailed information, see `WHITELIST_IMPLEMENTATION.md`.
