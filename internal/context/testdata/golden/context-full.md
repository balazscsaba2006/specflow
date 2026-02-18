# Execution Context: JWT Authentication Middleware

**Story:** jwt-middleware | **Status:** planned | **Priority:** critical
**Epic:** auth-system

---

## Layer 1: Project Conventions

### CLAUDE.md
# Project Conventions

- Use Go 1.24
- Follow standard Go conventions


---

## Layer 2: Epic & Initiative Context

### Initiative: Platform MVP
**Goal:** Ship the initial platform version with auth and core API

**Success Criteria:**
- All auth stories done
- API documented

### Epic: Authentication System
**Status:** active



#### Phases

- **Phase 1:** api-key-store jwt-middleware 

### Completed Stories
- [done] **api-key-store**: API Key Storage

### Decisions
- **Use JWT for API Authentication** (2025-01-10, accepted)

---

## Layer 3: Spec Requirements


### Acceptance Criteria
- [ ] Validate JWT tokens on protected endpoints
- [ ] Return 401 for invalid tokens

### Doc: Authentication PRD (prd)
## Problem

We need authentication for the API.

## Requirements

- JWT-based auth
- API key support

---

## Layer 4: Implementation Plan


**Status:** approved
## Steps

1. Create `internal/middleware/jwt.go`
2. Add token validation logic
3. Write tests in `internal/middleware/jwt_test.go`

---

## Layer 5: Referenced Files

### internal/middleware/jwt.go
*File does not exist yet (planned but not created).*

### internal/middleware/jwt_test.go
*File does not exist yet (planned but not created).*

---

## Layer 6: Open Items

### Open Questions
- [initiative:platform-mvp] Which OAuth provider to use?
- [epic:auth-system] Should we support refresh tokens in v1?
- [story:jwt-middleware] Which JWT library to use?

### Assumptions (from completed stories)
- [api-key-store] Keys are stored hashed, not plaintext

