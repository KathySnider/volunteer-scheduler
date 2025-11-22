# Database Docker Setup
# This is for the DB administrator.

## Quick Start

### Use Docker Compose (recommended):
```bash
docker-compose up -d
```

### OR build and run the database:
```bash
cd database
docker build -t volunteer_scheduler_db .
docker run -d --name volunteer_scheduler -p 5432:5433 -e POSTGRES_PASSWORD=your_secure_password -v volunteer_scheduler_data:/var/lib/postgresql/data volunteer_scheduler_db
```

## volunteer_scheduler ERD

```mermaid
erDiagram
    LOCATIONS ||--o{ EVENTS : hosts
    EVENTS ||--o{ EVENT_DATES : has
    EVENTS ||--o{ OPPORTUNITIES : offers
    EVENTS ||--o{ EVENT_ATTENDEES : has
    
    VOLUNTEERS ||--o{ VOLUNTEER_QUALIFICATIONS : has
    VOLUNTEERS ||--o| VOLUNTEER_PREFERENCES : has
    VOLUNTEERS ||--o{ VOLUNTEER_SHIFTS : assigned_to
    
    OPPORTUNITIES ||--o{ OPPORTUNITY_REQUIREMENTS : requires
    OPPORTUNITIES ||--o{ SHIFTS : divided_into
    
    STAFF ||--o{ SHIFTS : leads
    
    SHIFTS ||--o{ VOLUNTEER_SHIFTS : includes
    
    LOCATIONS {
        int location_id
        text location_name
        text street_address
        text city
        text state
        varchar zip_code
    }
    
    EVENTS {
        int event_id
        text event_name
        text description
        boolean event_is_virtual
        int location_id
    }
    
    EVENT_DATES {
        int event_date_id
        int event_id
        date event_date
        time start_time
        time end_time
    }
    
    VOLUNTEERS {
        int volunteer_id
        text first_name
        text last_name
        text email
        varchar phone
        varchar zip_code
        timestamp created_at
    }
    
    VOLUNTEER_QUALIFICATIONS {
        int volunteer_id
        enum qualification
        text other_description
        date acquired_date
    }
    
    VOLUNTEER_PREFERENCES {
        int volunteer_id
        array preferred_roles
        int max_distance_miles
        text availability_notes
    }
    
    STAFF {
        int staff_id
        text first_name
        text last_name
        text email
        varchar phone
        text position
    }
    
    OPPORTUNITIES {
        int opportunity_id
        int event_id
        enum role
        text other_role_description
        boolean opportunity_is_virtual
        text pre_event_instructions
    }
    
    OPPORTUNITY_REQUIREMENTS {
        int opportunity_id
        enum required_qualification
    }
    
    SHIFTS {
        int shift_id
        int opportunity_id
        timestamp shift_start
        timestamp shift_end
        int staff_lead_id
        int max_volunteers
    }
    
    VOLUNTEER_SHIFTS {
        int volunteer_id
        int shift_id
        timestamp assigned_at
        text status
        text notes
    }
    
    EVENT_ATTENDEES {
        int attendee_id
        int event_id
        text first_name
        text last_name
        text email
        timestamp registered_at
    }
```
## Configuration

### Environment Variables

- `POSTGRES_DB`: Database name (default: `volunteer_scheduler`)
- `POSTGRES_USER`: Database user (default: `postgres`)
- `POSTGRES_PASSWORD`: Database password (default: `changeme` - CHANGE THIS!)

### Loading Sample Data

Sample data is not loaded by default. To load the supplied data into your database:

1. Edit `load-sample-data.sql`. WARNING: If there is data in the database it will be lost due to the TRUNCATE commands at the start of the script.
2. In a Windows Powershell, run the command 
`psql  -U postgres -d volunteer_scheduler -p 5433 -a -f .\load-sample-data.sql`


## Connecting to the Database
```bash

# Connection string
postgresql://postgres:your_password@localhost:5433/volunteer_scheduler

# Using psql
docker exec -it volunteer_scheduler psql -U postgres -d volunteer_scheduler

# From your application
DATABASE_URL=postgresql://postgres:your_password@localhost:5432:5433/volunteer_scheduler
```

## Persistence

Data is stored in a Docker volume named `volunteer_scheduler_data`. To backup:
```bash
docker exec volunteer_scheduler pg_dump -U postgres volunteer_scheduler > backup.sql
```

To restore:
```bash
docker exec -i volunteer_scheduler psql -U postgres volunteer_scheduler < backup.sql
```

## Database Migrations

### Create a new migration:

```bash
migrate create -ext sql -dir database/migrations -seq migration_name
```

### Apply migrations:

```bash
migrate -database $DATABASE_URL -path database/migrations up
```

### Rollback last migration:

```bash
migrate -database $DATABASE_URL -path database/migrations down 1
```



## Production Deployment

**IMPORTANT**: Change the default password before deploying!
```bash
docker run -d --name volunteer_scheduler -p 5432:5433 -e POSTGRES_PASSWORD=$(openssl rand -base64 32) -v volunteer_scheduler_data:/var/lib/postgresql/data --restart unless-stopped volunteer_scheduler_db
```

