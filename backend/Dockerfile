# Backend Dockerfile
FROM golang:1.25.4-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Copy go mod files (from project root)
COPY go.mod go.sum ./
RUN go mod download

# Copy backend source code
COPY backend/ ./backend/

# Build the application
WORKDIR /app/backend
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/main ./cmd/server

# Final stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache sqlite-libs ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/main .

# Create data directory for SQLite
RUN mkdir -p /app/data

# Expose port
EXPOSE 8080

# Set timezone
ENV TZ=Asia/Shanghai

# Run the application
CMD ["./main"]
