# ============================================
# database/Dockerfile
# PostgreSQL Database for Volunteer Scheduler
# This Dockerfile should only need to be run
# once (unless DB needs to be reinstalled), 
# and should be run by the DB administrator 
# on the server to which the client code will 
# have access.
# ============================================


FROM postgres:16

# Copy schema initialization script
# PostgreSQL will automatically run files in /docker-entrypoint-initdb.d/
# in alphabetical order when the container is first created
COPY init-db.sql /docker-entrypoint-initdb.d/01-schema.sql

# Copy sample data files (optional - see 02-sample-data.sql to enable)
#COPY sample-data/*.csv /tmp/sample-data/

# Copy sample data loading script 
#COPY load-sample-data.sql /docker-entrypoint-initdb.d/02-sample-data.sql

# Expose PostgreSQL port
EXPOSE 5432/tcp
EXPOSE 5433

# The base postgres image will handle startup
# Data will be persisted in a volume at /var/lib/postgresql/data
