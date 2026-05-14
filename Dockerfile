# Stage 1: Build
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application statically
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/api/main.go

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /app

# Add ca-certificates for secure connections (HTTPS/SSL) and tzdata for timezones
RUN apk add --no-cache ca-certificates tzdata

# Copy the binary
COPY --from=builder /app/main .
COPY --from=builder /app/migrations ./migrations

# Create uploads directory
RUN mkdir uploads

# Export port
EXPOSE 8080

# Run
CMD ["./main"]
