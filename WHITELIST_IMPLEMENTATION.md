# Whitelist Implementation — Contact Name Based Registration

**Date:** 2026-08-06  
**Status:** ✅ Implemented  
**Feature:** WhatsApp contact name-based whitelist for routing and registration flow

---

## Overview

The whitelist feature tracks registered users by WhatsApp contact name (not phone number, since WhatsApp doesn't reliably transmit phone numbers). Contact names can follow a configurable prefix pattern (e.g., `CLAREO-STUDENT-Alice`), but the system is **lenient** — it accepts ANY contact name format.

**Key Behavior:**
- ✅ Coaches can set contact names with the configured prefix (recommended)
- ✅ Users can still register even if their contact name doesn't match the prefix
- ℹ️ Format mismatches are logged as warnings but don't block registration
- ✅ Any valid contact name is accepted into the whitelist

**Files Modified/Created:**
1. ✅ `personaConfig.js` — Add whitelist config reading
2. ✅ `whitelist.js` — New whitelist manager module
3. ✅ `app.js` — Integrate whitelist + admin API
4. ✅ `gateway-service/config/personas/default.toml` — Add whitelist section

---

## Configuration

### In Persona TOML File

```toml
[whitelist]
enabled = true
contact_name_prefix = "CLAREO-STUDENT"    # Contacts must start with this
storage = "memory"                         # "memory" or "file"
storage_path = "./whitelist.json"          # For persistent storage
```

### Environment Variables (Optional)

```bash
# Override persona name
PERSONA_NAME=executive_coach

# Override admin API port
ADMIN_PORT=8003
```

---

## How It Works

### 1. Contact Name Format

Coaches/admins must set WhatsApp contact names in a standardized format:

```
{PREFIX}-{Name}

Examples:
  ✓ CLAREO-STUDENT-Alice
  ✓ CLAREO-STUDENT-Bob
  ✓ CLAREO-STUDENT-Jane-Smith
  ✓ APPROACH-PATIENT-John
  
  ✗ Alice                      (missing prefix)
  ✗ CLAREO_STUDENT_ALICE       (wrong separator)
  ✗ student-alice              (wrong prefix)
```

### 2. Routing Decision

When a WhatsApp message arrives:

```
1. Extract contact_name from WhatsApp contact
2. Check: Is contact_name on whitelist?
   
   YES → Route to orchestrator.requests (coaching flow)
   NO  → Route to adiyan.registration.inbound (registration service)
   
3. Log routing decision with contact name and status
```

### 3. Registration Flow

When user sends "Register me in your coaching program":

```
1. Check if contact_name format is valid
2. If invalid format: Reject with guidance
3. If valid format:
   a. Collect occupation, LinkedIn, consent
   b. Create user profile (contact_name as key)
   c. Add contact_name to whitelist
   d. Publish enrollment event
4. User is now on whitelist
```

---

## Admin API Endpoints

The whitelist management API runs on port 8003 (configurable via `ADMIN_PORT` env var).

### 1. List All Whitelisted Contacts
```bash
curl http://localhost:8003/admin/whitelist
```

**Response:**
```json
{
  "enabled": true,
  "initialized": true,
  "contactNamePrefix": "CLAREO-STUDENT",
  "storage": "memory",
  "storagePath": "./whitelist.json",
  "count": 3,
  "entries": [
    "CLAREO-STUDENT-Alice",
    "CLAREO-STUDENT-Bob",
    "CLAREO-STUDENT-Jane"
  ]
}
```

### 2. Add Contact to Whitelist
```bash
curl -X POST http://localhost:8003/admin/whitelist \
  -H "Content-Type: application/json" \
  -d '{"contact_name": "CLAREO-STUDENT-Charlie"}'
```

**Response (Success - Matching Format):**
```json
{
  "success": true,
  "message": "[Whitelist] ✅ Contact whitelisted: CLAREO-STUDENT-Charlie",
  "contactName": "CLAREO-STUDENT-Charlie",
  "formatWarning": null
}
```

**Response (Success - Non-Matching Format):**
```json
{
  "success": true,
  "message": "[Whitelist] ✅ Contact whitelisted: Alice",
  "contactName": "Alice",
  "formatWarning": "Expected format: CLAREO-STUDENT-*"
}
```

**Note:** Any contact name is accepted, even if it doesn't match the configured prefix. The prefix is advisory (for coaches to follow), not a blocking requirement. Format mismatches are logged as warnings but don't prevent registration.

### 3. Check if Contact is Whitelisted
```bash
curl http://localhost:8003/admin/whitelist/check/CLAREO-STUDENT-Alice
```

**Response:**
```json
{
  "contact_name": "CLAREO-STUDENT-Alice",
  "is_whitelisted": true,
  "is_valid_format": true,
  "expected_prefix": "CLAREO-STUDENT"
}
```

### 4. Remove Contact from Whitelist
```bash
curl -X DELETE http://localhost:8003/admin/whitelist/CLAREO-STUDENT-Bob
```

**Response:**
```json
{
  "success": true,
  "message": "[Whitelist] ✅ Contact removed: CLAREO-STUDENT-Bob"
}
```

### 5. Health Check
```bash
curl http://localhost:8003/admin/health
```

**Response:**
```json
{
  "status": "ok",
  "whitelist_enabled": true,
  "whitelist_count": 3,
  "whatsapp_client_ready": true,
  "rabbitmq_connected": true
}
```

---

## Storage Options

### Option A: In-Memory Storage (Default)

```toml
[whitelist]
storage = "memory"
```

**Pros:**
- Fast (no I/O)
- Simple
- Good for testing

**Cons:**
- Lost on restart
- Only works on single instance

### Option B: File-Based Storage

```toml
[whitelist]
storage = "file"
storage_path = "./whitelist.json"
```

**Pros:**
- Persistent across restarts
- Simple, no database needed
- Easy to backup

**Cons:**
- Slower (disk I/O)
- Not suitable for high concurrency
- Single-file lock issues

**File Format:**
```json
[
  {
    "contact_name": "CLAREO-STUDENT-Alice",
    "added_at": "2026-08-06T12:30:00Z"
  },
  {
    "contact_name": "CLAREO-STUDENT-Bob",
    "added_at": "2026-08-06T12:31:00Z"
  }
]
```

---

## Console Logging

The whitelist logs routing decisions to console for debugging:

```
[Whitelist] [Routing] ✅ REGISTERED | CLAREO-STUDENT-Alice | → orchestrator.requests
[Whitelist] [Routing] ❌ UNREGISTERED | CLAREO-STUDENT-Charlie | → adiyan.registration.inbound
[Whitelist] ✅ Added CLAREO-STUDENT-David to whitelist
[Whitelist] ✅ Removed CLAREO-STUDENT-Eve from whitelist
```

---

## Integration Points

### With wa-echo-loop
- Extracts contact_name from WhatsApp message
- Makes routing decision (orchestrator vs. registration)
- Logs decisions

### With Registration Service (Coming Next)
- Validates contact_name format
- Creates user profile with contact_name as key
- Publishes enrollment event (includes "add_to_whitelist")

### With approach-road
- Listens for enrollment events
- Updates whitelist when registration completes
- Coaching flow for registered users

---

## Example Workflow

### Step 1: Coach Sets Contact Name

Coach opens WhatsApp and manually edits contact:
```
Old name: +1-555-123-4567
New name: CLAREO-STUDENT-Alice
```

### Step 2: User Sends Registration Message

User: "Register me in your coaching program"

### Step 3: wa-echo-loop Routes

```
Extract: contact_name = "CLAREO-STUDENT-Alice"
Check: Is on whitelist? NO
Route: adiyan.registration.inbound
```

### Step 4: Registration Service Processes

- Validates format: ✓ Starts with CLAREO-STUDENT-
- Collects data
- Creates profile
- Adds to whitelist via Admin API
- Publishes enrollment event

### Step 5: Future Messages

User: "Can you help me with my homework?"

```
Extract: contact_name = "CLAREO-STUDENT-Alice"
Check: Is on whitelist? YES
Route: orchestrator.requests (coaching flow)
```

---

## Testing

### Manual Test 1: Add Contact via API

```bash
# 1. Check current status
curl http://localhost:8003/admin/whitelist

# 2. Add a contact
curl -X POST http://localhost:8003/admin/whitelist \
  -H "Content-Type: application/json" \
  -d '{"contact_name": "CLAREO-STUDENT-TestUser"}'

# 3. Verify it was added
curl http://localhost:8003/admin/whitelist

# 4. Check if specific contact is whitelisted
curl http://localhost:8003/admin/whitelist/check/CLAREO-STUDENT-TestUser
```

### Manual Test 2: Invalid Format

```bash
# Try adding invalid format
curl -X POST http://localhost:8003/admin/whitelist \
  -H "Content-Type: application/json" \
  -d '{"contact_name": "InvalidName"}'

# Should return error with expected format
```

### Manual Test 3: Remove Contact

```bash
# Remove a contact
curl -X DELETE http://localhost:8003/admin/whitelist/CLAREO-STUDENT-TestUser

# Verify removal
curl http://localhost:8003/admin/whitelist
```

---

## Monitoring & Debugging

### Check Whitelist Status

```bash
curl http://localhost:8003/admin/health | jq .
```

### View Console Logs

```bash
# Logs show all routing decisions
docker logs wa-echo-loop | grep "\[Whitelist\]"
```

### Troubleshooting

**Issue: Contact name not being extracted**
- Check WhatsApp contact name is set correctly
- Verify contact name starts with configured prefix
- Check console logs for warnings

**Issue: Whitelist empty after restart**
- If using `storage = "memory"`, whitelist is cleared on restart
- Change to `storage = "file"` for persistence
- Or use registration service API to re-add contacts

**Issue: Admin API not responding**
- Check `ADMIN_PORT` environment variable
- Default is port 8003
- Verify port is not already in use

---

## Configuration Examples

### Example 1: Student Coaching Program

```toml
[whitelist]
enabled = true
contact_name_prefix = "CLAREO-STUDENT"
storage = "file"
storage_path = "./whitelist-students.json"
```

Usage:
- Coaches set contact names: `CLAREO-STUDENT-Alice`, `CLAREO-STUDENT-Bob`, etc.
- System routes messages from registered students to orchestrator
- Registration service handles unregistered students

### Example 2: Patient Monitoring Program

```toml
[whitelist]
enabled = true
contact_name_prefix = "APPROACH-PATIENT"
storage = "file"
storage_path = "./whitelist-patients.json"
```

Usage:
- Clinics set contact names: `APPROACH-PATIENT-John`, `APPROACH-PATIENT-Jane`, etc.
- Patients message the bot and are automatically routed to appropriate service

### Example 3: Development/Testing (Disabled)

```toml
[whitelist]
enabled = false
```

Usage:
- All messages route to orchestrator
- No whitelist checks
- Useful for development and testing

---

## Next Steps

1. **Update wa-echo-loop** to use whitelist for routing decisions
2. **Build registration service** to handle registration flow and add users to whitelist
3. **Add persistence layer** (Redis or database) if needed for high concurrency
4. **Add UI dashboard** for whitelist management (replacing Admin API)
5. **Add metrics/monitoring** for whitelist operations

---

## API Examples (cURL)

```bash
# Get all whitelisted contacts
curl -s http://localhost:8003/admin/whitelist | jq '.entries'

# Add multiple contacts
for name in Alice Bob Charlie; do
  curl -X POST http://localhost:8003/admin/whitelist \
    -H "Content-Type: application/json" \
    -d "{\"contact_name\": \"CLAREO-STUDENT-$name\"}"
done

# Export whitelist to file
curl -s http://localhost:8003/admin/whitelist | jq '.entries' > exported-whitelist.json

# Health check
curl http://localhost:8003/admin/health | jq .
```

---

**Status:** ✅ Ready to use  
**Next:** Implement registration service routing and enrollment flow
