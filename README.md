# Volunteer Scheduler

A full-stack web prototype for managing organizational events, volunteer opportunities, and shift assignments. Built with Next.js (React) frontend and Go GraphQL backend.

## Overview

This application will allow organizations to:
- Create and manage events with which volunteers can help.
- Define volunteer opportunities with specific roles (and required qualifications, if any).
- Schedule shifts for each opportunity.
- Allow volunteers to signup for opportunities and choose their shifts.
- Track volunteer assignments.

This is a prototype to see how the volunteers like the signup features. Currently, data  (events, volunteer information, etc.) have to be entered thru psql. 
Administration features coming soon.

### Events Listing Page
- Filter events by:
  - City (multi-select)
  - Event type (Virtual, In-Person, Hybrid)
  - Volunteer roles needed
  - Date range (default is today forward)
- View filtered events with key information.
- Navigate to an event's details using the "More Info" button.

### Event Detail Page
- View complete event information.
- See all volunteer opportunities and shifts for the event.
- View currently assigned volunteers.
- Track shift capacity (assigned vs. max volunteers).
- Allow volunteers to assign themselves to shifts.


## Architecture

### Frontend (in `vol_sched_app/`)
- **Framework**: Next.js
- **Styling**: Tailwind CSS
- **Icons**: Lucide React
- **API Client**: GraphQL queries to backend

### Backend (in `vol_sched_api/`)
- **Language**: Go
- **API**: GraphQL (using gqlgen)
- **Database**: PostgreSQL
- **ORM**: Standard library `database/sql`

### Database (in 'database/')
- **Type**: PostgreSQL
- **Migrations**: golang-migrate
- **Schema**: Fully normalized (3NF)

## Prerequisites
- **Git**
- **Docker**

## Software Installed by Docker
- **Node.js** 18+ and npm
- **Go** 1.25+
- **PostgreSQL** 14+
- **golang-migrate** (for database migrations)


## Getting Started

### 1. Clone the repo

```bash
git clone https://github.com/KathySnider/volunteer-scheduler.git
```

### 2. Set environment variables

#### Backend (vol_sched_api)

Set the environment variables in your shell (which will allow the varialbes to last only for the current session):

```bash
$env:DATABASE_URL="postgres://postgres:YOUR_PASSWORD@localhost:5433:5432/volunteer-scheduler?sslmode=disable"
$env:PORT=8080
```
**OR** set the Environment Variables for your user (in System Properties).

Either way, be sure to change YOUR_PASSWORD to the password you want to use.

#### Frontend (vol_sched_app)

No environment variables are required for local development. The app is configured to connect to `http://localhost:8080/query` by default.

For production, you may want to set:

```env
PUBLIC_API_URL=https://your-api-domain.com/query
```

### 3. There are 2 options. Before doing either, make sure you are in the proper directory:

```bash
cd volunteer-scheduler
```

### 3 - Option A. Quick Start with Docker-Compose

The easiest way to run the entire application, especially the first time.

#### 3.A.1. Start all services

```bash
docker-compose up -d
```

#### 3.A.2 Access the application
##### Frontend: http://localhost:3000

FYI: The frontend will access the API and the server will access the database, but, if you need to have those endpoints for some reason, you can access them at:
##### API: http://localhost:8080
##### Database: http://localhost:5433


### 3 - Option B. Start Each Component

This is the way to build and run each component individually, for example after you have made some changes to a component.

#### 3.B.1 Set up the database

```bash
cd database
docker-compose up -d
```

Docker will install golang-migrate, and create the DB

Note: There is sample data and more information in the README.md in the database folder.


#### 3.B.2 Set up and run the server (vol_sched_api)

```bash
cd vol_sched_api
docker build -t volunteer-api .
docker run -d -p 8080:8080 -e DATABASE_URL=$DATABASE_URL volunteer-api
```

Docker will install dependencies, generate the volunteer-api with gqlgen, and run the server.
The GraphQL API will be available to try out at:
##### **API Endpoint**: http://localhost:8080/query
##### **GraphQL Playground**: http://localhost:8080/graphql

#### 3.B.3 Set up the frontend

```bash
cd vol_sched_app
docker build -t volunteer-frontend .
docker run -d -p 3000:3000 -e PUBLIC_API_URL="http://your-api:8080/query" volunteer-frontend
```

The web application will be available at http://localhost:3000



### Stopping Services
```bash
docker-compose down
```

#### To also remove volumes - which deletes all data:
```bash 
docker-compose down -v
```

## Features

### GraphQL API
- Query events with flexible filtering.
- Query volunteers by qualifications.
- Mutations for assigning volunteers to shifts.
- Fully typed schema with enums for roles and qualifications.

## Database Schema
The database is normalized to Third Normal Form (3NF) with the following main entities:

- **Events**: An organization's events that will need volunteers, with dates and locations.
- **Locations**: Reusable venue information.
- **Opportunities**: Volunteer roles within events.
- **Shifts**: Time slots for opportunities.
- **Volunteers**: Registered volunteers (with qualifications if appropriate).
- **Staff**: Staff member to contacts for a shift.
- **Assignments**: Junction table linking volunteers to shifts.


## Development and customization

### Regenerate GraphQL Code (Backend)

After customizing `schema.graphql`:

```bash
cd vol_sched_api
gqlgen generate
```

### Frontend Development

The frontend uses:
- **React hooks** for state management.
- **Next.js App Router** for routing.
- **Tailwind CSS** for styling.
- **Client-side rendering** (all components use `'use client'`).

## API Documentation

### GraphQL Playground

Visit http://localhost:8080/graphql when the server is running to access the interactive GraphQL Playground where you can:
- Explore the schema.
- Test queries.

### Example Queries

#### Get all events in a city:

```graphql
query {
  events(filter: { cities: ["Austin"] }) {
    id
    name
    description
    eventType
    location {
      city
      state
    }
  }
}
```

#### Get event details with volunteer opportunities:

```graphql
query {
  event(id: "1") {
    name
    description
    opportunities {
      role
      shifts {
        date
        startTime
        endTime
        assignedVolunteers {
          firstName
          lastName
        }
      }
    }
  }
}
```

#### Assign a volunteer to a shift:

```graphql
mutation {
  assignVolunteerToShift(shiftId: "5", volunteerId: "3") {
    success
    message
  }
}
```

## Testing

### Backend (coming)
```bash
cd vol_sched_api
go test ./...
```

### Frontend (coming)
```bash
cd vol_sched_app
npm test
```

## Deployment

### Backend Deployment

1. Build the Go binary:
```bash
cd vol_sched_api
go build -o volunteer-api server.go
```

2. Run migrations on production database
3. Set `DATABASE_URL` environment variable
4. Run the binary: `./volunteer-api`

### Frontend Deployment

1. Build the Next.js app:
```bash
cd vol_sched_app
npm run build
```

2. Deploy to Vercel, Netlify, or your preferred hosting:
```bash
npm run start
```

Or use the Vercel CLI:
```bash
vercel deploy
```

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
- UI components styled with [Tailwind CSS](https://tailwindcss.com/)
- Icons from [Lucide React](https://lucide.dev/)
- Database migrations managed by [golang-migrate](https://github.com/golang-migrate/migrate)
