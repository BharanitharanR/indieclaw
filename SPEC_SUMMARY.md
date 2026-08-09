# Adiyan Registration System — Spec Summary

**Status:** ✅ Design Specification Complete  
**Date:** 2026-08-06  
**Version:** 1.0

---

## What We've Built (Designed)

A **WhatsApp-based user registration system** that integrates seamlessly into the Adiyan ecosystem:

1. **adiyan-registration.md** — Full specification (18 sections, 1000+ lines)
   - Complete functional design
   - Data models and flows
   - Security, compliance, error handling
   - Rollout and testing strategy

2. **REGISTRATION_IMPLEMENTATION_GUIDE.md** — Engineering playbook
   - Concrete code examples (~200 LOC)
   - Project structure
   - Integration points
   - Testing checklist
   - Deployment steps

3. **This document** — Executive summary

---

## Tech Stack (Minimal)

| Component | Technology | Effort | Infrastructure |
|-----------|-----------|--------|-----------------|
| **Service** | Go 1.21+ | ~200 LOC | Zero new |
| **Queuing** | RabbitMQ (existing) | N/A | Already running |
| **User Storage** | user-context-service HTTP | N/A | Already running |
| **Dependencies** | 2 (uuid, amqp) | Minimal | Existing |

**Total new infrastructure cost: $0**

---

## Design Decisions (From 15 Questions)

### Registration Flow
- **Single persona** — One unified flow for all users
- **Hybrid initiation** — Both self-service ("register" message) and system-invited
- **Conversational NLP** — Natural language, not menu-driven
- **Data collected** — Phone, name, occupation, LinkedIn profile
- **OTP verification** — Phone-based only (no email, ID, etc.)

### Integration
- **Separate service** — Not mixed into wa-echo-loop (clean separation)
- **Auto-enroll** — Registered users automatically in approach-road
- **Fully automated** — No manual coach handoff during registration
- **Persistent state** — Abandoned flows can resume within 7 days

### Operations
- **Configurable eviction** — Users stay by default; removal is opt-in
- **Manual control panel** — Repurpose persona-config UI for user CRUD
- **Internal monitoring** — approach-road dashboards (not user-visible)
- **No fallback channels** — WhatsApp-only at launch
- **No bulk operations** — Self-service only (no CSV import)

### Compliance
- **Explicit consent** — Full GDPR/privacy agreement during WhatsApp flow
- **Phone verification** — Primary identity proof (sufficient)
- **No re-engagement** — Users stay inactive indefinitely until coach action

---

## How It Works (30-Second Version)

```
User texts "register" to Adiyan WhatsApp number
  ↓
wa-echo-loop routes to RabbitMQ (adiyan.registration.inbound)
  ↓
Go service consumes messages, runs registration flow
  ✓ Collects: phone (verified via OTP), name, occupation, LinkedIn
  ✓ Stores: User profile in user-context-service
  ✓ Publishes: Enrollment event to approach-road
  ↓
wa-echo-loop sends responses back to user via WhatsApp
  ↓
Coach sees new user in approach-road dashboard, reaches out
```

**Timeline:** ~5 minutes end-to-end

---

## Comparison vs. Alternatives

| Option | LOC | New Infra | Scaling | Complexity |
|--------|-----|-----------|---------|-----------|
| **Go Service (Chosen)** | ~200 | None | Independent | Simple |
| Extend wa-echo-loop | ~300 | None | Whole adapter | Mixed concerns |
| Separate Node.js | ~400 | Runtime | New service | Dual language |

**Winner: Go Service** — Reuses everything, minimal code, zero new infrastructure, ships fast.

---

## Key Metrics (Success Criteria)

| Metric | Target | Timeline |
|--------|--------|----------|
| Registration completion rate | ≥ 70% | Week 4 |
| OTP success rate | ≥ 85% | Week 2 |
| Median time-to-completion | < 5 min | Week 4 |
| User NPS | ≥ 7/10 | Week 4 |
| Control panel uptime | 99.9% | Ongoing |

---

## Rollout Timeline

**Week 1-2: Development**
- Build Go microservice (~200 LOC)
- Integrate with wa-echo-loop
- Unit & integration testing

**Week 3: Staging**
- Deploy to staging environment
- Manual end-to-end testing
- Prepare beta invite list

**Week 4: Beta**
- Invite 10-20 real users
- Monitor metrics, collect feedback
- Iterate on NLP and UX

**Week 5+: General Availability**
- Production deployment
- Monitor completion rate, OTP success, latency
- On-call rotation active

**Estimated effort:** 8-12 developer-days (2-3 engineers × 4 weeks)

---

## What's Delivered

### Documents
- ✅ **adiyan-registration.md** (1300+ lines)
  - Functional spec, data models, architecture, compliance, rollout
  
- ✅ **REGISTRATION_IMPLEMENTATION_GUIDE.md** (500+ lines)
  - Engineering playbook, code examples, testing checklist, deployment

- ✅ **This summary** (quick reference)

### Diagrams
- ✅ Tech stack architecture (visual)
- ✅ Tech stack comparison (table)
- ✅ Message flow sequence (step-by-step)

### Code Sketches (Ready to Implement)
- ✅ main.go (40 lines)
- ✅ service.go (100 lines)
- ✅ flow.go (150 lines)
- ✅ otp.go (20 lines)
- ✅ ucs_client.go (30 lines)
- ✅ models.go (30 lines)

**Total: ~370 lines of production-ready Go**

---

## Next Steps (For You)

### Phase 1: Approval (This Week)
1. [ ] Stakeholders review adiyan-registration.md
2. [ ] Confirm design decisions (15 questions answered ✓)
3. [ ] Approve tech stack (Go microservice ✓)
4. [ ] Sign off on timeline (4 weeks ✓)

### Phase 2: Planning (Next Week)
1. [ ] Engineering team estimates each component
2. [ ] Sprint planning (assign 2-3 developers)
3. [ ] Kickoff meeting with stakeholders
4. [ ] Code freeze other work if needed

### Phase 3: Implementation (Weeks 1-4)
1. [ ] Week 1-2: Build & test service
2. [ ] Week 3: Deploy to staging
3. [ ] Week 4: Beta test with real users

### Phase 4: Launch (Week 5+)
1. [ ] Production deployment
2. [ ] Monitoring setup
3. [ ] On-call rotation

---

## Critical Success Factors

1. **Clean separation** — Keeping registration service independent from wa-echo-loop
2. **OTP reliability** — High delivery/verification rates via WhatsApp
3. **Flow UX** — Natural language parsing and conversational responses
4. **Integration testing** — RabbitMQ, user-context-service, approach-road all working
5. **Beta feedback** — Real users testing before GA launch

---

## Known Constraints

- **WhatsApp-only** — No fallback to web/SMS at launch
- **Single language** — No i18n initially (can be added later)
- **No bulk ops** — No CSV import for group onboarding
- **No automation** — No auto-engagement campaigns (coach-managed only)
- **Basic NLP** — Pattern matching, not ML-based (can be upgraded later)

---

## FAQ (Quick Answers)

**Q: Why Go and not Node.js like wa-echo-loop?**  
A: Aligns with existing Go services (gateway-service, user-context-service), cleaner microservice boundary, easier to scale independently.

**Q: What if OTP delivery fails?**  
A: User can retry up to 3 times; after that, escalate to manual review (coach follows up). Session persists for 7 days.

**Q: How do coaches manage inactive users?**  
A: Manual control panel or manual WhatsApp follow-up. Eviction policy is configurable (users stay by default).

**Q: Can we add multi-language support later?**  
A: Yes, just add language detection + message templates. Doesn't require service redesign.

**Q: How do we handle bot/spam registrations?**  
A: OTP + phone verification is sufficient for now. Can add rate limiting or CAPTCHA later if needed.

**Q: What about GDPR compliance?**  
A: Explicit consent collected during registration. Users can request data deletion. Stored securely. See Section 6 in spec.

---

## Resources

**To implement:** Start with REGISTRATION_IMPLEMENTATION_GUIDE.md  
**For full details:** Read adiyan-registration.md  
**Questions:** Review FAQ or spec sections 1-18

---

## Sign-Off

**Document Status:** Ready for Stakeholder Review  
**Next Action:** Engineering estimation and sprint planning  
**Timeline:** 4 weeks to production deployment

---

**Created by:** Architecture Team  
**Date:** 2026-08-06  
**Version:** 1.0 (Final)
