# Adiyan Registration Service — Implementation Guide

Quick reference for engineering teams building the registration system.

---

## Stack Decision Summary

✅ **Use Go microservice** (`adiyan-registration-service`)

**Why:**
- Aligns with existing gateway-service, user-context-service (both Go)
- Statically compiled, minimal dependencies
- ~200 LOC to ship
- Zero new infrastructure (reuses RabbitMQ, user-context-service)

**Rejected alternatives:**
- ❌ Extend wa-echo-loop (mixes concerns, harder to scale)
- ❌ Separate Node.js service (adds new runtime, more complex)

---

## Project Structure

```
gateway-service/
├── cmd/
│   ├── orchestrator/            (existing)
│   ├── user-context-service/    (existing)
│   └── registration-service/    (NEW)
│       ├── main.go              (40 lines)
│       └── go.mod
└── internal/
    ├── orchestrator/            (existing)
    ├── usercontext/             (existing)
    └── registration/            (NEW)
        ├── service.go           (100 lines)
        ├── flow.go              (150 lines)
        ├── otp.go               (20 lines)
        ├── ucs_client.go        (30 lines)
        └── models.go            (30 lines)
```

**Total new code: ~370 lines Go**

---

## Dependencies (Minimal)

```go
module gateway-service

go 1.21

require (
  github.com/google/uuid v1.5.0
  github.com/rabbitmq/amqp091-go v1.9.0  // Already used in wa-echo-loop
)
```

**That's it.** No frameworks, no databases, no extra libraries.

---

## Configuration (Environment Variables)

```bash
# RabbitMQ
RABBITMQ_URL=amqp://indieclaw:secretpass@localhost:5672/

# User-context-service
UCS_URL=http://localhost:8001

# Registration service
REG_PORT=8002
LOG_LEVEL=info

# OTP
OTP_EXPIRY_MINUTES=10

# Admin auth (optional, add later)
ADMIN_API_KEY=your-secret-key-here
```

---

## Core Components

### 1. Message Consumer (RabbitMQ → Registration Logic)

**File:** `internal/registration/service.go`

```go
func (s *Service) StartConsumer() error {
  msgs, _ := s.rabbitmqCh.Consume("adiyan.registration.inbound", ...)
  
  for msg := range msgs {
    var incoming map[string]interface{}
    json.Unmarshal(msg.Body, &incoming)
    
    session := s.getOrCreateSession(incoming["from_phone"])
    response := s.processMessage(session, incoming["message"])
    
    s.publishResponse(response)
    msg.Ack(false)
  }
}
```

**Responsibilities:**
- Listen on `adiyan.registration.inbound`
- Route messages to flow engine
- Publish responses to `adiyan.registration.outbound`
- Acknowledge messages when done

### 2. Registration Flow Engine

**File:** `internal/registration/flow.go`

```go
func (s *Service) processMessage(session *Session, userInput string) Response {
  switch session.CurrentStep {
  case "greeting":
    // Send welcome, move to phone verification
    
  case "phone_verify":
    // Generate OTP, send to user
    
  case "otp_confirm":
    // Verify OTP, move to name collection
    
  case "full_name":
    // Parse name, move to occupation
    
  case "occupation":
    // Store occupation, move to LinkedIn
    
  case "linkedin_profile":
    // Optional field, move to consent
    
  case "consent":
    // Verify "I agree", create user profile
    // Publish enrollment event
    
  default:
    return ErrorResponse(...)
  }
}
```

**State Transitions:**
```
greeting → phone_verify → otp_confirm → full_name → occupation → linkedin_profile → consent → active
```

### 3. OTP Manager

**File:** `internal/registration/otp.go`

```go
type OTPAttempt struct {
  SessionID string
  Code      string    // Hashed
  Attempts  int
  ExpiresAt time.Time
}

func (s *Service) GenerateOTP() string {
  // 6-digit random number
  // Hash and store in memory with 10-min TTL
}

func (s *Service) VerifyOTP(sessionID, userCode string) bool {
  // Retrieve OTP from cache
  // Hash user input and compare
  // Increment attempt counter
  // Return true/false (max 3 attempts)
}
```

**Storage:** In-memory map with TTL (can be optimized later with Redis if needed)

### 4. User Profile Creation

**File:** `internal/registration/ucs_client.go`

```go
func (s *Service) CreateUserProfile(session *Session) error {
  profile := map[string]interface{}{
    "user_id": "USER_" + generateID(),
    "phone_number": session.PhoneNumber,
    "full_name": session.Data["full_name"],
    "occupation": session.Data["occupation"],
    "linkedin_profile_url": session.Data["linkedin_url"],
    "registration_timestamp": time.Now().Format(time.RFC3339),
    "consent_agreed": true,
    "status": "active",
  }
  
  resp, err := http.Post(
    s.ucsURL + "/profiles",
    "application/json",
    bytes.NewBuffer(marshalJSON(profile)),
  )
  
  if resp.StatusCode != 200 {
    return fmt.Errorf("failed to create profile")
  }
  
  return nil
}
```

**Pattern:** Simple HTTP POST to user-context-service (already running at localhost:8001)

### 5. Admin HTTP Server

**File:** `internal/registration/server.go` (added to service)

```go
func (s *Service) StartAdminServer(port string) error {
  mux := http.NewServeMux()
  
  // GET /admin/users - List users
  mux.HandleFunc("GET /admin/users", s.handleListUsers)
  
  // POST /admin/users - Create user manually
  mux.HandleFunc("POST /admin/users", s.handleCreateUser)
  
  // PUT /admin/users/{id} - Update user
  mux.HandleFunc("PUT /admin/users/{id}", s.handleUpdateUser)
  
  // DELETE /admin/users/{id} - Remove user
  mux.HandleFunc("DELETE /admin/users/{id}", s.handleDeleteUser)
  
  http.ListenAndServe(":" + port, mux)
}
```

**Security:** Add bearer token validation middleware later

---

## Integration Points

### wa-echo-loop (Node.js)

**Current:** Routes all messages to orchestrator queue
**Change:** Detect registration intent, route to registration queue instead

```javascript
// In wa-echo-loop/app.js message handler
if (msg.body.toLowerCase().includes('register')) {
  await rabbitmq.publish('adiyan.registration.inbound', {
    from_phone: msg.from,
    message: msg.body,
    timestamp: new Date().toISOString(),
  });
  return; // Don't send to orchestrator
}

// Otherwise, route to orchestrator as before
```

Also add consumer for registration responses:
```javascript
ch.consume('adiyan.registration.outbound', async (msg) => {
  const { to_phone, message } = JSON.parse(msg.content);
  await client.sendMessage(to_phone, message);
  ch.ack(msg);
});
```

### approach-road (Monitoring)

**Trigger:** On successful registration completion

```go
func (s *Service) PublishEnrollmentEvent(userID, phone, name string) error {
  event := map[string]interface{}{
    "event_type": "user_registered",
    "user_id": userID,
    "phone_number": phone,
    "full_name": name,
    "timestamp": time.Now().Format(time.RFC3339),
  }
  
  body, _ := json.Marshal(event)
  return s.rabbitmqCh.Publish("", "adiyan.approach_road.enrollment", false, false, amqp.Publishing{
    ContentType: "application/json",
    Body: body,
  })
}
```

### user-context-service (User Storage)

**Pattern:** Simple HTTP client, already demonstrated above

---

## Testing Checklist

### Unit Tests
- [ ] OTP generation and verification
- [ ] Flow step transitions
- [ ] Duplicate phone detection
- [ ] Message parsing and intent detection

### Integration Tests
- [ ] RabbitMQ message consumption
- [ ] HTTP call to user-context-service
- [ ] Enrollment event publishing
- [ ] Session persistence and resume

### Manual Testing
- [ ] Full registration flow via WhatsApp (greeting → completion)
- [ ] Abandoned flow resume
- [ ] Admin panel CRUD (create, read, update, delete users)
- [ ] OTP failures and retries
- [ ] Duplicate phone rejection

### Load Testing
- [ ] 100 concurrent registrations
- [ ] Message queue throughput (msgs/sec)
- [ ] Database write latency

---

## Deployment

### Docker Image

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY . .
RUN go build -o registration-service ./cmd/registration-service

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /build/registration-service /app/
EXPOSE 8002
CMD ["/app/registration-service"]
```

### Environment Setup

```bash
# Create service account in RabbitMQ
rabbitmqctl add_user registration-svc <password>
rabbitmqctl set_permissions -p / registration-svc "adiyan.*" "adiyan.*" "adiyan.*"

# Declare queues (script or manual)
./scripts/setup-rabbitmq-queues.sh
```

### Health Check

```bash
# HTTP endpoint for k8s/Docker health checks
GET /health
Response: {"status": "ok", "version": "1.0"}
```

---

## Monitoring & Observability

### Key Metrics

```go
// Use existing logging pattern from gateway-service
log.Printf("[Registration] User registered: %s", userID)
log.Printf("[OTP] Verification failed for: %s (attempt %d)", phone, attempts)
log.Printf("[Error] Failed to create profile: %v", err)
```

### Logs to Track

- Registration start/completion
- OTP generation and verification
- Profile creation success/failure
- Enrollment event publishing
- Admin operations (user create/update/delete)
- Message queue errors

### Alerts

- OTP success rate < 80%
- Registration completion rate < 50%
- Message queue latency > 5s
- HTTP errors to user-context-service

---

## Rollout Timeline

| Phase | Duration | Users | Checklist |
|-------|----------|-------|-----------|
| **Dev** | Week 1-2 | 0 | Code, unit tests, integration tests |
| **Staging** | Week 3 | 5 | Deploy to staging, manual testing |
| **Beta** | Week 4 | 10-20 | Invite real users, collect feedback |
| **GA** | Week 5+ | All | Production deployment, monitoring |

---

## Common Pitfalls & Solutions

| Issue | Solution |
|-------|----------|
| OTP messages not arriving | Check WhatsApp adapter is consuming outbound queue |
| Duplicate users registering | Verify phone deduplication logic in `getOrCreateSession` |
| User profiles not showing in control panel | Ensure user-context-service is responding to GET /profiles |
| RabbitMQ queue backlog | Scale consumer (add goroutine pool or workers) |
| Session state lost on restart | Persist to file or add Redis (optional) |

---

## Quick Commands

```bash
# Build service
cd gateway-service
go build -o bin/registration-service ./cmd/registration-service

# Run locally
RABBITMQ_URL=amqp://localhost:5672 UCS_URL=http://localhost:8001 ./bin/registration-service

# View RabbitMQ queues
docker exec rabbitmq rabbitmqctl list_queues

# Test message publishing (for manual testing)
python scripts/test-registration.py --phone "+1-555-123-4567" --message "register"
```

---

## Success Criteria (End of Week 4)

- [ ] Registration completion rate ≥ 70%
- [ ] OTP success rate ≥ 85%
- [ ] Time-to-completion (median) < 5 min
- [ ] Zero critical bugs in beta
- [ ] User feedback (NPS) ≥ 7/10
- [ ] Ready for GA deployment

---

## Next Steps

1. **Spec approval** — Stakeholders review adiyan-registration.md ✓
2. **Engineering estimation** — Team estimates ~8 dev days
3. **Sprint planning** — Assign 2-3 engineers for 4 weeks
4. **Kickoff** — Code freeze other work, focus on registration
5. **Weekly demos** — Show progress to stakeholders

---

**Questions?** See adiyan-registration.md for full specification details.
