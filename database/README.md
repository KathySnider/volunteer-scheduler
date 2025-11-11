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
docker build -t volunteer-scheduler-db .
docker run -d \
  --name volunteer-db \
  -p 5433:5432 \
  -e POSTGRES_PASSWORD=your_secure_password \
  -v volunteer-db-data:/var/lib/postgresql/data \
  volunteer-scheduler-db
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
3. Rebuild the image: `docker build -t volunteer-scheduler-db .`
4. Run the container

## Connecting to the Database
```bash
# Connection string
postgresql://postgres:your_password@localhost:5432/volunteer_scheduler

# Using psql
docker exec -it volunteer-db psql -U postgres -d volunteer_scheduler

# From your application
DATABASE_URL=postgresql://postgres:your_password@localhost:5432/volunteer_scheduler
```

## Persistence

Data is stored in a Docker volume named `volunteer-db-data`. To backup:
```bash
docker exec volunteer-db pg_dump -U postgres volunteer_scheduler > backup.sql
```

To restore:
```bash
docker exec -i volunteer-db psql -U postgres volunteer_scheduler < backup.sql
```

## Production Deployment

**IMPORTANT**: Change the default password before deploying!
```bash
docker run -d \
  --name volunteer-db \
  -p 5433:5432 \
  -e POSTGRES_PASSWORD=$(openssl rand -base64 32) \
  -v volunteer-db-data:/var/lib/postgresql/data \
  --restart unless-stopped \
  volunteer-scheduler-db
```
