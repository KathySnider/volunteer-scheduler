# ============================================
# vol_sched_app/Dockerfile
# Next.js Frontend for Volunteer Scheduler
# ============================================

# Build stage
FROM node:20-alpine AS builder

# Set working directory
WORKDIR /app

# Increase npm timeout and add retry settings
RUN npm config set fetch-retry-maxtimeout 120000
RUN npm config set fetch-retry-mintimeout 15000
RUN npm config set fetch-retries 5

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci

# Copy source code
COPY . .

# Build the Next.js application
RUN npm run build

# Production stage
FROM node:20-alpine

# Set working directory
WORKDIR /app

# Set to production environment
ENV NODE_ENV=production

# Copy package files
COPY package*.json ./

# Install only production dependencies
RUN npm ci --omit=dev

# Copy built application from builder
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public
COPY --from=builder /app/next.config.mjs ./
COPY --from=builder /app/src ./src

# Expose port
EXPOSE 3000

# Set environment variable for API URL (can be overridden)
ENV PUBLIC_API_URL=http://localhost:8080/query

# Start the application
CMD ["npm", "start"]