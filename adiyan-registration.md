# Adiyan Registration System Specification

**Version:** 1.0  
**Date:** 2026-08-06  
**Status:** Design Specification  
**Audience:** Architecture, Backend Engineering, WhatsApp Integration Team

---

## Quick Reference (TL;DR)

**What:** WhatsApp registration via contact name (not phone) + auto-enrollment to approach-road  
**How:** Whitelist-based routing + separate Go microservice, ~250 LOC, uses existing RabbitMQ + user-context-service  
**When:** 4 weeks (setup → beta → GA)  
**Cost:** Zero new infrastructure, two new dependencies (google/uuid, amqp091-go)  
**Key Decisions:**
- Registration trigger: **Message starting with "Register me in your coaching program"**
- Unique identifier: **Contact name** (WhatsApp phone unreliable; contact name is definitive)
- Contact name format: **Configurable prefix pattern** (e.g., `CLAREO-STUDENT-{name}`)
- Routing mechanism: **Whitelist-based**
  - Registered contacts (on whitelist) → flow to orchestrator
  - Unregistered contacts → flow to registration service
- Both wa-echo-loop AND user-context-service use contact_name as primary key
- Auto-enroll to approach-road on completion
- Configurable eviction policy for inactive users
- Manual control panel for user lifecycle management
- Persistent registration state (resume if abandoned)

**Next Steps:**
1. Stakeholders approve this spec ✓
2. Engineering scopes implementation (1 week)
3. Build MVP service (2 weeks)
4. Beta test with 10 users (1 week)
5. Deploy to production (1 week)

---

## 1. Executive Summary

This document specifies the design and architecture for **Adiyan Registration System**, a WhatsApp-based user registration and onboarding flow that integrates newly registered users into the Adiyan ecosystem for monitoring via approach-road.

**Core Innovation:** Uses **contact name** (not phone number) as the unique user identifier, since WhatsApp doesn't reliably transmit phone numbers. Contact names follow a configurable pattern (e.g., `CLAREO-STUDENT-*`) set by product users.

The system enables:
- **Registration trigger:** Message starting with "Register me in your coaching program"
- **Whitelist-based routing:** Registered contacts route to orchestrator; unregistered to registration service
- **Configurable contact name format** (e.g., `CLAREO-STUDENT-{name}`, `APPROACH-PATIENT-{name}`)
- **Contact name as primary key** in both wa-echo-loop and user-context-service (no phone_number field)
- **Conversational registration flow** collecting name, occupation, LinkedIn profile, and consent
- **Automatic enrollment** into approach-road monitoring upon successful registration
- **Separate microservice architecture** independent from the existing wa-echo-loop adapter
- **Manual user management UI** (control panel) for admin/coach access to add, remove, or update users
- **Configurable eviction policies** for inactive user retention/removal
- **Persistent registration state** to allow abandoned flows to resume

---

## 2. System Goals & Constraints

### Goals
1. Onboard new users into Adiyan via WhatsApp without requiring separate web/app signup
2. Create normalized user profiles with verified identity (phone-based)
3. Automatically integrate registered users into approach-road monitoring workflows
4. Support both self-service (user-initiated) and invitation-based (system-initiated) registration modes
5. Maintain compliance with GDPR and health data privacy regulations
6. Provide coaches/admins with operational control panel for user lifecycle management

### Constraints
1. **Single persona:** All registrations follow one unified flow (no role-based variants initially)
2. **Phone-only verification:** No email, government ID, or secondary identity checks required
3. **WhatsApp-only channel:** No fallback to web, SMS, or phone registration at launch
4. **Separate service:** Registration logic isolated from wa-echo-loop; own queue and context management
5. **No bulk operations:** Individual self-service registration only (no CSV import, no batch invites)
6. **Internal monitoring:** approach-road metrics for coaches/admins only; users don't see progress/badges

---

## 3. Registration Flow Architecture

### 3.1 Registration Initiation & Routing

**Critical Design:** WhatsApp phone numbers are unreliable; **contact name is the unique identifier**.

#### Mode A: User-Initiated (Self-Service)
```
User → Types message starting with "Register me in your coaching program"
  → wa-echo-loop detects registration trigger
  → Extracts: contact_name, whatsapp_message_id
  → Checks whitelist: Is this contact already registered?
     YES → Route to orchestrator (skip registration)
      NO → Route to registration service
  → Registration service processes message
  → On completion: Add contact_name to whitelist
```

**Trigger Text:** Message must start with exactly: `"Register me in your coaching program"`

**Routing Decision Logic:**
```
if (message.startsWith("Register me in your coaching program")) {
  contact_name = whatsapp_message.contact_name  // From WhatsApp
  
  if (whitelist.contains(contact_name)) {
    // Already registered, route to orchestrator
    route_to_queue("orchestrator.requests")
  } else {
    // New registration, route to registration service
    route_to_queue("adiyan.registration.inbound")
  }
}
```

#### Mode B: System-Invited (Coaches set contact names)
```
Coach/Admin → Sets WhatsApp contact name manually in phone
  → Format: Configurable prefix + user identifier (e.g., "CLAREO-STUDENT-Alice")
  → User messages: "Register me in your coaching program"
  → System recognizes contact_name format
  → Registration flow begins
```

**Trigger:** Coach manually manages WhatsApp contacts with standardized naming; user initiates with exact message.

#### Contact Name Format (Configurable)

The contact name serves as a unique, user-friendly identifier. Format is configurable:

**Examples:**
```
CLAREO-STUDENT-Alice
CLAREO-STUDENT-Bob
APPROACH-PATIENT-John
APPROACH-PATIENT-Jane
COACH-TRAINER-Sarah
PROGRAM-PARTICIPANT-123
```

**Configuration:**
```yaml
contact_name_format:
  enabled: true
  prefix: "CLAREO-STUDENT"     # Configurable
  separator: "-"               # Between prefix and user identifier
  pattern_regex: "^CLAREO-STUDENT-.*$"  # Validation pattern
  description: "Contact names must start with CLAREO-STUDENT-"
```

**Enforcement:**
- Coaches/admins are instructed to set WhatsApp contact names in this format
- System validates on registration
- If format doesn't match, reject with guidance: "Contact name must be formatted as CLAREO-STUDENT-{YourName}"

---

### 3.2 Conversational Registration Flow

**Interface:** Conversational with NLP (natural language parsing)

**Key Point:** Contact name is already captured from WhatsApp. Registration flow does NOT collect phone numbers (unreliable via WhatsApp). Instead, focuses on occupational and consent data.

**Flow Steps:**

```
1. REGISTRATION INITIATION
   User: "Register me in your coaching program"
   System: Detects trigger, extracts contact_name from WhatsApp
   System: "Hi [contact_name extracted from WhatsApp]! Let's get you set up for coaching."
   
2. VERIFY CONTACT NAME FORMAT
   System: Validates contact_name against configured pattern (e.g., CLAREO-STUDENT-*)
   If valid:
     System: "Great! I see you as [contact_name]. Proceeding with registration."
   If invalid:
     System: "Your contact name doesn't match our format. Ask your coach to set it as: CLAREO-STUDENT-{YourName}"
     System: Escalate or pause registration

3. OCCUPATION
   System: "What's your current role or occupation? (e.g., Student, Teacher, Healthcare Provider, Coach)"
   User: "Student"
   NLP: Extract and suggest category (with freeform fallback)

4. LINKEDIN PROFILE (OPTIONAL)
   System: "Do you have a LinkedIn profile? (Share the URL if you do, or reply 'Skip' to continue)"
   User: "linkedin.com/in/jane-student" or "Skip"
   System: Validates URL format if provided; stores or leaves blank

5. CONSENT & COMPLIANCE
   System: "To complete registration, please confirm:
   • Your contact information is stored securely per GDPR regulations
   • You'll be monitored via our coaching system (approach-road)
   • We'll send you WhatsApp updates and coaching guidance
   • Your contact name: [extracted contact_name]
   
   Reply 'I agree' to confirm"
   User: "I agree"
   System: Records consent timestamp and version

6. PROFILE CREATION & WHITELIST ADDITION
   System: Creates user profile using contact_name as unique key
   System: Adds contact_name to whitelist (future messages route to orchestrator)
   System: Publishes enrollment event to approach-road

7. REGISTRATION COMPLETE
   System: "Welcome to coaching, [contact_name]! 🎉
   Your profile is live. A coach will reach out shortly.
   You can message us anytime with questions."
```

---

### 3.3 Data Collection Summary

**Collected at Registration:**
- `contact_name` (string, UNIQUE KEY) — extracted from WhatsApp contact, must match configured format (e.g., `CLAREO-STUDENT-Alice`)
- `whatsapp_message_id` (string) — message ID from WhatsApp (for audit trail)
- `occupation` (string or enum) — freeform with category suggestions
- `linkedin_profile_url` (string, optional) — validated URL
- `consent_agreed` (boolean) — explicit GDPR/privacy consent
- `consent_timestamp` (ISO 8601) — when consent was given
- `consent_version` (string) — version of terms accepted
- `registration_source` (enum: "self_initiated" | "coach_invited") — how user was onboarded
- `registration_timestamp` (ISO 8601) — when registration started
- `completion_timestamp` (ISO 8601) — when registration completed
- `approach_road_enrolled` (boolean) — auto-set to true on completion

**Whitelist Entry:**
- `contact_name` (string) — added to routing whitelist upon successful registration
- `registered_timestamp` (ISO 8601) — when contact was added to whitelist
- `status` (enum: "active" | "inactive" | "archived") — for tracking

**NOT Collected (Design Decision):**
- `phone_number` — Unreliable via WhatsApp; contact_name is the definitive identifier
- `age`, `gender`, `health_conditions` — Can be collected later via separate flow
- `subscription_tier` — Defaults to standard; can be upgraded separately

---

### 3.4 State Persistence & Resume Capability

Registration state is persisted in a `registration_sessions` table to enable resumption:

```json
{
  "session_id": "REG_USER_12345_20260806T120000Z",
  "phone_number": "+1-555-123-4567",
  "current_step": "linkedin_profile",
  "steps_completed": ["greeting", "phone_verified", "full_name", "occupation"],
  "data": {
    "full_name": "Jane Doe",
    "occupation": "Healthcare Provider",
    "linkedin_profile_url": null
  },
  "created_at": "2026-08-06T12:00:00Z",
  "last_activity": "2026-08-06T12:05:00Z",
  "status": "in_progress",
  "expires_at": "2026-08-13T12:00:00Z"
}
```

**Resume Rules:**
- Sessions expire after **7 days** of inactivity
- Users can resume by messaging WhatsApp anytime
- System detects incomplete session and offers: "Continue where you left off?" → Resume or start fresh
- On resume, system repeats last step context and allows correction

---

## 4. Validation & Account Activation

### 4.1 OTP Verification (Required)

**OTP Generation & Delivery:**
- 6-digit code, valid for **10 minutes**
- Sent via WhatsApp to the phone number being registered
- Maximum **3 retry attempts** per OTP
- After 3 failures, session escalates to manual review (flag for admin)

**OTP Validation Rules:**
- Must match exactly
- Case-insensitive if alphanumeric
- Single-use (invalidated after successful use)
- Prevents registration bypass

### 4.2 Duplicate Phone Prevention

If a user messages with a phone number already in the system:

```
System: "This phone number is already registered under the account 'Jane Doe'.
Options:
1. Login to your existing account
2. Register with a different phone number
3. Contact support if you need help"
```

**Behavior:**
- No new account created
- Existing user directed to login or contact support
- Session terminated; user must restart or provide different phone

### 4.3 Account Activation

**Automatic activation upon successful registration:**
- All required fields completed
- OTP verified
- Consent agreement recorded
- User profile created in database
- Auto-enrolled into approach-road (see Section 5)

**Status transitions:**
```
pending_otp → otp_verified → pending_profile_data → pending_consent → active
```

---

## 5. Approach-Road Integration

### 5.1 Auto-Enrollment

Upon successful registration completion, the system automatically:

1. **Creates a monitoring entry** in approach-road with:
   - User ID (Adiyan user_id)
   - Phone number
   - Full name
   - Registration timestamp
   - Coach assignment (if available; otherwise queued for assignment)

2. **Triggers initial coaching kickoff** (optional; depends on coaching workflow)
   - Automated onboarding message: "Welcome to Adiyan coaching!"
   - Coach is notified of new user via approach-road dashboard

### 5.2 Monitoring Visibility

**For Coaches/Admins (Internal Use):**
- User profile, registration date, engagement metrics
- Approach-road dashboard shows new users needing assignment
- Can track user activity, messaging patterns, coaching progress

**For End Users:**
- No metrics, badges, or progress bars visible
- Coaches initiate contact; users receive coaching guidance via WhatsApp
- Users don't see approach-road system

---

## 6. Compliance & Consent

### 6.1 GDPR & Data Privacy

**During Registration:**
- Explicit consent required (user types "I agree")
- Consent captures:
  - Data collection and processing
  - WhatsApp communication channel
  - approach-road monitoring activities
  - Data retention policies

**Consent Statement (Template):**
```
To continue, please confirm you understand:
• Your data is stored securely per GDPR regulations and local privacy laws
• You'll receive WhatsApp messages from Adiyan's coaching system
• Your activity will be monitored to provide personalized coaching
• We retain your data for 2 years post-account deactivation
• You can request data deletion anytime

Reply "I agree" to confirm, or "More info" for details.
```

### 6.2 Data Retention

- **Active users:** Data retained indefinitely (user still in system)
- **Inactive users:** Retained per configurable eviction policy (see Section 8.3)
- **Deletion requests:** Honor within 30 days; purge all personal data

---

## 7. Incomplete Registration Handling

### 7.1 Abandoned Flow Recovery

If user stops mid-registration:

**Scenario:**
```
User completes steps 1-3 (greeting, phone verified, full name)
User goes offline for 2 days without completing occupation step
User sends message again
```

**System Response:**
```
System: "Welcome back! I see you started registering on Aug 6.
You had completed:
✓ Phone verified
✓ Full name: Jane Doe

Next step: What's your occupation?
[Or reply 'Start over' to begin fresh]"
```

### 7.2 Session Expiration

- Sessions expire after **7 days** of last activity
- Expired sessions can be restarted (no data loss; just restart from beginning)
- Optional: Send reminder at **3 days** of inactivity: "Complete your Adiyan registration?"

### 7.3 State Cleanup

- Incomplete sessions older than 30 days can be archived (not deleted)
- Can be used for analytics: "X% of users who start registration complete it"

---

## 8. User Lifecycle Management

### 8.1 Control Panel Requirements

**Repurposed Persona Config UI** as the admin/coach control panel:

**Features:**
1. **View all registered users** — filterable table with:
   - Phone number
   - Full name
   - Registration date
   - Occupation
   - approach-road enrollment status
   - Last activity date

2. **Add new user manually** — form to:
   - Enter phone number
   - Enter full name
   - Select occupation
   - Add LinkedIn profile (optional)
   - Manually mark as approached-road enrolled
   - Skip OTP (admin override)

3. **Update user details** — edit:
   - Full name
   - Occupation
   - LinkedIn profile
   - Approach-road enrollment status

4. **Remove user** — archive or delete:
   - Soft delete (flag as inactive)
   - Hard delete (purge all data; only for testing)
   - Requires confirmation

5. **Search & filter** — by:
   - Phone number
   - Name
   - Registration date range
   - Occupation
   - Enrollment status

### 8.2 Inactive User Management

**Default Behavior:**
- Users stay in the system indefinitely after registration
- No automatic removal or degradation

**Configurable Eviction Policy:**
```yaml
user_lifecycle:
  eviction_enabled: false  # Default: disabled
  inactivity_threshold_days: 90  # Days without activity
  eviction_action: "soft_delete"  # soft_delete | archive | notify_coach
  notification_before_eviction_days: 7
```

**Eviction Actions:**
- `soft_delete`: User marked as inactive; data retained; can be reactivated
- `archive`: Moved to archive table; searchable but not active
- `notify_coach`: Coach gets alert; coach decides next step

### 8.3 Re-engagement Strategy

**For Inactive Users (Coach-Managed):**
- Coach reviews inactive users in approach-road
- Coach sends manual WhatsApp nudge or re-engagement message
- No automated system campaigns (configurable in future)

---

## 9. Technical Architecture

### 9.1 Separate Registration Service

**Service Name:** `adiyan-registration-service`

**Responsibilities:**
- Handle WhatsApp registration flows
- Manage registration state
- Generate and verify OTPs
- Create user profiles
- Trigger approach-road enrollment
- Provide control panel backend API

**Independence from wa-echo-loop:**
- Own message queue (separate from wa-echo-loop queue)
- Own user context service (or read-only access to shared context)
- Dedicated database tables for registration data
- Independent deployment and scaling

**Architecture:**
```
WhatsApp Message (incoming)
    ↓
wa-echo-loop (Node.js) routes to registration queue
    ↓
adiyan-registration-service (Go) consumes and processes
    ↓
Stores in user-context-service (existing Go HTTP service)
    ↓
Publishes enrollment event to approach-road queue
```

### 9.2 Technology Stack (Minimal & Pragmatic)

We recommend the **simplest possible stack** that leverages existing Adiyan infrastructure:

#### Language & Runtime
- **Go** (1.21+) for `adiyan-registration-service`
  - Aligns with existing gateway-service, user-context-service
  - Statically compiled, minimal dependencies
  - Easy to containerize and deploy
  - Excellent for microservices

#### Message Queue
- **RabbitMQ** (already running)
  - `adiyan.registration.inbound` — incoming registration messages
  - `adiyan.registration.outbound` — responses to send via WhatsApp
  - `adiyan.approach_road.enrollment` — new user enrollment events
  - Library: `github.com/rabbitmq/amqp091-go` (what you're already using)

#### User Storage
- **HTTP API call to existing user-context-service** (already in Go at port :8001)
  - No new database needed; reuse existing user profile storage
  - Service already handles JSON file I/O at `~/.adiyan/users/{userID}.json`
  - Call `POST /profiles` to create, `GET /profiles/{id}` to read
  - Example: `http://localhost:8001/profiles`

#### OTP Storage & Verification
- **In-memory cache with RabbitMQ fallback** for minimal setup
  - Store OTP + expiry in memory (TTL = 10 minutes)
  - Optional: Persist to user-context-service as metadata
  - Use Go's built-in `sync.Map` for thread-safe storage
  - Library: `github.com/google/uuid` for OTP generation

#### HTTP Server for Control Panel
- **Standard Go `net/http`** (no heavy frameworks)
  - Same pattern as user-context-service
  - Endpoint: `POST /admin/users` (create)
  - Endpoint: `GET /admin/users?phone=+1...` (lookup/filter)
  - Endpoint: `PUT /admin/users/{id}` (update)
  - Endpoint: `DELETE /admin/users/{id}` (remove)
  - Add role-based auth (bearer token or API key)

#### NLP for Intent Detection
- **Start simple: Pattern matching + keywords** (no ML library needed)
  - Example: User says "register" → intent = REGISTER
  - User says "help" → intent = HELP
  - Fallback to menu prompts if intent unclear
  - Later: Integrate with Claude API if needed
  - Alternative: Use open-source `nlp.js` or `compromise` from Node.js layer if complexity grows

#### Configuration & Environment
- **Environment variables + JSON config file**
  ```
  ADIYAN_REG_PORT=8002
  RABBITMQ_URL=amqp://user:pass@localhost:5672/
  UCS_URL=http://localhost:8001  # User-context-service
  OTP_EXPIRY_MINUTES=10
  LOG_LEVEL=info
  ```

#### Persistence
- **No new database** — use existing user-context-service HTTP API
  - Store registration sessions in-memory or in RabbitMQ
  - Optional: Add a `registration_sessions.json` file in same location as user profiles
  - Or persist to RabbitMQ message store (durability built-in)

#### Deployment
- **Single Docker container** for registration service
  - Image: `golang:1.21-alpine` (minimal base)
  - Dependencies: RabbitMQ client library only
  - Network: Communicate with wa-echo-loop, user-context-service, approach-road via Docker network

#### Summary of New Dependencies (Minimal)
```go
github.com/google/uuid         // UUID generation
github.com/rabbitmq/amqp091-go // RabbitMQ client (already used)
```
No additional databases, caches, or frameworks needed.

#### Infrastructure Cost
- **Zero additional infrastructure** — reuses:
  - RabbitMQ (already running)
  - user-context-service (already running)
  - Docker network (already set up)
  - File storage (already at `~/.adiyan/users`)

---

### 9.3 Data Model

**User Profile (Stored via user-context-service HTTP API):**

**KEY CHANGE:** The registration service creates user profiles using **contact_name as the primary key** (not phone_number or email). Contact name is extracted from WhatsApp and must match the configured format.

Example profile structure (stored as JSON at `~/.adiyan/users/{contact_name}.json`):

```json
{
  "contact_name": "CLAREO-STUDENT-Alice",
  "user_id": "USER_12345",
  "whatsapp_message_id": "wamsg_abc123def456",
  "occupation": "Student",
  "linkedin_profile_url": "linkedin.com/in/alice-student",
  "registration_source": "self_initiated",
  "registration_timestamp": "2026-08-06T12:30:00Z",
  "completion_timestamp": "2026-08-06T12:35:00Z",
  "consent_agreed": true,
  "consent_timestamp": "2026-08-06T12:35:00Z",
  "consent_version": "v1.0",
  "approach_road_enrolled": true,
  "whitelist_added_timestamp": "2026-08-06T12:35:00Z",
  "status": "active",
  "last_activity_timestamp": "2026-08-06T12:35:00Z",
  "created_at": "2026-08-06T12:30:00Z",
  "updated_at": "2026-08-06T12:35:00Z"
}
```

**HTTP API Calls (Updated to use contact_name):**
```bash
# Create new user profile (using contact_name as key)
POST http://localhost:8001/profiles
Content-Type: application/json

{
  "contact_name": "CLAREO-STUDENT-Alice",
  "user_id": "USER_12345",
  "occupation": "Student",
  "linkedin_profile_url": "linkedin.com/in/alice-student",
  "consent_agreed": true,
  ...
}

# Get user profile by contact_name
GET http://localhost:8001/profiles/CLAREO-STUDENT-Alice

# List all users by contact name pattern
GET http://localhost:8001/profiles?contact_name_prefix=CLAREO-STUDENT

# Whitelist operations
POST http://localhost:8001/whitelist
{
  "contact_name": "CLAREO-STUDENT-Alice",
  "status": "active"
}

GET http://localhost:8001/whitelist/CLAREO-STUDENT-Alice
```

**Change to user-context-service:**
- Replace `phone_number` field with `contact_name` (unique key)
- Update profile lookup to use contact_name instead of phone
- All messaging uses contact_name as the identifier
- File storage changes from `users/{userID}.json` to `users/{contact_name}.json`

**Registration Sessions (In-Memory or File-based):**

During registration flow, maintain session state. Can be stored:
- Option A: In-memory Go map with TTL (simplest, for single-instance deployments)
- Option B: As JSON files in `~/.adiyan/registrations/{sessionID}.json` (persistent, works with user-context-service model)

Example session file:
```json
{
  "session_id": "REG_USER_12345_20260806T120000Z",
  "phone_number": "+1-555-123-4567",
  "current_step": "linkedin_profile",
  "steps_completed": ["greeting", "phone_verified", "full_name", "occupation"],
  "data": {
    "full_name": "Jane Doe",
    "occupation": "Healthcare Provider",
    "linkedin_profile_url": null
  },
  "created_at": "2026-08-06T12:00:00Z",
  "last_activity": "2026-08-06T12:05:00Z",
  "status": "in_progress",
  "expires_at": "2026-08-13T12:00:00Z"
}
```

**OTP Storage (In-Memory with Optional Persistence):**

OTP attempts tracked in memory (expires after 10 min). Optional: Store in RabbitMQ message for durability.

```go
type OTPAttempt struct {
  ID        string    `json:"id"`
  Phone     string    `json:"phone"`
  OTPCode   string    `json:"otp_code"`      // Hashed
  Verified  bool      `json:"verified"`
  Attempts  int       `json:"attempts"`
  CreatedAt time.Time `json:"created_at"`
  ExpiresAt time.Time `json:"expires_at"`
  SessionID string    `json:"session_id"`
}
```

### 9.3 Integration Points

**WhatsApp Adapter (wa-echo-loop) → Registration Service:**

The existing wa-echo-loop (Node.js, at `approach-road/wa-echo-loop/app.js`) becomes the **routing gateway** using contact name as the key.

**1. Whitelist-Based Routing (Core Change):**

```javascript
// In wa-echo-loop message handler
const contact_name = msg.contact_name;  // Extract from WhatsApp
const message_text = msg.body;

// Check if this contact is on the registration whitelist
const isRegistered = await isOnWhitelist(contact_name);

if (message_text.startsWith("Register me in your coaching program")) {
  // Registration trigger detected
  if (isRegistered) {
    // Contact already registered, route to orchestrator
    console.log(`[wa-echo-loop] ${contact_name} already registered, routing to orchestrator`);
    await rabbitmq.publish('orchestrator.requests', {
      contact_name: contact_name,
      message: message_text,
      message_id: msg.id,
      timestamp: new Date().toISOString()
    });
  } else {
    // New registration, route to registration service
    console.log(`[wa-echo-loop] ${contact_name} new registration, routing to registration service`);
    await rabbitmq.publish('adiyan.registration.inbound', {
      contact_name: contact_name,
      message: message_text,
      message_id: msg.id,
      timestamp: new Date().toISOString()
    });
  }
} else {
  // Non-registration message
  if (isRegistered) {
    // Route to orchestrator (normal coaching flow)
    await rabbitmq.publish('orchestrator.requests', {
      contact_name: contact_name,
      message: message_text,
      message_id: msg.id,
      timestamp: new Date().toISOString()
    });
  } else {
    // Unregistered user sending non-registration message
    // Route to registration service or send guidance
    await rabbitmq.publish('adiyan.registration.inbound', {
      contact_name: contact_name,
      message: message_text,
      message_id: msg.id,
      timestamp: new Date().toISOString(),
      note: "unregistered_user_non_registration_message"
    });
  }
}
```

**2. Whitelist Storage & Management:**

```javascript
// Whitelist can be:
// Option A: In-memory set (for development)
const whitelist = new Set();

// Option B: Persistent (production)
// Store in user-context-service or dedicated whitelist service
async function addToWhitelist(contact_name) {
  await ucsClient.post('/whitelist', {
    contact_name: contact_name,
    added_at: new Date().toISOString(),
    status: 'active'
  });
}

async function isOnWhitelist(contact_name) {
  const result = await ucsClient.get(`/whitelist/${contact_name}`);
  return result.status === 'active';
}
```

**3. Listen for registration responses:**
```javascript
// Subscribe to registration outbound queue
ch.consume('adiyan.registration.outbound', async (msg) => {
  const response = JSON.parse(msg.content);
  // Use contact_name instead of phone
  const { contact_name, message } = response;
  
  // Send response back to user via WhatsApp
  await client.sendMessageByContactName(contact_name, message);
  ch.ack(msg);
});
```

**4. Whitelist Update on Registration Completion:**

Registration service publishes enrollment event that includes:
```json
{
  "event_type": "user_registered",
  "contact_name": "CLAREO-STUDENT-Alice",
  "message": "add_to_whitelist"
}
```

wa-echo-loop consumes this and updates its whitelist:
```javascript
ch.consume('adiyan.approach_road.enrollment', async (msg) => {
  const { contact_name, message } = JSON.parse(msg.content);
  if (message === 'add_to_whitelist') {
    await addToWhitelist(contact_name);
    console.log(`[wa-echo-loop] Added ${contact_name} to whitelist`);
  }
});
```

**Approach-Road (Monitoring System):**
- Listens on `adiyan.approach_road.enrollment` queue
- Creates coaching records for newly registered users
- Assigns coaches (if available)
- Exposes dashboard alerts for new enrollments

**Control Panel (Persona Config UI → Admin Panel):**
- Existing UI at `gateway-service/cmd/user-context-service` gets new routes
- Add HTTP endpoints for user management:
  - `POST /admin/users` — manual user add
  - `GET /admin/users` — list/filter users
  - `PUT /admin/users/{id}` — edit user
  - `DELETE /admin/users/{id}` — remove user
- Calls user-context-service HTTP API under the hood
- Requires role-based auth (bearer token)

### 9.4 Message Queue Architecture (RabbitMQ)

RabbitMQ is already running; registration service reuses it for async communication.

**Queues:**
```
adiyan.registration.inbound       (durable)  ← wa-echo-loop publishes registration intents
adiyan.registration.outbound      (durable)  ← registration service publishes WhatsApp responses
adiyan.approach_road.enrollment   (durable)  ← registration service publishes enrollment events
```

**Message Flow:**

```
1. User sends "register" via WhatsApp
   ↓
   wa-echo-loop (Node.js) receives message
   ↓
   wa-echo-loop publishes to: adiyan.registration.inbound
   
   Message:
   {
     "from_phone": "+1-555-123-4567",
     "message": "register",
     "timestamp": "2026-08-06T12:00:00Z",
     "message_id": "wamsg_12345"
   }

2. adiyan-registration-service (Go) consumes from: adiyan.registration.inbound
   ↓
   Processes registration flow (collects data, verifies OTP, creates profile)
   ↓
   Publishes response to: adiyan.registration.outbound
   
   Message:
   {
     "to_phone": "+1-555-123-4567",
     "message": "What's your full name?",
     "session_id": "REG_USER_12345_20260806T120000Z",
     "timestamp": "2026-08-06T12:01:00Z",
     "correlation_id": "wamsg_12345"
   }

3. wa-echo-loop consumes from: adiyan.registration.outbound
   ↓
   Sends WhatsApp message back to user
   
4. On successful registration:
   adiyan-registration-service publishes to: adiyan.approach_road.enrollment
   
   Message:
   {
     "event_type": "user_registered",
     "user_id": "USER_12345",
     "phone_number": "+1-555-123-4567",
     "full_name": "Jane Doe",
     "occupation": "Healthcare Provider",
     "linkedin_profile_url": "linkedin.com/in/jane-doe",
     "registration_timestamp": "2026-08-06T12:30:00Z",
     "consent_agreed": true,
     "consent_version": "v1.0"
   }
```

**Go Code Example (RabbitMQ Connection):**
```go
import "github.com/rabbitmq/amqp091-go"

conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
ch, err := conn.Channel()

// Declare queues (durable)
ch.QueueDeclare("adiyan.registration.inbound", true, false, false, false, nil)
ch.QueueDeclare("adiyan.registration.outbound", true, false, false, false, nil)
ch.QueueDeclare("adiyan.approach_road.enrollment", true, false, false, false, nil)

// Consume incoming registration messages
msgs, err := ch.Consume("adiyan.registration.inbound", "", false, false, false, false, nil)
for msg := range msgs {
  // Process registration flow
  // Publish responses to outbound queue
  ch.Publish("", "adiyan.registration.outbound", false, false, message)
}
```

---

## 9.5 Minimal Go Service Skeleton (adiyan-registration-service)

To get started, here's the minimal structure needed:

```
gateway-service/cmd/registration-service/
├── main.go                    # Entry point
├── go.mod / go.sum           # Dependencies (minimal: amqp, uuid)
└── internal/registration/
    ├── service.go            # Core logic
    ├── flow.go               # Registration flow steps
    ├── otp.go                # OTP generation/verification
    ├── ucs_client.go         # User-context-service HTTP client
    └── models.go             # Data structures
```

**main.go (40 lines):**
```go
package main

import (
  "log"
  "os"
  "gateway-service/internal/registration"
)

func main() {
  // Get config from env
  rabbitmqURL := os.Getenv("RABBITMQ_URL")
  if rabbitmqURL == "" {
    rabbitmqURL = "amqp://indieclaw:secretpass@localhost:5672/"
  }
  
  ucsURL := os.Getenv("UCS_URL")
  if ucsURL == "" {
    ucsURL = "http://localhost:8001"
  }

  port := os.Getenv("REG_PORT")
  if port == "" {
    port = "8002"
  }

  // Initialize service
  svc, err := registration.NewService(rabbitmqURL, ucsURL)
  if err != nil {
    log.Fatalf("Failed to init service: %v", err)
  }

  log.Printf("[Main] Adiyan Registration Service starting on port %s", port)
  
  // Start message consumer in goroutine
  go svc.StartConsumer()
  
  // Start admin HTTP server
  go svc.StartAdminServer(port)

  // Wait for interrupt
  select {}
}
```

**service.go (100 lines - core):**
```go
package registration

import (
  "encoding/json"
  "net/http"
  "sync"
  amqp "github.com/rabbitmq/amqp091-go"
  "github.com/google/uuid"
)

type Service struct {
  rabbitmqConn *amqp.Connection
  rabbitmqCh   *amqp.Channel
  ucsURL       string
  
  sessions map[string]*Session // In-memory session cache
  mu       sync.RWMutex
}

type Session struct {
  SessionID      string
  PhoneNumber    string
  CurrentStep    string
  StepsCompleted []string
  Data           map[string]interface{}
  CreatedAt      string
  LastActivity   string
  ExpiresAt      string
  Status         string
}

func NewService(rabbitmqURL, ucsURL string) (*Service, error) {
  conn, err := amqp.Dial(rabbitmqURL)
  if err != nil {
    return nil, err
  }

  ch, err := conn.Channel()
  if err != nil {
    return nil, err
  }

  // Declare queues
  ch.QueueDeclare("adiyan.registration.inbound", true, false, false, false, nil)
  ch.QueueDeclare("adiyan.registration.outbound", true, false, false, false, nil)
  ch.QueueDeclare("adiyan.approach_road.enrollment", true, false, false, false, nil)

  return &Service{
    rabbitmqConn: conn,
    rabbitmqCh:   ch,
    ucsURL:       ucsURL,
    sessions:     make(map[string]*Session),
  }, nil
}

func (s *Service) StartConsumer() error {
  msgs, err := s.rabbitmqCh.Consume("adiyan.registration.inbound", "", false, false, false, false, nil)
  if err != nil {
    return err
  }

  for msg := range msgs {
    var incomingMsg map[string]interface{}
    json.Unmarshal(msg.Body, &incomingMsg)
    
    phone := incomingMsg["from_phone"].(string)
    text := incomingMsg["message"].(string)
    
    // Get or create session
    session := s.getOrCreateSession(phone)
    
    // Process message through flow
    response := s.processMessage(session, text)
    
    // Publish response
    s.publishResponse(response)
    
    msg.Ack(false)
  }
  return nil
}

func (s *Service) getOrCreateSession(phone string) *Session {
  s.mu.Lock()
  defer s.mu.Unlock()

  key := "reg_" + phone
  if session, exists := s.sessions[key]; exists {
    return session
  }

  session := &Session{
    SessionID:      uuid.New().String(),
    PhoneNumber:    phone,
    CurrentStep:    "greeting",
    StepsCompleted: []string{},
    Data:           make(map[string]interface{}),
    CreatedAt:      time.Now().Format(time.RFC3339),
    LastActivity:   time.Now().Format(time.RFC3339),
    Status:         "in_progress",
  }
  
  s.sessions[key] = session
  return session
}

func (s *Service) processMessage(session *Session, userInput string) map[string]interface{} {
  // Flow router based on current step
  switch session.CurrentStep {
  case "greeting":
    session.CurrentStep = "phone_verify"
    return map[string]interface{}{
      "to_phone": session.PhoneNumber,
      "message": "Hi! Welcome to Adiyan. Verifying your phone...",
    }
  case "phone_verify":
    // Generate and send OTP
    otp := generateOTP()
    session.Data["otp"] = otp
    session.CurrentStep = "otp_confirm"
    return map[string]interface{}{
      "to_phone": session.PhoneNumber,
      "message": "I've sent you a 6-digit code. What is it?",
    }
  // ... more steps
  default:
    return map[string]interface{}{
      "to_phone": session.PhoneNumber,
      "message": "Something went wrong. Please start over.",
    }
  }
}

func (s *Service) publishResponse(response map[string]interface{}) error {
  body, _ := json.Marshal(response)
  return s.rabbitmqCh.Publish("", "adiyan.registration.outbound", false, false, amqp.Publishing{
    ContentType: "application/json",
    Body:        body,
  })
}
```

**otp.go (20 lines):**
```go
package registration

import (
  "crypto/rand"
  "fmt"
)

func generateOTP() string {
  var otp [6]byte
  rand.Read(otp[:])
  return fmt.Sprintf("%06d", int(otp[0])%1000000)
}

func verifyOTP(storedOTP, userOTP string) bool {
  return storedOTP == userOTP
}
```

**ucs_client.go (30 lines):**
```go
package registration

import (
  "bytes"
  "encoding/json"
  "net/http"
)

func (s *Service) createUserProfile(profile map[string]interface{}) error {
  body, _ := json.Marshal(profile)
  resp, err := http.Post(
    s.ucsURL+"/profiles",
    "application/json",
    bytes.NewBuffer(body),
  )
  if err != nil {
    return err
  }
  defer resp.Body.Close()
  return nil
}

func (s *Service) getUserByPhone(phone string) (map[string]interface{}, error) {
  resp, err := http.Get(s.ucsURL + "/profiles?phone=" + phone)
  if err != nil {
    return nil, err
  }
  defer resp.Body.Close()
  
  var profile map[string]interface{}
  json.NewDecoder(resp.Body).Decode(&profile)
  return profile, nil
}
```

**That's it!** ~200 lines of Go to handle:
- ✅ Message consumption from RabbitMQ
- ✅ Registration flow state management
- ✅ User profile creation via HTTP
- ✅ OTP generation
- ✅ Response publishing
- ✅ Admin HTTP endpoints

---

## 10. Error Handling & Edge Cases

### 10.1 OTP Failures
- **Scenario:** User enters wrong OTP 3 times
- **Action:** Session escalates; flag for manual review; send escalation alert to coach
- **User Message:** "Let me connect you with a coach for verification. Please wait..."

### 10.2 WhatsApp Network Issues
- **Scenario:** Message delivery fails
- **Action:** Retry with exponential backoff (1s, 5s, 30s, 5m)
- **Max Retries:** 5 attempts over 30 minutes
- **Fallback:** Manual coach review if persistent failure

### 10.3 Duplicate Registration Attempts
- **Scenario:** User tries to register same phone twice
- **Action:** Detect in database; reject with guidance to login
- **User Message:** "This phone is already registered. Login here: [link]"

### 10.4 Invalid Data Input
- **Scenario:** User provides malformed phone, name, or LinkedIn URL
- **Action:** NLP parsing with graceful fallback; request correction
- **Strategy:** Validate but don't reject; ask for clarification

### 10.5 Session State Corruption
- **Scenario:** Malformed session JSON or database inconsistency
- **Action:** Log error; start fresh session; notify admin
- **User Experience:** "There was a small hiccup. Let's start fresh."

---

## 11. Security & Privacy

### 11.1 OTP Security
- OTP sent over WhatsApp (encrypted channel)
- OTP never stored in plaintext; hashed in database
- OTP expires after 10 minutes
- Rate limiting: Max 5 OTP requests per phone per hour

### 11.2 Phone Number Verification
- Phone number is the primary identity proof
- E.164 format normalized in database
- No additional verification (government ID, email) required

### 11.3 Data Encryption
- All sensitive fields encrypted at rest (TLS/AES-256)
- WhatsApp channel encrypted end-to-end (WhatsApp's responsibility)
- Database connections use SSL/TLS

### 11.4 Access Control
- Only coaches/admins can access control panel (role-based access)
- Admin audit log: All manual add/update/remove operations logged
- Users cannot access their own registration state via API (only via WhatsApp chat)

---

## 12. Monitoring & Observability

### 12.1 Metrics to Track

**Registration Funnel:**
- Total registration initiations (self-service vs. invited)
- Completion rate (users who finish all steps)
- Drop-off rate (abandoned after X step)
- Time-to-completion (median and p95)

**OTP Metrics:**
- OTP success rate
- OTP retry rate (users retrying)
- OTP failure escalations

**approach-road Enrollment:**
- Users successfully enrolled
- Enrollment delay (time from registration to approach-road)

**User Activity:**
- Daily active users (DAU)
- Messaging patterns (frequency, response time)
- Inactive user count

### 12.2 Alerting

**Alert Conditions:**
- OTP verification success rate < 80% (possible WhatsApp issue)
- Registration completion rate < 50% (flow UX issue)
- approach-road enrollment delay > 5 minutes (integration issue)
- Message delivery failures > 5% (queue or WhatsApp issue)

---

## 11.6 Why This Tech Stack Is Minimal

**Comparison Summary:**

| Option | LOC | Infrastructure | Complexity | Scaling |
|--------|-----|-----------------|------------|---------|
| **Go Service (Recommended)** | ~200 | Zero new | Clean | Independent |
| Extend wa-echo-loop | ~300 | Zero new | Mixed concerns | Scales whole adapter |
| Separate Node.js | ~400 | New runtime | Dual language | New service |

**Winner: Go Service** — Reuses everything you have, adds ~200 lines of battle-tested Go patterns, zero new infrastructure, ships in 1-2 sprints.

---

## 12. Implementation Roadmap (Quick Start)

### Week 1: Setup & Core
- [ ] Create `gateway-service/cmd/registration-service/` directory structure
- [ ] Write `main.go`, `service.go`, `otp.go` (3 files, ~200 lines total)
- [ ] Add RabbitMQ queue declarations (in service)
- [ ] Test message consumer with mock messages

### Week 2: Flow & Integration
- [ ] Implement `flow.go` (registration step handlers)
- [ ] Add `ucs_client.go` (HTTP calls to user-context-service)
- [ ] Integrate with wa-echo-loop for routing (edit wa-echo-loop message handler)
- [ ] Manual testing: Send messages via WhatsApp, verify flow

### Week 3: Admin & Polish
- [ ] Add HTTP admin endpoints to service (`StartAdminServer`)
- [ ] Write simple HTML control panel UI (can reuse existing persona-config UI pattern)
- [ ] Add error handling, logging, metrics
- [ ] Internal smoke testing with team

### Week 4: Beta & Iterate
- [ ] Deploy to staging
- [ ] Beta test with 5-10 users
- [ ] Collect feedback, iterate on NLP and UX
- [ ] Performance & load testing

### Week 5+: GA
- [ ] Deploy to production
- [ ] Monitor metrics (completion rate, OTP success, latency)
- [ ] On-call rotation ready

**Estimated Effort:** 2-3 developers × 4 weeks = 8-12 dev days total

---

## 13. Rollout & Testing Strategy

### 13.1 Phased Rollout

**Phase 1: Beta (Week 1-2)**
- Internal team testing (2-5 beta users)
- Manual registration flow validation
- OTP and approach-road enrollment verification
- Control panel smoke tests

**Phase 2: Pilot (Week 3-4)**
- Invite 20-50 real users
- Monitor funnel, drop-off, completion rates
- Gather user feedback on conversational flow
- Validate NLP intent detection

**Phase 3: General Availability (Week 5+)**
- Open registration to all users
- Monitor metrics and performance
- Scale registration service if needed
- Iterate on UX based on feedback

### 13.2 Testing Checklist

- [ ] OTP generation and validation
- [ ] Phone number deduplication
- [ ] Session persistence and resume
- [ ] Consent capture and storage
- [ ] approach-road enrollment triggered correctly
- [ ] Control panel CRUD operations
- [ ] Eviction policy applied correctly
- [ ] NLP intent detection for key steps
- [ ] WhatsApp message delivery (happy path + error cases)
- [ ] Database consistency under concurrent registrations
- [ ] Load test: 1000+ concurrent registration sessions

---

## 14. Future Enhancements (Not in MVP)

1. **Multi-language support** — Detect user language; respond in same language
2. **Bulk invite system** — Admins can upload CSV to send batch invites
3. **Email verification** — Optional secondary contact for password resets
4. **Subscription tiers** — Different registration paths for free/premium users
5. **Referral codes** — Users can share invite codes for friend recruitment
6. **Auto re-engagement** — Automated nudges for inactive users (configurable)
7. **Identity verification** — Government ID or email verification for certain roles
8. **Social login** — Register via LinkedIn, Google (if needed)
9. **A/B testing** — Test different conversation flows to optimize completion rates
10. **Analytics dashboard** — User-facing progress and engagement dashboard

---

## 15. Dependencies & Prerequisites

**Existing Systems:**
- WhatsApp adapter (wa-echo-loop) — message routing
- approach-road system — user monitoring
- RabbitMQ — message queue
- Database — user profile and session storage

**New Requirements:**
- NLP library/service for intent detection (e.g., spaCy, Dialogflow, or custom)
- OTP generation service (e.g., AWS SNS, Twillio, or in-house)
- Registration service codebase (Go or Node.js, aligns with Adiyan stack)
- Control panel UI modifications (repurpose persona config UI)

---

## 16. Success Criteria

| Metric | Target | Timeline |
|--------|--------|----------|
| Registration completion rate | ≥ 70% | Week 4 (pilot end) |
| OTP verification success rate | ≥ 85% | Week 2 (beta end) |
| approach-road enrollment latency | < 5 min | Week 1 (beta) |
| Time-to-completion (median) | < 5 min | Week 4 (pilot end) |
| Control panel uptime | 99.9% | Ongoing |
| User feedback (NPS) | ≥ 7/10 | Week 4 (pilot end) |

---

## 17. Appendix: Conversation Flow Examples

### Example 1: Successful Self-Service Registration
```
User: "Hi, I want to register"

Adiyan: "Hi! Welcome to Adiyan. I'll help you get started in just a few minutes.
First, what's the best phone number to reach you?"

User: "+1-555-123-4567"

Adiyan: "Great! I'm sending a 6-digit code to that number now. What is it?"

User: [Checks WhatsApp, receives OTP 123456]
        "123456"

Adiyan: "Perfect! Phone verified. Now, what's your full name?"

User: "Sarah Johnson"

Adiyan: "Nice to meet you, Sarah! What's your occupation or current role?"

User: "I'm a health coach"

Adiyan: "Excellent! Do you have a LinkedIn profile? (Share the URL or reply 'Skip')"

User: "linkedin.com/in/sarahjohnson"

Adiyan: "Awesome. Last step, Sarah. To finish, please confirm:
• Your data is secure per GDPR regulations
• You'll receive WhatsApp coaching updates
• Your activity will be monitored for personalized guidance

Reply 'I agree' to confirm"

User: "I agree"

Adiyan: "Welcome to Adiyan, Sarah! 🎉
Your profile is live (ID: USER_99087).
A coach will reach out soon. You can message us anytime."

[System creates user profile, enrolls in approach-road, notifies coaching team]
```

### Example 2: Abandoned & Resumed Registration
```
[Day 1, 2:00 PM]
User: "register me please"

Adiyan: [Registration starts]
        "First, what's your phone number?"

User: "+1-555-987-6543"

Adiyan: "Sending OTP code now..."

User: "654321"

Adiyan: "Verified! What's your full name?"

User: "Mike Chen"

Adiyan: "What's your occupation?"

User: [No response - goes offline]

[Day 3, 4:30 PM]
User: "hello"

Adiyan: "Welcome back, Mike! I see you started registering on Aug 6.
You'd completed:
✓ Phone verified
✓ Full name: Mike Chen

Next up: What's your occupation? (e.g., Coach, Patient, Healthcare Provider)"

User: "I'm a patient"

Adiyan: "Got it. Do you have a LinkedIn profile?"

User: "skip"

Adiyan: "Last step, please confirm:
• Your data is secure per GDPR regulations
• You'll receive WhatsApp updates and coaching guidance
• Your activity will be monitored to provide personalized coaching

Reply 'I agree' to complete"

User: "I agree"

Adiyan: "Welcome to Adiyan, Mike! Your profile is live.
A coach will contact you shortly."
```

### Example 3: Duplicate User Attempt
```
User: "+1-555-123-4567"

Adiyan: "This phone number is already registered under 'Sarah Johnson'.

Options:
1. Login to your existing account
2. Register with a different phone number
3. Contact support if you need help"

User: "1"

Adiyan: "Great! Use this link to access your account: [login link]"
```

---

## 18. Document History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-08-06 | Architecture Team | Initial design specification based on stakeholder requirements |

---

**End of Specification**
