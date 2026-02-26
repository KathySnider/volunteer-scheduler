# Developer Guide — Startup and Shutdown

## Prerequisites

- Docker Desktop running
- Node.js installed
- `npm install` completed in `frontend/` directory

## Startup

### Scenario 1: Full Startup (after shutdown or first time)

Use this after `docker compose down`, a reboot, or any time the system is fully stopped.

```powershell
cd c:\Projects\volunteer-scheduler
docker compose up -d
docker stop volunteer-frontend
docker start mailhog
cd frontend
npm run dev
```

### Scenario 2: Switch Email Mode (while system is already running)

The `USE_RESEND` setting in `.env` controls where magic link emails are sent:

| USE_RESEND | Emails go to | When to use |
|------------|-------------|-------------|
| `"false"` | **Mailhog** (localhost:8025) | Safe for testing — no real emails are sent. All emails are captured in the Mailhog inbox. |
| `"true"` | **Real email** via Resend API | Sends to actual email addresses. **Use with extreme caution during testing** — only enter an email address you personally own. Never test with a customer or colleague's email. |

**To switch:** Edit `USE_RESEND` in `.env`, save, then restart the API container:

```powershell
cd c:\Projects\volunteer-scheduler
docker compose up -d --no-deps api
```

If you also changed Go backend code (not just .env values), add `--build`:

```powershell
docker compose up -d --build --no-deps api
```

**Do not** use this after a full shutdown — use Scenario 1 instead. This only restarts the API container; the database and other services must already be running.

No need to restart the frontend dev server — it is unaffected by backend .env changes.

### Scenario 3: Shutdown

```powershell
# Press Ctrl+C in the terminal running npm run dev, then:
cd c:\Projects\volunteer-scheduler
docker compose down
docker stop mailhog
```

## Quick Reference

| Service          | Port | URL                       |
|------------------|------|---------------------------|
| Frontend         | 3000 | http://localhost:3000      |
| GraphQL API      | 8080 | http://localhost:8080      |
| Mailhog UI       | 8025 | http://localhost:8025      |
| Mailhog SMTP     | 1025 | (used internally by API)  |
| PostgreSQL       | 5433 | (used internally by API)  |

## Database Migrations (after fresh Docker volume)

If the database was recreated, re-run migrations and load sample data:

```powershell
cd c:\Projects\volunteer-scheduler

# Copy sample data into the container
docker cp "database\sample-data" volunteer-scheduler:/tmp/sample-data

# Load sample data
docker exec -i volunteer-scheduler psql -U postgres -d "volunteer-scheduler" -f - < "database\load-sample-data.sql"

# If volunteers fail to load, use explicit columns:
docker exec volunteer-scheduler psql -U postgres -d "volunteer-scheduler" -c "\copy volunteers(volunteer_id, first_name, last_name, email, phone, zip_code, created_at) FROM '/tmp/sample-data/04-volunteers.csv' CSV HEADER"

# Run magic link and sessions migrations
docker exec -i volunteer-scheduler psql -U postgres -d "volunteer-scheduler" -f - < "database\migrations\000002_magic_links.up.sql"
docker exec -i volunteer-scheduler psql -U postgres -d "volunteer-scheduler" -f - < "database\migrations\000003_sessions.up.sql"
```
