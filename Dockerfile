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

# Install MySQL client and prerequisites for the PostgreSQL Apt repository.
# Bookworm's default postgresql-client is version 15, which cannot dump PG18.
RUN apt-get update && apt-get install -y --no-install-recommends \
    default-mysql-client \
    ca-certificates \
    curl \
    tzdata \
    && install -d /usr/share/postgresql-common/pgdg \
    && curl --fail --silent --show-error --location \
        https://www.postgresql.org/media/keys/ACCC4CF8.asc \
        -o /usr/share/postgresql-common/pgdg/apt.postgresql.org.asc \
    && printf 'Types: deb\nURIs: https://apt.postgresql.org/pub/repos/apt\nSuites: bookworm-pgdg\nComponents: main\nSigned-By: /usr/share/postgresql-common/pgdg/apt.postgresql.org.asc\n' \
        > /etc/apt/sources.list.d/pgdg.sources \
    && apt-get update && apt-get install -y --no-install-recommends \
        postgresql-client-18 \
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
