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

# Frontend URL for CORS — update if the frontend domain changes
ENV ALLOWED_ORIGIN=https://feisty-enchantment-production-878e.up.railway.app

# Sender address for magic link emails
ENV MAIL_FROM="Volunteer Scheduler <onboarding@resend.dev>"

# Run the application
CMD ["./volunteer-api"]