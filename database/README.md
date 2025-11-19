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
docker run -d \
  --name volunteer_scheduler \
  -p 5433:5432 \
  -e POSTGRES_PASSWORD=your_secure_password \
  -v volunteer_scheduler_data:/var/lib/postgresql/data \
  volunteer_scheduler_db
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
DATABASE_URL=postgresql://postgres:your_password@localhost:5432/volunteer_scheduler
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
docker run -d \
  --name volunteer_scheduler \
  -p 5432:5432 \
  -e POSTGRES_PASSWORD=$(openssl rand -base64 32) \
  -v volunteer_scheduler_data:/var/lib/postgresql/data \
  --restart unless-stopped \
  volunteer_scheduler_db
```

