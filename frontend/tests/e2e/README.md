# E2E Tests (Playwright)

End-to-end tests run against the full docker-compose stack.

## Prerequisites

1. Install Playwright browsers (one-time setup):
   ```
   npx playwright install chromium
   ```
2. Docker stack is running:
   ```
   docker compose up -d
   ```
3. At least one `ADMINISTRATOR` account exists in the database.

## Configuration

Set the following environment variable before running tests:

| Variable | Description | Example |
|---|---|---|
| `E2E_ADMIN_EMAIL` | Email of an existing admin account in the test DB | `admin@example.com` |
| `BASE_URL` | Frontend URL (default: `http://localhost:3000`) | `http://localhost:3000` |
| `MAILHOG_URL` | Mailhog API URL (default: `http://localhost:8025`) | `http://localhost:8025` |

You can set these in a `.env.test` file or export them in your shell:
```
export E2E_ADMIN_EMAIL=admin@yourapp.test
```

## Running the tests

```bash
# Run all E2E tests (headless)
npm run test:e2e

# Run with Playwright UI (interactive mode)
npm run test:e2e:ui

# View the last HTML report
npm run test:e2e:report
```

## Test suites

| File | What it tests |
|---|---|
| `auth.spec.ts` | Magic-link login, role routing, unknown email, invalid token |
| `shift-signup.spec.ts` | Volunteer signs up for / cancels a shift, full shift handling |
| `feedback.spec.ts` | Volunteer submits feedback, admin workflow, note visibility |
| `admin-event.spec.ts` | Admin creates events, access control for non-admins |

## Notes

- Tests run serially (`workers: 1`) because they share a live database.
- Each test that needs a volunteer creates a fresh account via the admin API to avoid cross-test contamination.
- Mailhog captures all outbound email — `clearMailbox()` is called before tests that check email to prevent stale messages interfering.
