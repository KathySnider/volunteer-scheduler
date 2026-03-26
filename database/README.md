# Database Docker Setup
This information is for the DB administrator.

## Quick Start

### Use Docker-Compose (recommended):

Make sure your have a file in volunteer-scheduler called `secret_postgres_pw.txt` that contains your password. Then use docker-compose:

```bash
docker-compose up -d
```



## volunteer-scheduler ERD

```mermaid
erDiagram
    venues {
        int venue_id PK
        varchar venue_name
        varchar street_address
        varchar city
        varchar state
        varchar zip_code
        text timezone
    }

    events {
        int event_id PK
        varchar event_name
        text description
        bool event_is_virtual
        int venue_id FK
    }

    event_dates {
        int event_date_id PK
        int event_id FK
        timestamp start_date_time
        timestamp end_date_time
    }

    service_types {
        int service_type_id PK
        varchar service_type_name
    }

    event_service_types {
        int event_id FK
        int service_type_id FK
    }

    opportunities {
        int opportunity_id PK
        int event_id FK
        varchar job
        varchar other_job_description
        bool opportunity_is_virtual
        text pre_event_instructions
    }

    shifts {
        int shift_id PK
        int opportunity_id FK
        timestamp shift_start
        timestamp shift_end
        int max_volunteers
        int staff_contact_id FK
    }

    volunteers {
        int volunteer_id PK
        varchar first_name
        varchar last_name
        varchar email
        varchar phone
        varchar zip_code
        timestamp created_at
        timestamp last_login_at
    }

    volunteer_shifts {
        int volunteer_id FK
        int shift_id FK
        timestamp assigned_at
    }

    staff {
        int staff_id PK
        varchar first_name
        varchar last_name
        varchar email
    }

    magic_links {
        int id PK
        varchar email
        varchar token
        timestamp created_at
        timestamp expires_at
        timestamp used_at
        varchar ip_address
        varchar user_agent
    }

    sessions {
        int id PK
        varchar email
        varchar token
        timestamp created_at
        timestamp expires_at
        timestamp last_activity_at
        int volunteer_id FK
    }

    venues ||--o{ events : "hosts"
    events ||--o{ event_dates : "has"
    events ||--o{ event_service_types : "categorized by"
    service_types ||--o{ event_service_types : "used in"
    events ||--o{ opportunities : "offers"
    opportunities ||--o{ shifts : "has"
    shifts ||--o{ volunteer_shifts : "filled by"
    volunteers ||--o{ volunteer_shifts : "assigned to"
    staff ||--o{ shifts : "contacts"
    volunteers ||--o{ sessions : "authenticated via"

```


### Loading Sample Data

Sample data is not loaded by default. 

If you suspect you have already loaded the DB with sample data, run the trunc.sql script to
remove all of the the data:
```bash
psql -U postgres -d volunteer-scheduler -p 5433 -a -f .\trunc.sql
```

You will be prompted for the postgres user's password.

To load the data, run the script:

```bash
psql -U postgres -d volunteer-scheduler -p 5433 -a -f .\load-sample-data.sql
```

Again, you will be prompted for the postgres user's password.


## Connecting to the Database
```bash

# Connection string
postgresql://postgres@db:5432/volunteer-scheduler

# Using psql
psql -U postgres -d volunteer-scheduler -p 5433

# Inside of the docker image
docker exec -it volunteer-scheduler psql -U postgres -d volunteer-scheduler

```

## Persistence

Data is stored in a Docker volume named `volunteer-scheduler-data`. To backup:
```bash
docker exec volunteer-scheduler pg_dump -U postgres volunteer-scheduler > backup.sql
```

To restore:
```bash
docker exec -i volunteer-scheduler psql -U postgres volunteer-scheduler < backup.sql
```

## Database Migrations

If you want to do database migrations, the files are in backend/database/migrations. You will 
need to either install golang-migrate, or:

```bash
git clone github.com/golang-migrate/migrate/v4
```
The latter is recommended since it will perform the command and then exit.

### Create a new migration:

```bash
migrate create -ext sql -dir database/migrations -seq migration_name
```

### Apply migrations:

```bash
migrate -database postgresql://postgres@db:5432/volunteer-scheduler -path database/migrations up
```

### Rollback last migration:

```bash
migrate -database postgresql://postgres@db:5432/volunteer-scheduler -path database/migrations down 1
```


## Production Deployment

**IMPORTANT**: Change the default password before deploying!
```bash
docker run -d --name volunteer-scheduler -p 5433:5432 -e POSTGRES_PASSWORD=$(openssl rand -base64 32) -v volunteer-scheduler-data:/var/lib/postgresql/data --restart unless-stopped volunteer-scheduler-db
```

