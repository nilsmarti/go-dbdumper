FROM golang:1.24 AS builder

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

# Use Debian-based image for the final container
FROM debian:bookworm-slim

# Install MySQL 8 client, PostgreSQL client, and other tools
RUN apt-get update && apt-get install -y \
    default-mysql-client \
    postgresql-client \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

# Create a non-root user
RUN useradd -m -u 1000 appuser

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
