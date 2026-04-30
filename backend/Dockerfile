# ============================================
# backend/Dockerfile
# Go/GraphQL backend for Volunteer Scheduler
# ============================================

# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies and gqlgen
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy all source code
COPY . .

# Generate GraphQL code
RUN go run github.com/99designs/gqlgen generate --config gqlgen-volunteer.yml
RUN go run github.com/99designs/gqlgen generate --config gqlgen-admin.yml
RUN go run github.com/99designs/gqlgen generate --config gqlgen-auth.yml

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o volunteer-api ./cmd/server

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS and tzdata for time.LoadLocation
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/volunteer-api .

# and the DB migrations
COPY --from=builder /app/migrations/ /app/migrations/

# Expose port
EXPOSE 8080

# The following vars are hardcoded here for the railway environment only, and
# should be removed if you are going to deploy elsewhere. Railway is *very*
# stubborn about editing its env vars and we needed to override its values
# here. 
# The values in the "secrets" files (see docker-compose.yml at the project's
# root) override these values when building locally (for development).

# Frontend URL for CORS — update if the frontend domain changes
ENV ALLOWED_ORIGIN=https://feisty-enchantment-production-878e.up.railway.app

# Frontend URL for magic link emails — must match the frontend's public domain
ENV FRONTEND_BASE_URL=https://feisty-enchantment-production-878e.up.railway.app

# Sender address for magic link emails
ENV MAIL_FROM="Volunteer Scheduler <noreply@volunteer-scheduler.org>"

# Reply-To address — volunteers who reply to emails will reach this inbox
ENV MAIL_REPLY_TO=adminvolsched@gmail.com

# Run the application
CMD ["./volunteer-api"]