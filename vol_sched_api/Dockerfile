# ============================================
# api/Dockerfile
# Go GraphQL API for Volunteer Scheduler
# ============================================

# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o volunteer-api server.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/volunteer-api .

# Copy schema and config files
COPY --from=builder /app/schema.graphql .
COPY --from=builder /app/gqlgen.yml .

# Expose port
EXPOSE 8080

# Set environment variables (can be overridden)
ENV PORT=8080
ENV DATABASE_URL=postgres://postgres:changeme@db:5432/volunteer_scheduler?sslmode=disable

# Run the application
CMD ["./volunteer-api"]