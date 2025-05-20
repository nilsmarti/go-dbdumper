FROM golang:1.20-alpine AS builder

# Install required system dependencies
RUN apk add --no-cache git

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o go-dbdumper .

# Use a minimal image for the final container
FROM alpine:3.18

# Install MySQL and PostgreSQL client tools
RUN apk add --no-cache mysql-client postgresql-client ca-certificates tzdata

# Create a non-root user
RUN adduser -D -u 1000 appuser

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/go-dbdumper /app/go-dbdumper

# Set the ownership of the application
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Set the entrypoint
ENTRYPOINT ["/app/go-dbdumper"]

# Default command
CMD ["run"]
