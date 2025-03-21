# Build stage
FROM golang:1.23.3-alpine AS builder

# Install build dependencies
# RUN apk add --no-cache gcc musl-dev

# Set working directory
WORKDIR /build

# Copy go mod files first to leverage Docker cache
COPY go.mod go.sum ./

# Download dependencies - this layer will be cached if dependencies don't change
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o runnerservice .

# Final stage - creating minimal runtime image
FROM alpine:3.19

# Add ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy only the binary from builder stage
COPY --from=builder /build/runnerservice .

# Run as non-root user for better security
RUN adduser -D appuser
USER appuser

# Set the entrypoint
# CMD ["./runnerservice"] # will handle in deloyment.yaml