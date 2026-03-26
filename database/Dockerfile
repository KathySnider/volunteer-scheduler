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

# Expose PostgreSQL port
EXPOSE 5432/tcp
EXPOSE 5433

# Data will be persisted in a volume at /var/lib/postgresql/data
