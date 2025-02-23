# Development stage
FROM node:21-alpine AS builder

# Install dependencies required for building
RUN apk add --no-cache libc6-compat

# Set working directory
WORKDIR /app

# Copy package files first to leverage layer caching
COPY package*.json ./

# Install dependencies
RUN npm ci

# Copy the rest of the application
COPY . .

# Build the application
ENV BACKEND_URL="https://api.iamanshik.online"
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

# Production stage
FROM node:21-alpine AS runner

# Install production dependencies only
RUN apk add --no-cache libc6-compat

WORKDIR /app

# Create non-root user
RUN addgroup -g 1001 nextjs && \
    adduser -S -u 1001 -G nextjs nextjs

# Set environment variables
ENV NODE_ENV=production
ENV BACKEND_URL="https://api.iamanshik.online"
ENV NEXT_TELEMETRY_DISABLED=1

# Copy package files
COPY --from=builder /app/package*.json ./

# Install only production dependencies
RUN npm ci --only=production

# Copy necessary files from builder
COPY --from=builder --chown=nextjs:nextjs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nextjs /app/.next/static ./.next/static
COPY --from=builder --chown=nextjs:nextjs /app/public ./public

# Switch to non-root user
USER nextjs

# Expose the port Next.js runs on
EXPOSE 3000

# Start the application
CMD ["node", "server.js"]