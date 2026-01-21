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


## Dependencies Managed by Docker
- **Node.js** 18+ and npm
- **Go** 1.25+
- **PostgreSQL** 14+


## Getting Started

### 1. Clone the repo

```bash
git clone https://github.com/KathySnider/volunteer-scheduler.git
```

### 2. Create Docker secrets

The server must connect to the database. To do this, it will need the
database url, which must contain the postgres password. The database 
needs the same password when it creates the database and the postgres
user. To avoid having to set (and later update) the password in multiple
places, the server gets the password from one file, gets the url from 
another file and replaces the string `database_password` with the actual 
password.

So, you will need 2 secrets files:
 - secret_postgres_pw.txt contains **only** the password for your database.
 - secret_db_url.txt contains **only** the url:

```bash
<<<<<<< HEAD
postgres://postgres:database_password@db:5432/volunteer-scheduler?sslmode=disable"
=======
postgres://postgres:database_password@db:5432/volunteer-scheduler?sslmode=disable
>>>>>>> main
```

Docker will look for these 2 files (with these names) in the root (volunteer-scheduler)
directory. To change the names or locations, edit the docker-compose.yml files in
volunteer-scheduler and in volunteer-scheduler/database.
Note: that the server **expects** the string `database_password`.

No environment variables are required for local development of the frontend. The frontend 
is configured to connect to `http://localhost:8080/query` by default. For production, you 
may want to set:

```env
PUBLIC_API_URL=https://your-api-domain.com/query
```

### 3. There are 2 options. Before doing either, make sure you are in the proper directory:

```bash
cd volunteer-scheduler
```

### 3 - Option A. Quick Start with Docker-Compose

Option A is The easiest way to run the entire application, especially the first time.

#### 3.A.1. Start all services

```bash
docker-compose up -d
```

#### 3.A.2 Access the application

The web application will be available at `http://localhost:3000`

The GraphQL API will be available to try out at:
 -  **API Endpoint**: `http://localhost:8080/query`
 -  **GraphQL Playground**: `http://localhost:8080/graphql`



### 3 - Option B. Start Each Component
Option B is the way to build and run each component individually, for example 
after you have made some changes to a component.

#### 3.B.1 Set up the database

```bash
cd database
docker-compose up -d
```

Docker will create the DB and its tables. 

Note: The first time you do this, the tables are empty. There is sample data, a
script, and more information in the database folder.


#### 3.B.2 Set up and run the server (vol_sched_api)

```bash
cd vol_sched_api
docker build -t volunteer-api .
docker run -d volunteer-api
```

Docker will install dependencies, generate the volunteer-api with gqlgen, and run the server.


The GraphQL API will be available to try out at:
 -  **API Endpoint**: http://localhost:8080/query
 -  **GraphQL Playground**: http://localhost:8080/graphql

#### 3.B.3 Set up the frontend

```bash
cd vol_sched_app
docker build -t volunteer-frontend .
docker run -d -p 3000:3000 -e PUBLIC_API_URL="http://your-api:8080/query" volunteer-frontend
```

The web application will be available at `http://localhost:3000`



### Stopping Services

```bash
docker-compose down
```

To also remove volumes - which **deletes all data**:
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

(Docker runs `gqlgen generate` when it builds the api.)

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
3. Cretae a new password value.
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
