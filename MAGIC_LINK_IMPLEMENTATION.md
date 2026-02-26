# Magic Link Authentication Implementation Guide
## System 1 (volunteer-scheduler) - Backend Integration

**Date:** February 25, 2026  
**Purpose:** Enable passwordless email-based authentication in System 1 using magic links, leveraging both Resend.com (production) and Mailhog (development).

---

## Overview

This document describes the implementation of passwordless email authentication (magic links) in the `volunteer-scheduler` (System 1) GraphQL backend. The implementation mirrors the authentication architecture from System 2 (aarp-volunteer-system) but is adapted for Go/GraphQL instead of Next.js/NextAuth.

**Key Features:**
- One-time magic link tokens sent via email
- Automatic transport selection: Resend API (production/real emails) or Mailhog SMTP (development)
- Session management via signed tokens
- Rate limiting (5 requests per hour per email)
- 15-minute token expiry
- Comprehensive test support using Mailhog

---

## Architecture

### Components

#### 1. **Database Schema** (`database/migrations/`)
- `000002_magic_links.up.sql` — Stores magic link tokens
  - Fields: `id`, `email`, `token`, `created_at`, `expires_at`, `used_at`, `ip_address`, `user_agent`
  - Indexes on token lookups for performance
  
- `000003_sessions.up.sql` — Manages user sessions
  - Fields: `id`, `email`, `token`, `created_at`, `expires_at`, `last_activity_at`

#### 2. **Services** (`backend/services/`)

**`mailer.go`** — Email transport abstraction
- `Mailer` struct wraps transport selection
- `ResendTransport` — Uses Resend API (`https://api.resend.com/emails`)
- `MailhogTransport` — Uses local SMTP (`localhost:1025`)
- Transport selection logic:
  - `USE_RESEND=true` → Resend
  - `NODE_ENV=production` → Resend
  - Otherwise → Mailhog

**`auth_magiclink.go`** — Token lifecycle management
- `MagicLinkService` — Main service with methods:
  - `GenerateMagicLink(ctx, email, ip, userAgent)` — Create & store token
  - `SendMagicLinkEmail(ctx, to, token)` — Send email with link
  - `ConsumeMagicLink(ctx, token)` — Validate & mark token as used
  - `CreateSessionToken(email)` — Generate session JWT
  - `ValidateSessionToken(ctx, token)` — Check session validity
  - `CleanupExpiredTokens(ctx)` — Periodic cleanup job

#### 3. **GraphQL Schema** (`backend/graph/volunteer/schema.graphql`)

New mutations added to volunteer schema:
```graphql
Type Mutation {
  requestMagicLink(email: String!): MagicLinkResult!
  consumeMagicLink(token: String!): AuthResult!
}

type MagicLinkResult {
  success: Boolean!
  message: String!
  email: String
}

type AuthResult {
  success: Boolean!
  message: String!
  email: String
  sessionToken: String
}
```

#### 4. **Resolvers** (`backend/graph/volunteer/schema.resolvers.go`)

Implemented mutations:
- `RequestMagicLink` — Calls `MagicLinkService.GenerateMagicLink()` and `SendMagicLinkEmail()`
- `ConsumeMagicLink` — Calls `ConsumeMagicLink()` and `CreateSessionToken()`

#### 5. **Tests** (`e2e/`)

- `e2e/helpers/mailhog.go` — Mailhog API helpers (same pattern as System 2)
  - `SearchEmails(recipient)` — Find emails in Mailhog
  - `WaitForEmail(recipient, timeout)` — Poll for email arrival
  - `ExtractMagicLink(message)` — Parse token from email body
  - `ClearInbox()` — Clean up between tests

- `e2e/magic_link_test.go` — Full flow E2E test
  - `TestMagicLinkFlow` — Complete request → email → consume flow
  - `TestRequestMagicLink` — Isolated mutation test

---

## Environment Configuration

### `.env.example` (Committed)

New file at the project root: `volunteer-scheduler/.env.example`

**Key Variables:**

```bash
# Email Service (Magic Link)
RESEND_API_KEY="re_your_api_key_here"
EMAIL_FROM="noreply@your-verified-domain.com"
USE_RESEND="false"

# Mailhog (Dev/Test)
EMAIL_SERVER_HOST="localhost"
EMAIL_SERVER_PORT="1025"

# Application
NODE_ENV="development"
APP_URL="http://localhost:3000"
SESSION_SECRET="REPLACE_WITH_GENERATED_SECRET_FROM_OPENSSL_RAND_BASE64_32"
SESSION_MAX_AGE="2592000"
```

### `.gitignore` (Already Updated)

Both `volunteer-scheduler/.gitignore` and `backend/.gitignore` already exclude:
- `.env`
- `.env.local`
- `*secret*`

No changes needed.

---

## Setup Instructions

### Development (Using Mailhog)

#### Prerequisites
- PostgreSQL 16 running (or Docker container)
- Go 1.25.2+
- Mailhog running on `localhost:1025` (SMTP) and `localhost:8025` (Web UI)

#### Steps

1. **Copy `.env.example` to `.env`:**
   ```bash
   cp volunteer-scheduler/.env.example volunteer-scheduler/.env
   ```

2. **Update database connection in `.env`:**
   ```bash
   DATABASE_URL="postgresql://volunteer_dev:dev_password@localhost:5432/volunteer_dev"
   ```

3. **Start Mailhog (if not already running):**
   ```bash
   docker run -d -p 1025:1025 -p 8025:8025 mailhog/mailhog:latest
   ```

4. **Run database migrations:**
   ```bash
   # Ensure migrations 000002 and 000003 are applied
   # Using migrate CLI:
   migrate -path ./database/migrations -database "$DATABASE_URL" up
   ```

5. **Start the GraphQL server:**
   ```bash
   cd volunteer-scheduler/backend
   go run ./cmd/server/main.go
   ```

6. **Test the flow:**
   - Open GraphQL playground at `http://localhost:8080`
   - Execute `requestMagicLink` mutation
   - Check Mailhog at `http://localhost:8025`
   - Click magic link or copy token
   - Execute `consumeMagicLink` mutation with token

### Mailhog Web UI

Access the Mailhog inbox at: **http://localhost:8025**

- View all caught emails
- Click an email to see full headers, HTML, and raw MIME
- Use search to find emails by recipient

### Production (Using Resend)

#### Prerequisites
- Resend.com account with verified sending domain
- API key from Resend dashboard

#### Steps

1. **Set environment variables:**
   ```bash
   NODE_ENV="production"
   APP_URL="https://volunteers.aarp-wa.org"  # your production domain
   RESEND_API_KEY="re_xxxxxxxxxxxx"          # your Resend API key
   EMAIL_FROM="noreply@aarp-wa.org"          # must match verified Resend domain
   USE_RESEND="true"
   SESSION_SECRET="<strong-random-secret>"   # openssl rand -base64 32
   SESSION_MAX_AGE="2592000"
   ```

2. **Verify Resend domain:**
   - Log into Resend dashboard
   - Add domain if not present
   - Verify DNS records
   - Confirm domain is verified before deploying

3. **Deploy the application** with environment variables set in your deployment system (Docker, K8s, Lambda, etc.)

4. **Test in production:**
   - Use GraphQL endpoint at your production URL
   - Check email delivery in Resend dashboard

---

## API Usage Examples

### Request Magic Link

**GraphQL Query:**
```graphql
mutation {
  requestMagicLink(email: "user@example.org") {
    success
    message
    email
  }
}
```

**Response (Success):**
```json
{
  "data": {
    "requestMagicLink": {
      "success": true,
      "message": "Magic link sent to your email",
      "email": "user@example.org"
    }
  }
}
```

**Response (Rate Limited):**
```json
{
  "data": {
    "requestMagicLink": {
      "success": false,
      "message": "too many magic link requests; please try again later"
    }
  }
}
```

### Consume Magic Link

**GraphQL Query:**
```graphql
mutation {
  consumeMagicLink(token: "a1b2c3d4e5f6...") {
    success
    message
    email
    sessionToken
  }
}
```

**Response (Success):**
```json
{
  "data": {
    "consumeMagicLink": {
      "success": true,
      "message": "Successfully signed in",
      "email": "user@example.org",
      "sessionToken": "session_abc123..."
    }
  }
}
```

**Response (Invalid Token):**
```json
{
  "data": {
    "consumeMagicLink": {
      "success": false,
      "message": "Invalid or expired magic link"
    }
  }
}
```

---

## Testing

### Running E2E Tests

**Prerequisites:**
- Mailhog running on `localhost:8025`
- GraphQL server running on `localhost:8080`
- `USE_RESEND="false"` in `.env` (forces Mailhog)

**Run tests:**
```bash
cd volunteer-scheduler
go test -v ./e2e
```

**Run specific test:**
```bash
go test -v ./e2e -run TestMagicLinkFlow
```

### Test Coverage

- ✅ `TestMagicLinkFlow` — Full request → email → consume flow
- ✅ `TestRequestMagicLink` — Isolated magic link request
- ✅ Token expiry validation
- ✅ Rate limiting (5 per hour)
- ✅ Email formatting and delivery

---

## Security Considerations

### Token Security
- **Generation:** 32 random bytes (64 hex chars) via `crypto/rand`
- **Storage:** Hashed/indexed in PostgreSQL (indexed for performance)
- **Expiry:** 15 minutes (configurable)
- **Single-use:** Marked as `used_at` after consumption

### Rate Limiting
- **5 magic link requests per email per hour**
- Checked at token generation time
- Returns error if limit exceeded

### Email Security
- **HTTPS enforced in production** (Resend requires it)
- **No sensitive data in email headers** (email address only)
- **Magic link includes token in query parameter** (not header/cookie)

### Session Management
- **Session tokens are random 32-byte strings**
- **Stored with expiry time** (default 30 days)
- **Should be set as HttpOnly cookie** (handled by frontend/middleware)

### Resend API Security
- **API key protection:** Use environment variables, never commit `.env`
- **Rate limit:** Resend's 2 requests/sec (managed in code with logging)
- **Domain verification:** All emails must be from verified Resend domain

---

## Integration with System 2 (aarp-volunteer-system)

### Shared Concepts
- ✅ Magic link token lifecycle (generate → store → validate → consume)
- ✅ Resend + Mailhog transport selection via `USE_RESEND` env var
- ✅ Email templates and subjects
- ✅ Rate limiting per email
- ✅ Session token generation

### Differences
| Aspect | System 1 (Go) | System 2 (Next.js) |
|--------|------|------|
| Framework | Go + gqlgen GraphQL | Next.js + NextAuth |
| Session | Manual JWT/token table | NextAuth server sessions |
| Email Transport | Direct `net/smtp` | Nodemailer wrapper |
| Rate Limiting | Manual query | NextAuth plugin |
| Token Storage | PostgreSQL `magic_links` table | NextAuth tables |

---

## Migration Checklist

Before deploying to production:

- [ ] Run database migrations (000002, 000003)
- [ ] Verify Resend domain is set up and verified
- [ ] Generate strong `SESSION_SECRET` (`openssl rand -base64 32`)
- [ ] Set `NODE_ENV="production"`
- [ ] Set `RESEND_API_KEY` and `EMAIL_FROM` with real values
- [ ] Set `USE_RESEND="true"` (or omit, defaults to Resend in production)
- [ ] Test the flow in staging with real Resend emails
- [ ] Verify email deliverability (check Resend dashboard)
- [ ] Enable rate limiting if desired
- [ ] Set up log monitoring for auth failures
- [ ] Schedule periodic `CleanupExpiredTokens` cleanup job (daily)

---

## Troubleshooting

### Emails Not Arriving

**Development (Mailhog):**
1. Verify Mailhog is running: `curl http://localhost:8025/api/v1/stats`
2. Check GraphQL response for errors in `requestMagicLink`
3. View Mailhog inbox: `http://localhost:8025`
4. Check email logs: `docker logs <mailhog-container>`

**Production (Resend):**
1. Verify `RESEND_API_KEY` is set and valid
2. Verify domain is verified in Resend dashboard
3. Check `EMAIL_FROM` matches verified domain
4. View delivery status in Resend dashboard
5. Check server logs for API errors

### Token Validation Fails

- Ensure `USE_RESEND=false` in `.env` for dev (don't accidentally use Resend)
- Check token format: must be 64 hex characters
- Verify token hasn't expired (15 minutes default)
- Check database: `SELECT * FROM magic_links WHERE token = '...'`

### Rate Limiting Issues

- Default: 5 requests per email per hour
- Check query: `SELECT COUNT(*) FROM magic_links WHERE email = '...' AND created_at > NOW() - INTERVAL '1 hour' AND used_at IS NULL`
- Adjust in `auth_magiclink.go` if needed

---

## Future Enhancements

1. **Email Template Customization** — Allow custom HTML/text via config
2. **Webhook Integration** — Resend webhooks for bounce/complaint tracking
3. **Audit Logging** — Track login attempts, failed auths, IP patterns
4. **2FA Support** — Add optional TOTP or SMS confirmation
5. **OAuth Integration** — Add Google/Microsoft OAuth alternatives
6. **Batch Cleanup Job** — Automatic daily cleanup of expired tokens
7. **Analytics** — Track magic link request/consumption metrics

---

## References

- **Resend Docs:** https://resend.com/docs
- **Mailhog GitHub:** https://github.com/mailhog/MailHog
- **Go PostgreSQL Driver:** https://github.com/lib/pq
- **gqlgen:** https://gqlgen.com

---

**Created:** February 25, 2026  
**Last Updated:** February 25, 2026
