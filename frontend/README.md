# Volunteer Scheduler — Frontend

Next.js (App Router) frontend for the Volunteer Scheduler application.

## Stack

- **Framework**: Next.js (App Router)
- **Components**: React with hooks (`useState`, `useEffect`, `useCallback`, `useRef`)
- **Styling**: CSS Modules (one `.module.css` file per page/component)
- **API**: GraphQL via plain `fetch` — no external GraphQL client library

## Structure

```
src/app/
  page.js                        # Root redirect → /events
  layout.js                      # Root layout
  globals.css                    # CSS custom properties (design tokens)
  lib/
    api.js                       # GraphQL fetch helpers + localStorage auth
  components/
    UserMenu.js / .module.css    # Top-bar user name + admin gear menu
  auth/
    magic-link/page.js           # Magic-link callback (stores token + role)
  login/page.js                  # Login / request magic link
  events/
    page.js / events.module.css  # Events listing with filters
    [id]/
      page.js / event-detail.module.css   # Volunteer event detail + sign-up
  my-shifts/
    page.js / my-shifts.module.css        # Volunteer's own shift history
  admin/
    events/
      page.js / admin-events.module.css   # Manage events list
      new/page.js / add-event.module.css  # Add event form
      [id]/page.js / admin-event-detail.module.css  # Edit event + roster
    venues/
      page.js / admin-venues.module.css   # Manage venues
    volunteers/
      page.js / admin-volunteers.module.css  # Manage volunteers
```

## Auth

Authentication uses magic links (passwordless email). After login:
- A JWT token is stored in `localStorage` as `authToken`.
- The user's role (`VOLUNTEER` or `ADMINISTRATOR`) is stored as `authRole`.
- The display name is stored as `authName`.
- All GraphQL calls include the token in the `Authorization: Bearer` header.
- Admins use the `/graphql/admin` endpoint; volunteers use `/graphql/volunteer`.

## Running with Docker

From the project root:

```bash
docker-compose up --build -d
```

The app is available at http://localhost:3000.

## Running Locally (without Docker)

```bash
npm install
npm run dev
```

Requires `NEXT_PUBLIC_API_URL` (or equivalent) pointing at the running backend.

## Key Patterns

- **Module-level sub-components**: Form field components (`ShiftFormFields`, `VenueFormFields`, etc.) are defined at module scope — never inside a page component — to prevent React from remounting them on every render and stealing input focus.
- **Separate date/time fields**: Date and time are kept as separate state fields so changing one never resets the other.
- **`TimeInput` component**: Free-form time text input that stores raw typed text locally and only normalizes + commits the value on blur, allowing natural typing.
