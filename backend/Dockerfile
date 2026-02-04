# ============================================
# backend/Dockerfile
# Go/GraphQL backend for Volunteer Scheduler
# ============================================

# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies and gqlgen
RUN apk add --no-cache git
RUN go install github.com/99designs/gqlgen@latest

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy all source code
COPY . .

# Generate GraphQL code
RUN gqlgen generate --config gqlgen-volunteer.yml
#RUN gqlgen generate --config gqlgen-admin.yml

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o volunteer-api ./cmd/server

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/volunteer-api .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./volunteer-api"]