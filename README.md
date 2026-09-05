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
| `BACKUP_TIMEOUT` | Timeout for a single backup or restore operation (Go duration format: `30m`, `1h`, `45m30s`) | `30m` |
| `COMPRESSION_ENABLED` | Gzip-compress backups before encryption/upload | `true` |
| `CLEANUP_ENABLED` | Delete old backups beyond `KEEP_LAST` from the app. Set to `false` when using Object Lock (retention enforced server-side). | `true` |
| `OBJECT_LOCK_MODE` | S3 Object Lock retention mode: `GOVERNANCE` or `COMPLIANCE`. Requires Object Lock enabled on the bucket. | *disabled* |
| `OBJECT_LOCK_RETAIN_UNTIL_DAYS` | Days to retain each object under Object Lock (from upload time). Required if `OBJECT_LOCK_MODE` is set. | *required if Object Lock enabled* |

### Encryption Configuration

Backups can be asymmetrically encrypted client-side using [age](https://age-encryption.org) before they reach S3. The app server holds only the **public key** (age recipient) and can therefore encrypt backups without ever being able to decrypt them. Decryption requires the corresponding **private key** (age identity), which should live on a separate, hardened restore host — never on the app server.

| Variable | Description | Default |
|----------|-------------|--------|
| `ENCRYPTION_ENABLED` | Enable client-side encryption of backups | `false` |
| `ENCRYPTION_PUBLIC_KEY` | Age recipient key (`age1...`), mutually exclusive with `ENCRYPTION_PUBLIC_KEY_FILE` | *required if enabled* |
| `ENCRYPTION_PUBLIC_KEY_FILE` | Path to a file containing one or more age recipients (one per line, `#` comments allowed) | *required if enabled* |

### Backup File Naming

The object suffix reflects the transformations applied, so the restore command can automatically detect how to process each backup:

| Suffix | Description |
|--------|-------------|
| `.sql` | Plain SQL dump (legacy, no compression or encryption) |
| `.sql.gz` | Gzip-compressed SQL dump |
| `.sql.age` | Encrypted SQL dump (no compression) |
| `.sql.gz.age` | Gzip-compressed + encrypted SQL dump |

Compression is applied **before** encryption (age does not compress, and encrypted data is incompressible). The restore command automatically detects the suffixes and reverses the transformations in the correct order (decrypt first, then decompress).

To generate a key pair:

```bash
age-keygen -o key.txt   # contains the identity (AGE-SECRET-KEY-1...) and recipient (age1...)
```

Keep `key.txt` (the private key) offline or on your restore host. Put the `age1...` public key into `ENCRYPTION_PUBLIC_KEY` or `ENCRYPTION_PUBLIC_KEY_FILE` on the app server.

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
  -e ENCRYPTION_ENABLED=true \
  -e ENCRYPTION_PUBLIC_KEY=age1... \
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
- `restore`: Restore a backup from S3 to a database
- `list-backups`: List available backups in S3

Example:

```bash
# Run the scheduler
docker run nilsmarti/go-dbdumper:latest run

# Run a backup immediately
docker run nilsmarti/go-dbdumper:latest backup-now

# List available backups
docker run nilsmarti/go-dbdumper:latest list-backups

# Restore the latest backup
docker run nilsmarti/go-dbdumper:latest restore

# Restore a specific backup
docker run nilsmarti/go-dbdumper:latest restore --object backup/mydb-mysql-20260101-120000.sql.age

# Dry-run: decrypt and print to stdout without modifying the database
docker run nilsmarti/go-dbdumper:latest restore --dry-run
```

### Restore Configuration

The `restore` command is intended to run on a **dedicated restore host** that has the age private key — never on the app server that creates backups. It uses the same DB and S3 env vars as backup, plus the decryption key:

| Variable | Description | Default |
|----------|-------------|--------|
| `DECRYPTION_PRIVATE_KEY` | Age identity (`AGE-SECRET-KEY-1...`), mutually exclusive with `DECRYPTION_PRIVATE_KEY_FILE` | *required for encrypted backups* |
| `DECRYPTION_PRIVATE_KEY_FILE` | Path to a file containing an age identity | *alternative to above* |

The restore command automatically detects whether a backup is encrypted based on the `.sql.age` suffix and decrypts it if needed. For unencrypted backups (`.sql`), no decryption key is required.

```bash
# Restore on a dedicated host with the private key
docker run -i --rm \
  -e DB_TYPE=mysql \
  -e DB_HOST=target-db \
  -e DB_NAME=mydb \
  -e DB_USER=dbuser \
  -e DB_PASSWORD=dbpassword \
  -e S3_ENDPOINT=your-s3-endpoint \
  -e S3_BUCKET=backups \
  -e S3_ACCESS_KEY=your-access-key \
  -e S3_SECRET_KEY=your-secret-key \
  -e DECRYPTION_PRIVATE_KEY_FILE=/run/secrets/age_identity \
  -v /path/to/key.txt:/run/secrets/age_identity:ro \
  nilsmarti/go-dbdumper:latest restore
```

## Security Hardening

For workloads handling sensitive PII (e.g., Swiss AHV/SS numbers, patient data), the following layered defenses are recommended. Encryption alone protects backup *confidentiality* but does not prevent a compromised app server from *tampering with the backup set* — that requires immutability.

### 1. Client-side encryption (age)

Already covered above. The app server holds only the public key and cannot decrypt past backups.

### 2. S3 Object Lock (WORM immutability)

Object Lock prevents objects from being deleted or overwritten for a configured retention period, even by the S3 credentials that created them. This means a compromised app server cannot delete or replace existing backups.

**Bucket setup (AWS S3):**
1. Create a new bucket with Object Lock enabled (cannot be added to existing buckets).
2. Enable versioning (required for Object Lock).
3. Configure a default retention mode and period at the bucket level, or set per-object retention on upload (this tool does the latter via `OBJECT_LOCK_MODE` and `OBJECT_LOCK_RETAIN_UNTIL_DAYS`).

**Two retention modes:**
- `GOVERNANCE` — privileged users (with `s3:BypassGovernanceRetention` permission) can bypass or shorten retention. Good for operational flexibility.
- `COMPLIANCE` — no one, including the root account, can shorten retention until it expires. Required for regulatory compliance (e.g., financial/medical records).

**With this tool:**
```bash
-e CLEANUP_ENABLED=false \
-e OBJECT_LOCK_MODE=COMPLIANCE \
-e OBJECT_LOCK_RETAIN_UNTIL_DAYS=365
```

Set `CLEANUP_ENABLED=false` because the app should not have delete permissions — retention is enforced server-side.

### 3. Least-privilege IAM

The app server's S3 credentials should have the minimum permissions needed. Use separate credentials for backup and restore:

**App server (backup only — no delete):**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:PutObject",
        "s3:GetObject",
        "s3:ListBucket",
        "s3:GetBucketLocation"
      ],
      "Resource": [
        "arn:aws:s3:::your-backup-bucket",
        "arn:aws:s3:::your-backup-bucket/*"
      ]
    }
  ]
}
```

Do **not** grant `s3:DeleteObject` to the app server. When Object Lock is enabled, the app cannot delete objects even if it tries — but defense in depth means not granting the permission in the first place.

If using Object Lock in `GOVERNANCE` mode, also deny `s3:BypassGovernanceRetention`:
```json
{
  "Effect": "Deny",
  "Action": "s3:BypassGovernanceRetention",
  "Resource": "arn:aws:s3:::your-backup-bucket/*"
}
```

**Restore host (read + delete for cleanup):**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:ListBucket",
        "s3:GetBucketLocation",
        "s3:DeleteObject"
      ],
      "Resource": [
        "arn:aws:s3:::your-backup-bucket",
        "arn:aws:s3:::your-backup-bucket/*"
      ]
    }
  ]
}
```

### 4. SSE-KMS (server-side encryption, defense in depth)

In addition to client-side age encryption, enable SSE-KMS on the bucket. This adds a second layer of encryption at rest with a KMS key you control. Even if someone gains physical access to the S3 storage, the data is encrypted with your KMS key. The age encryption and SSE-KMS are independent — both must be compromised to read the data.

### Summary of defense layers

| Layer | Protects against | Config |
|-------|-----------------|--------|
| Client-side age encryption | S3/storage compromise, rogue storage admin | `ENCRYPTION_ENABLED=true` |
| TLS to S3 | Network eavesdropping | `S3_USE_SSL=true` (default) |
| S3 Object Lock | App server compromise (can't delete/overwrite backups) | `OBJECT_LOCK_MODE=COMPLIANCE` |
| Least-privilege IAM | App server compromise (no delete permission) | IAM policy without `s3:DeleteObject` |
| SSE-KMS | Physical storage access | Bucket-level config (AWS) |
| Private key on restore host only | App server compromise (can't decrypt backups) | `DECRYPTION_PRIVATE_KEY_FILE` only on restore host |

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
