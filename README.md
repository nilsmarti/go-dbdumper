# Go DB Dumper

A lightweight Go application that creates database dumps and uploads them directly to S3-compatible storage. It supports both MySQL and PostgreSQL databases and can be scheduled using cron expressions.

## Features

- Supports MySQL and PostgreSQL databases
- Direct streaming of database dumps to S3 (no local storage required)
- Configurable backup schedule via cron expressions
- Automatic cleanup of old backups based on retention settings
- Docker support for easy deployment
- Command-line interface for manual backups

## Configuration

The application is configured entirely through environment variables:

### Database Configuration

| Variable | Description | Default |
|----------|-------------|--------|
| `DB_TYPE` | Database type (`mysql` or `postgres`) | `mysql` |
| `DB_HOST` | Database host | *required* |
| `DB_PORT` | Database port | `3306` for MySQL, `5432` for PostgreSQL |
| `DB_NAME` | Database name | *required* |
| `DB_USER` | Database user | *required* |
| `DB_PASSWORD` | Database password | *required* |

### S3 Configuration

| Variable | Description | Default |
|----------|-------------|--------|
| `S3_ENDPOINT` | S3 endpoint (e.g., `s3.amazonaws.com` or `minio:9000`) | *required* |
| `S3_REGION` | S3 region | `us-east-1` |
| `S3_BUCKET` | S3 bucket name | *required* |
| `S3_ACCESS_KEY` | S3 access key | *required* |
| `S3_SECRET_KEY` | S3 secret key | *required* |
| `S3_USE_SSL` | Whether to use SSL for S3 connections | `true` |

### Backup Configuration

| Variable | Description | Default |
|----------|-------------|--------|
| `CRON_EXPRESSION` | Cron expression for backup schedule | `0 0 * * *` (daily at midnight) |
| `KEEP_LAST` | Number of backups to keep | `5` |
| `BACKUP_PREFIX` | Prefix for backup files in S3 | `backup` |

## Usage

### Using Docker

The easiest way to run Go DB Dumper is using Docker:

```bash
docker run -d \
  -e DB_TYPE=mysql \
  -e DB_HOST=your-db-host \
  -e DB_NAME=your-db-name \
  -e DB_USER=your-db-user \
  -e DB_PASSWORD=your-db-password \
  -e S3_ENDPOINT=your-s3-endpoint \
  -e S3_BUCKET=your-bucket \
  -e S3_ACCESS_KEY=your-access-key \
  -e S3_SECRET_KEY=your-secret-key \
  -e CRON_EXPRESSION="0 0 * * *" \
  -e KEEP_LAST=5 \
  nilsmarti/go-dbdumper:latest
```

### Using Docker Compose

A `docker-compose.yml` file is provided for easy setup. You can customize it to fit your needs:

```yaml
version: '3.8'

services:
  dbdumper:
    image: nilsmarti/go-dbdumper:latest
    environment:
      # Database configuration
      - DB_TYPE=mysql # or postgres
      - DB_HOST=db
      - DB_PORT=3306 # or 5432 for postgres
      - DB_NAME=mydb
      - DB_USER=dbuser
      - DB_PASSWORD=dbpassword
      
      # S3 configuration
      - S3_ENDPOINT=minio:9000
      - S3_REGION=us-east-1
      - S3_BUCKET=backups
      - S3_ACCESS_KEY=minioadmin
      - S3_SECRET_KEY=minioadmin
      - S3_USE_SSL=false
      
      # Backup configuration
      - CRON_EXPRESSION=0 0 * * * # Daily at midnight
      - KEEP_LAST=5 # Keep last 5 backups
      - BACKUP_PREFIX=myapp
    restart: unless-stopped
```

Start the service with:

```bash
docker-compose up -d
```

## Commands

The application provides the following commands:

- `run`: Run the backup scheduler (default)
- `backup-now`: Run a backup immediately

Example:

```bash
# Run the scheduler
docker run nilsmarti/go-dbdumper:latest run

# Run a backup immediately
docker run nilsmarti/go-dbdumper:latest backup-now
```

## Building from Source

### Prerequisites

- Go 1.23 or later
- MySQL client (for MySQL backups)
- PostgreSQL client (for PostgreSQL backups)

### Build

```bash
git clone https://github.com/nilsmarti/go-dbdumper.git
cd go-dbdumper
go build -o go-dbdumper
```

## GitHub Actions

This repository includes a GitHub Actions workflow that automatically builds and pushes the Docker image to Docker Hub when changes are pushed to the main branch or when a new tag is created.

To use this workflow, you need to set the following secrets in your GitHub repository:

- `DOCKERHUB_USERNAME`: Your Docker Hub username
- `DOCKERHUB_TOKEN`: Your Docker Hub access token

## License

MIT
