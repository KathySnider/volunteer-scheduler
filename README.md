# Volunteer Scheduler

A full-stack web application for managing organizational events, volunteer opportunities, and shift assignments. Built with a Next.js (React) frontend and a Go/GraphQL backend.

## Overview

This application allows organizations to:
- Create and manage events with which volunteers can help.
- Define volunteer opportunities with specific jobs.
- Schedule multiple shifts for each opportunity, with maximum slots available.
- Allow volunteers to sign up for opportunities and choose their shifts.
- Track and display volunteer assignments.

This application is still under construction. Currently, volunteers can:
- See events and filter on dates, regions, event format, and job types.
- See available jobs and shifts for an event, grouped by job.
- Sign up for a shift.
- See their own shift assignments on the My Shifts page.
- Cancel a shift.

Admins can:
- Do everything volunteers can do, including sign up for shifts.
- Add/edit/delete events (dates, opportunities, and shifts).
- Add/edit/delete venues.
- Add/edit/delete volunteers and view their shift history.
- View the volunteer roster for each event.

More administration features coming all the time.

#### Events Listing Page
- Filter events by:
  - Region
  - Event format (Virtual, In-Person, Hybrid)
  - Date range (defaults to today forward)
- View filtered events with key information.
- Navigate to an event's details using the "View Details" button.

#### Event Detail Page (Volunteer view)
- View complete event information.
- See all volunteer opportunities grouped by job, with service type badges.
- View shift capacity (spots available / full).
- Sign up for or cancel a shift directly on the page.

#### Admin Event Management Page
- Edit event details, dates, opportunities, and shifts.
- View the volunteer roster (who is signed up for each shift).

#### My Shifts Page
- View upcoming, past, or all signed-up shifts grouped by event.
- Cancel upcoming shifts.

#### Manage Venues Page (Admin)
- Add, edit, and delete venues.

#### Manage Volunteers Page (Admin)
- Search volunteers by name or email.
- Edit volunteer profile and role.
- Delete a volunteer.
- View a volunteer's shift history (upcoming or all).


## Architecture

### Frontend
- **Framework**: Next.js (App Router)
- **Components**: React (hooks, client components)
- **Styling**: CSS Modules
- **API Client**: Plain `fetch` with GraphQL queries — no external GraphQL client library

### Backend
- **Language**: Go
- **API**: GraphQL (using gqlgen), three separate endpoints:
  - `/graphql/auth` — magic-link login
  - `/graphql/volunteer` — volunteer-facing queries and mutations
  - `/graphql/admin` — admin-only queries and mutations
- **Database**: PostgreSQL
- **DB Access**: Standard library `database/sql`

### Database (in `backend/database/`)
- **Type**: PostgreSQL
- **Migrations**: golang-migrate
- **Schema**: Fully normalized (3NF)

## Prerequisites
- **Git**
- **Docker**


## Dependencies Managed by Docker
- **Node.js** 18+ and npm
- **Go** 1.21+
- **PostgreSQL** 14+


## Getting Started

### 1. Clone the repo

```bash
git clone https://github.com/KathySnider/volunteer-scheduler.git
```

### 2. Create Docker secrets

The server must connect to the database. To do this, it will need the
database URL, which must contain the postgres password. The database
needs the same password when it creates the database and the postgres
user. To avoid having to set (and later update) the password in multiple
places, the server gets the password from one file, gets the URL from
another file and replaces the string `database_password` with the actual
password.

Once in production, the application will need a Resend API key.

So, you will need 3 secrets files:
 - `secret_resend_api_key.txt` (for now this can be empty).
 - `secret_postgres_pw.txt` contains **only** the password for your database.
 - `secret_db_url.txt` contains **only** the URL:

```
postgres://postgres:database_password@db:5432/volunteer-scheduler?sslmode=disable
```

Docker will look for these 3 files (with these names) in the root (`volunteer-scheduler`)
directory. To change the names or locations, edit the `docker-compose.yml` files in
`volunteer-scheduler` and in `volunteer-scheduler/database`.
Note: the server **expects** the string `database_password` as a placeholder in the URL file.

There are several environment variables that will be needed. Once you have checked out the files, copy `env.example` to `.env` in `volunteer-scheduler`. Make any local edits needed for development.

In production, you will need to add the `PUBLIC_API_URL`:

```env
PUBLIC_API_URL=https://your-api-domain.com/query
```

### 3. Start the application

Before doing either option below, make sure you are in the root directory:

```bash
cd volunteer-scheduler
```

### Option A. Quick Start with Docker Compose (recommended)

#### Start all services

```bash
docker-compose up -d
```

To rebuild after code changes:

```bash
docker-compose down && docker-compose up --build -d
```

#### Access the application

| Service | URL |
|---|---|
| Web application | http://localhost:3000 |
| Mailhog (email preview) | http://localhost:8025 |
| GraphQL — auth | http://localhost:8080/graphql/auth |
| GraphQL — volunteer | http://localhost:8080/graphql/volunteer |
| GraphQL — admin | http://localhost:8080/graphql/admin |


### Option B. Start Each Component Individually

#### 1. Set up the database

```bash
cd database
docker-compose up -d
```

Docker will create the DB and its tables.

Note: The first time you do this, the tables are empty. There is sample data, a
script, and more information in the `database` folder.

#### 2. Set up and run the server

```bash
cd backend
docker build -t volunteer-api .
docker run -d volunteer-api
```

Docker will install dependencies, run gqlgen code generation, and start the server.

#### 3. Set up the frontend

```bash
cd frontend
docker build -t volunteer-frontend .
docker run -d -p 3000:3000 -e PUBLIC_API_URL="http://your-api:8080" volunteer-frontend
```

The web application will be available at `http://localhost:3000`.


### Stopping Services

From the root directory:

```bash
docker-compose down
```

To also remove volumes — which **deletes all data**:
```bash
docker-compose down -v
```


## Database Schema

The database is normalized to Third Normal Form (3NF) with the following main entities:

- **Events**: Organization events that need volunteers, with dates and locations.
- **Venues**: Reusable venue information (address, timezone).
- **Opportunities**: Volunteer roles within events (job type, instructions).
- **Shifts**: Time slots within an opportunity, with max volunteer capacity.
- **Volunteers**: Registered volunteers. Administrators are also stored here with `role = 'ADMINISTRATOR'`.
- **Staff**: Staff members who serve as shift contacts.
- **Assignments** (`volunteer_shifts`): Junction table linking volunteers to shifts, with sign-up and cancellation timestamps.

See `database/README.md` for the full ERD.


## Development

### Regenerate GraphQL Code (Backend)

After changing any `.graphql` schema file, re-run gqlgen for both endpoints:

```bash
cd backend
go run github.com/99designs/gqlgen generate --config gqlgen-volunteer.yml
go run github.com/99designs/gqlgen generate --config gqlgen-admin.yml
```

Docker runs this automatically during `docker-compose up --build`.

### Frontend Development

The frontend uses:
- **React hooks** for state management.
- **Next.js App Router** for routing (all pages are in `src/app/`).
- **CSS Modules** for scoped styling (`.module.css` files alongside each page).
- **Client-side rendering** (all components use `'use client'`).

> **Important pattern**: Sub-components that contain form inputs (e.g. `ShiftFormFields`,
> `VenueFormFields`) must be defined at **module level**, outside the page component.
> If defined inside the page function, React treats them as new component types on every
> render and unmounts/remounts them on each keystroke, stealing input focus.


## Testing

### Backend
```bash
cd backend
go test ./tests/integration/...
```

### Frontend (E2E — Playwright)

E2E tests run against the full docker-compose stack using Playwright and Chromium.

**One-time setup:**
```bash
cd frontend
npm run test:e2e:install   # installs Chromium
```

**Configuration** — create `frontend/.env` with:
```
E2E_ADMIN_EMAIL=your-admin@example.com
```
Set this to the email of an existing `ADMINISTRATOR` account in the database.

To get an admin account for testing, you have two options:
- **Sample data** (recommended): Load `database/load-sample-data.sql` — it includes an admin account. Update the placeholder email in that file to your own before loading, then use that email here.
- **Manual**: Insert a row directly in PostgreSQL with `role = 'ADMINISTRATOR'`.

**Run all tests:**
```bash
cd frontend
npm run test:e2e
```

**Other commands:**
```bash
npm run test:e2e:ui       # interactive Playwright UI (great for debugging)
npm run test:e2e:report   # open the last HTML report
```

**Test suites:**

| Suite | What it covers |
|---|---|
| `auth.spec.ts` | Magic-link login, role routing, unknown email, invalid token |
| `events.spec.ts` | Events listing page, filtering, event count display |
| `shift-signup.spec.ts` | Sign up for a shift, cancel, full shift, access control |
| `feedback.spec.ts` | Submit feedback, admin Q&A workflow, note visibility |
| `admin-event.spec.ts` | Admin creates/edits/deletes events, form validation, access control |
| `admin-volunteers.spec.ts` | Admin views, edits, and deletes volunteers |
| `profile.spec.ts` | View and edit volunteer profile |

See `frontend/tests/e2e/README.md` for more details.


## Deployment

### Backend
1. Run migrations on the production database.
2. Set the secrets files and `.env` for production.
3. Build and run via Docker.

### Frontend
1. Build the Next.js app:
```bash
cd frontend
npm run build
npm run start
```
Or deploy the Docker container to your preferred host.


## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Commit changes: `git commit -am 'Add feature'`
4. Push to branch: `git push origin feature-name`
5. Submit a pull request


## License

MIT License


## Acknowledgments

- Built with [gqlgen](https://gqlgen.com/) for GraphQL in Go
- Database migrations managed by [golang-migrate](https://github.com/golang-migrate/migrate)
- Email delivery via [Resend](https://resend.com/)
- Email previewing in development via [Mailhog](https://github.com/mailhog/MailHog)
- Developed with the assistance of [Claude Code](https://claude.ai/code) by Anthropic
