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
  -p 5432:5432 \
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

Sample data is **disabled by default**. To enable:

1. Edit `load-sample-data.sql`
2. Remove the `/*` and `*/` around the block of `\copy` commands
3. Rebuild the image: `docker build -t volunteer_scheduler_db .`
4. Run the container

## Connecting to the Database
```bash

# Connection string
postgresql://postgres:your_password@localhost:5432/volunteer_scheduler

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
