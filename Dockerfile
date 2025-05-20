FROM golang:1.24-alpine AS builder

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

# Install PostgreSQL client and other tools
RUN apk add --no-cache postgresql-client ca-certificates tzdata curl

# Install official MySQL client (not MariaDB's client)
RUN apk add --no-cache --virtual .build-deps \
    bash \
    openssl \
    && mkdir -p /tmp/mysql && cd /tmp/mysql \
    && curl -sSL https://dev.mysql.com/get/Downloads/MySQL-8.0/mysql-community-client-8.0.36-linux-glibc2.28-x86_64.tar.xz -o mysql.tar.xz \
    && tar -xf mysql.tar.xz \
    && cp mysql-community-client-8.0.36-linux-glibc2.28-x86_64/bin/mysqldump /usr/local/bin/ \
    && chmod +x /usr/local/bin/mysqldump \
    && cd / && rm -rf /tmp/mysql \
    && apk add --no-cache libaio ncurses-libs \
    && apk del .build-deps

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
