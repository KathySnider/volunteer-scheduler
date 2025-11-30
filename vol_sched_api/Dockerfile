# ============================================
# vol_sched_api/Dockerfile
# Go GraphQL API for Volunteer Scheduler
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

# Copy source code
COPY . .

# Generate GraphQL code
RUN gqlgen generate

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o volunteer-api .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/volunteer-api .

# Copy schema files
COPY --from=builder /app/schema.graphql .

# Expose port
EXPOSE 8080

# Set environment variables (can be overridden)
#ENV PORT=8080

# Run the application
CMD ["./volunteer-api"]