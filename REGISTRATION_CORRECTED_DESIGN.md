# Adiyan Registration System — Corrected Design (Contact Name Based)

**Date:** 2026-08-06  
**Status:** Revised Design Specification  
**Key Change:** WhatsApp contact name (not phone number) is the unique user identifier

---

## Design Clarification

The registration system uses **contact name as the unique identifier** because:
- ❌ WhatsApp does NOT reliably transmit phone numbers
- ✅ WhatsApp contact names ARE reliable and user-manageable
- ✅ Contact names can be standardized with configurable prefixes (e.g., `CLAREO-STUDENT-*`)

---

## Architecture Overview

```
User (WhatsApp)
  │
  ├─ Types: "Register me in your coaching program"
  │
  └─→ wa-echo-loop (Node.js) [ROUTING GATEWAY]
        │
        ├─ Extracts: contact_name from WhatsApp
        ├─ Checks: Is contact_name on whitelist?
        │
        ├─ YES → Route to orchestrator.requests (normal coaching flow)
        │
        └─ NO → Route to adiyan.registration.inbound (registration service)
             │
             └─→ adiyan-registration-service (Go)
                  │
                  ├─ Conversational flow (occupation, LinkedIn, consent)
                  ├─ Validate contact_name format (CLAREO-STUDENT-*)
                  ├─ Create user profile in user-context-service
                  ├─ Add contact_name to whitelist
                  └─ Publish enrollment event
                       │
                       └─→ approach-road enrollment
```

---

## Critical Design Points

### 1. Registration Trigger

**Exact message:** `"Register me in your coaching program"`

```javascript
// In wa-echo-loop
if (message.startsWith("Register me in your coaching program")) {
  // Handle registration
}
```

### 2. Whitelist-Based Routing

**wa-echo-loop makes routing decision based on whitelist:**

```
Message arrives
  ↓
Extract contact_name
  ↓
Is contact_name on whitelist?
  ├─ YES → orchestrator.requests (coaching flow)
  └─ NO → adiyan.registration.inbound (registration service)
```

### 3. Contact Name as Primary Key

**Before (OLD - Unreliable):**
```
User Profile:
  phone_number: "+1-555-123-4567"  ❌ Unreliable
  name: "Alice"
```

**After (NEW - Reliable):**
```
User Profile:
  contact_name: "CLAREO-STUDENT-Alice"  ✅ Definitive
  whatsapp_message_id: "wamsg_abc123"
```

### 4. Contact Name Format (Configurable)

**Coaches set contact names in WhatsApp with standardized format:**

```yaml
Format Configuration:
  prefix: "CLAREO-STUDENT"      # Can be changed
  separator: "-"
  pattern: "^CLAREO-STUDENT-.*$"

Examples:
  ✓ CLAREO-STUDENT-Alice
  ✓ CLAREO-STUDENT-Bob
  ✓ CLAREO-STUDENT-123
  
  ✗ alice-student          (missing prefix)
  ✗ CLAREO-STUDENT         (missing name part)
  ✗ CLAREO_STUDENT_ALICE   (wrong separator)
```

**Configuration file:**
```yaml
# config.yaml or environment
contact_name:
  format_enabled: true
  prefix: "CLAREO-STUDENT"
  pattern_regex: "^CLAREO-STUDENT-.*$"
  validation_required: true
  error_message: "Contact name must be: CLAREO-STUDENT-YourName"
```

---

## Message Flow Sequence

### Complete Registration Flow

```
1. User Action
   User: "Register me in your coaching program"
   WhatsApp Contact Name: "CLAREO-STUDENT-Alice"
   
2. wa-echo-loop Gateway
   Extracts: contact_name = "CLAREO-STUDENT-Alice"
   Checks: Is "CLAREO-STUDENT-Alice" in whitelist?
   Result: NO (first-time user)
   Route: adiyan.registration.inbound
   
3. Registration Service Consumes
   Message: {
     contact_name: "CLAREO-STUDENT-Alice",
     message: "Register me in your coaching program",
     message_id: "wamsg_abc123",
     timestamp: "2026-08-06T12:00:00Z"
   }
   
4. Registration Flow Begins
   Step 1: Validate contact_name format
     ✓ Matches CLAREO-STUDENT-* → Proceed
     ✗ Doesn't match → Reject with guidance
   
   Step 2: Collect occupation
     System: "What's your role?"
     Publish: adiyan.registration.outbound
     wa-echo-loop sends: "What's your role?"
   
   Step 3: Collect LinkedIn (optional)
     User: "linkedin.com/in/alice"
   
   Step 4: Consent
     User: "I agree"
   
5. User Profile Creation
   Create in user-context-service:
   {
     contact_name: "CLAREO-STUDENT-Alice",
     user_id: "USER_12345",
     occupation: "Student",
     linkedin_profile_url: "linkedin.com/in/alice",
     consent_agreed: true,
     status: "active"
   }
   
6. Whitelist Addition
   Add to whitelist:
   {
     contact_name: "CLAREO-STUDENT-Alice",
     added_at: "2026-08-06T12:05:00Z",
     status: "active"
   }
   
7. Enrollment Event
   Publish: adiyan.approach_road.enrollment
   {
     event_type: "user_registered",
     contact_name: "CLAREO-STUDENT-Alice",
     user_id: "USER_12345",
     add_to_whitelist: true
   }
   
8. Completion
   System: "Welcome to coaching, CLAREO-STUDENT-Alice! 🎉"
   wa-echo-loop: Now recognizes this contact as registered
```

### Subsequent Messages (After Registration)

```
User: "Hi, how do I start?"
Contact Name: "CLAREO-STUDENT-Alice"

wa-echo-loop routing:
  Is "CLAREO-STUDENT-Alice" on whitelist? YES
  Route to: orchestrator.requests
  
Message flows to orchestrator (coaching flow)
```

---

## wa-echo-loop Modifications

### Current State (Before Changes)
- Routes all messages to orchestrator
- No filtering or whitelist concept

### New State (After Changes)

**1. Whitelist Module:**
```javascript
class WhitelistManager {
  async addToWhitelist(contactName) {
    // Persist to user-context-service or database
  }
  
  async isOnWhitelist(contactName) {
    // Check if contact_name is registered
  }
  
  async loadWhitelistOnStartup() {
    // Load all active users into memory cache for fast lookup
  }
}
```

**2. Routing Logic:**
```javascript
async function handleIncomingMessage(msg) {
  const contactName = msg.contact_name;  // From WhatsApp
  const messageText = msg.body;
  
  // Decision 1: Is this a registration message?
  if (messageText.startsWith("Register me in your coaching program")) {
    if (await whitelist.isOnWhitelist(contactName)) {
      // Already registered
      await publish('orchestrator.requests', msg);
    } else {
      // New registration
      await publish('adiyan.registration.inbound', msg);
    }
  } else {
    // Regular message
    if (await whitelist.isOnWhitelist(contactName)) {
      // Registered user → coaching flow
      await publish('orchestrator.requests', msg);
    } else {
      // Unregistered user → ask to register
      await publish('adiyan.registration.inbound', {
        ...msg,
        note: "unregistered_user"
      });
    }
  }
}
```

**3. Whitelist Update Listener:**
```javascript
// Listen for enrollment events
ch.consume('adiyan.approach_road.enrollment', async (msg) => {
  const event = JSON.parse(msg.content);
  
  if (event.event_type === 'user_registered') {
    const contactName = event.contact_name;
    await whitelist.addToWhitelist(contactName);
    console.log(`✓ Whitelist updated: ${contactName}`);
  }
  
  ch.ack(msg);
});
```

---

## User-Context-Service Modifications

### Current State
- Stores user profiles keyed by phone_number
- Fields: phone_number, name, etc.

### New State

**1. Data Model Change:**
```javascript
// OLD
{
  user_id: "USER_123",
  phone_number: "+1-555-1234",  ❌ Remove this
  full_name: "Alice Smith",
}

// NEW
{
  contact_name: "CLAREO-STUDENT-Alice",  ✅ Primary key
  user_id: "USER_123",
  occupation: "Student",
  linkedin_profile_url: "...",
  whatsapp_message_id: "wamsg_abc123"
}
```

**2. New Endpoints:**
```
POST /profiles
  Create user (contact_name as key)

GET /profiles/{contact_name}
  Get user by contact_name

GET /profiles?contact_name_prefix=CLAREO-STUDENT
  List users matching prefix pattern

POST /whitelist
  Add contact_name to whitelist

GET /whitelist/{contact_name}
  Check if contact_name is registered

DELETE /whitelist/{contact_name}
  Remove from whitelist (for admin)
```

**3. File Storage:**
```
Old: ~/.adiyan/users/USER_123.json
New: ~/.adiyan/users/CLAREO-STUDENT-Alice.json

File content indexed by contact_name for faster lookups
```

---

## Registration Service (Go) Changes

### Core Logic (Minimal Changes)

**1. Session State:**
```go
type RegistrationSession struct {
  SessionID    string                 // Unique session
  ContactName  string                 // PRIMARY KEY (extracted from WhatsApp)
  CurrentStep  string                 // greeting, format_check, occupation, etc.
  StepsCompleted []string
  Data         map[string]interface{}
  CreatedAt    time.Time
  LastActivity time.Time
}
```

**2. Whitelist Publish on Completion:**
```go
func (s *Service) onRegistrationComplete(session *RegistrationSession) error {
  // Create user profile with contact_name
  profile := map[string]interface{}{
    "contact_name": session.ContactName,  // KEY CHANGE
    "user_id": generateUserID(),
    "occupation": session.Data["occupation"],
    "consent_agreed": true,
    "status": "active",
  }
  
  // Store in UCS
  s.ucsClient.CreateProfile(profile)
  
  // Publish enrollment + whitelist event
  event := map[string]interface{}{
    "event_type": "user_registered",
    "contact_name": session.ContactName,
    "add_to_whitelist": true,
  }
  
  s.rabbitmqCh.Publish("", "adiyan.approach_road.enrollment", event)
  
  return nil
}
```

---

## Configuration Example

```yaml
# registration-config.yaml

# WhatsApp Settings
whatsapp:
  adapter: "wa-echo-loop"
  registration_trigger: "Register me in your coaching program"

# Contact Name Format
contact_name_format:
  enabled: true
  prefix: "CLAREO-STUDENT"        # Configurable
  separator: "-"
  pattern_regex: "^CLAREO-STUDENT-.*$"
  validation_error: "Contact name must start with: CLAREO-STUDENT-"
  min_length: 15                    # CLAREO-STUDENT-A = 15 chars minimum
  max_length: 50

# User-Context-Service
user_context_service:
  url: "http://localhost:8001"
  timeout: "5s"
  whitelist_endpoint: "/whitelist"

# RabbitMQ Queues
message_queues:
  registration_inbound: "adiyan.registration.inbound"
  registration_outbound: "adiyan.registration.outbound"
  orchestrator_requests: "orchestrator.requests"
  approach_road_enrollment: "adiyan.approach_road.enrollment"

# Whitelist
whitelist:
  storage: "ucs"                   # or "memory" for dev
  cache_ttl: "1h"
  background_sync_interval: "5m"

# Logging
logging:
  level: "info"
  log_routing_decisions: true      # Log each routing decision
  log_whitelist_operations: true   # Log whitelist adds/removes
```

---

## Deployment Checklist

- [ ] Update wa-echo-loop:
  - [ ] Add whitelist manager module
  - [ ] Add routing logic (registration vs. orchestrator)
  - [ ] Add whitelist update listener
  - [ ] Deploy and test whitelist logic

- [ ] Update user-context-service:
  - [ ] Replace phone_number field with contact_name
  - [ ] Add whitelist endpoints
  - [ ] Update file storage path (contact_name-based)
  - [ ] Add contact_name validation
  - [ ] Deploy and test

- [ ] Build registration-service (Go):
  - [ ] Contact name format validation
  - [ ] Publish whitelist event on completion
  - [ ] Test end-to-end flow
  - [ ] Deploy to staging

- [ ] Testing:
  - [ ] User with proper contact name (CLAREO-STUDENT-Alice) can register
  - [ ] User with improper name is rejected with guidance
  - [ ] Whitelist is updated after registration
  - [ ] Registered users route to orchestrator on subsequent messages
  - [ ] Unregistered users route to registration service

---

## Success Criteria

✓ Contact name extracted reliably from WhatsApp  
✓ Whitelist-based routing works (registered → orchestrator, unregistered → registration)  
✓ User profiles stored with contact_name as key  
✓ Contact name format validation enforced  
✓ No phone numbers stored or used in system  
✓ User-context-service updated to use contact_name  
✓ Coaches can manage contact names in WhatsApp  

---

## FAQ

**Q: What if a user changes their WhatsApp contact name?**  
A: They become unregistered (new contact_name). They'd need to register again with the new name. Coaches should NOT change contact names after registration.

**Q: Can we use email or phone as fallback if contact_name is unreliable?**  
A: No. Contact name is the design. If WhatsApp contact names are unreliable, we have a bigger platform problem.

**Q: How do we handle contact name collisions?**  
A: The format should be specific enough (e.g., CLAREO-STUDENT-Alice-Smith-2026-08-06) to avoid collisions. Coaches manage this.

**Q: What if a coach sets a wrong contact name format?**  
A: User sees error: "Your contact name must be: CLAREO-STUDENT-YourName. Ask your coach to correct it."

**Q: How do we migrate existing users from phone-based to contact-name-based?**  
A: One-time migration script: Map phone_number → contact_name, recreate profiles with new keys.

---

**Status:** Ready for engineering implementation  
**Next:** Modify wa-echo-loop, update user-context-service, build registration service
