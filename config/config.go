package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// DatabaseType represents the type of database
type DatabaseType string

const (
	// MySQL database type
	MySQL DatabaseType = "mysql"
	// PostgreSQL database type
	PostgreSQL DatabaseType = "postgres"
)

// Config holds all application configuration
type Config struct {
	// Database configuration
	DBType     DatabaseType
	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string

	// S3 configuration
	S3Endpoint  string
	S3Region    string
	S3Bucket    string
	S3AccessKey string
	S3SecretKey string
	S3UseSSL    bool

	// Backup configuration
	CronExpression string
	KeepLast       int
	BackupPrefix   string

	// Encryption configuration
	EncryptionEnabled       bool
	EncryptionPublicKey     string   // age recipient key (age1...) — mutually exclusive with EncryptionPublicKeyFile
	EncryptionPublicKeyFile string   // path to a file containing one or more age recipients
	EncryptionRecipients    []string // parsed recipient keys, ready for use

	// Decryption configuration (restore only). The private key (age identity)
	// should only be present on a dedicated restore host, never on the app
	// server that creates backups.
	DecryptionPrivateKey     string // age identity (AGE-SECRET-KEY-1...) — mutually exclusive with DecryptionPrivateKeyFile
	DecryptionPrivateKeyFile string // path to a file containing an age identity
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	dbType := os.Getenv("DB_TYPE")
	if dbType == "" {
		dbType = string(MySQL) // Default to MySQL
	}

	if dbType != string(MySQL) && dbType != string(PostgreSQL) {
		return nil, fmt.Errorf("invalid DB_TYPE: %s, must be 'mysql' or 'postgres'", dbType)
	}

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		return nil, errors.New("DB_HOST environment variable is required")
	}

	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		// Set default ports based on database type
		if DatabaseType(dbType) == MySQL {
			dbPort = "3306"
		} else {
			dbPort = "5432"
		}
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		return nil, errors.New("DB_NAME environment variable is required")
	}

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		return nil, errors.New("DB_USER environment variable is required")
	}

	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		return nil, errors.New("DB_PASSWORD environment variable is required")
	}

	s3Endpoint := os.Getenv("S3_ENDPOINT")
	if s3Endpoint == "" {
		return nil, errors.New("S3_ENDPOINT environment variable is required")
	}

	s3Region := os.Getenv("S3_REGION")
	if s3Region == "" {
		s3Region = "us-east-1" // Default region
	}

	s3Bucket := os.Getenv("S3_BUCKET")
	if s3Bucket == "" {
		return nil, errors.New("S3_BUCKET environment variable is required")
	}

	s3AccessKey := os.Getenv("S3_ACCESS_KEY")
	if s3AccessKey == "" {
		return nil, errors.New("S3_ACCESS_KEY environment variable is required")
	}

	s3SecretKey := os.Getenv("S3_SECRET_KEY")
	if s3SecretKey == "" {
		return nil, errors.New("S3_SECRET_KEY environment variable is required")
	}

	s3UseSSLStr := os.Getenv("S3_USE_SSL")
	s3UseSSL := true // Default to true
	if s3UseSSLStr != "" {
		var err error
		s3UseSSL, err = strconv.ParseBool(s3UseSSLStr)
		if err != nil {
			return nil, fmt.Errorf("invalid S3_USE_SSL value: %v", err)
		}
	}

	cronExpression := os.Getenv("CRON_EXPRESSION")
	if cronExpression == "" {
		cronExpression = "0 0 * * *" // Default to daily at midnight
	}

	keepLastStr := os.Getenv("KEEP_LAST")
	keepLast := 5 // Default to keeping last 5 backups
	if keepLastStr != "" {
		var err error
		keepLast, err = strconv.Atoi(keepLastStr)
		if err != nil {
			return nil, fmt.Errorf("invalid KEEP_LAST value: %v", err)
		}
		if keepLast < 1 {
			return nil, errors.New("KEEP_LAST must be at least 1")
		}
	}

	backupPrefix := os.Getenv("BACKUP_PREFIX")
	if backupPrefix == "" {
		backupPrefix = "backup" // Default prefix
	}

	// --- Encryption configuration ---
	encryptionEnabledStr := os.Getenv("ENCRYPTION_ENABLED")
	encryptionEnabled := false
	if encryptionEnabledStr != "" {
		var err error
		encryptionEnabled, err = strconv.ParseBool(encryptionEnabledStr)
		if err != nil {
			return nil, fmt.Errorf("invalid ENCRYPTION_ENABLED value: %v", err)
		}
	}

	encryptionPublicKey := os.Getenv("ENCRYPTION_PUBLIC_KEY")
	encryptionPublicKeyFile := os.Getenv("ENCRYPTION_PUBLIC_KEY_FILE")

	if encryptionEnabled {
		if encryptionPublicKey == "" && encryptionPublicKeyFile == "" {
			return nil, errors.New("ENCRYPTION_ENABLED is true but neither ENCRYPTION_PUBLIC_KEY nor ENCRYPTION_PUBLIC_KEY_FILE is set")
		}
		if encryptionPublicKey != "" && encryptionPublicKeyFile != "" {
			return nil, errors.New("ENCRYPTION_PUBLIC_KEY and ENCRYPTION_PUBLIC_KEY_FILE are mutually exclusive")
		}
	}

	// Parse recipients eagerly so that config errors surface at startup rather
	// than at the first backup.
	var encryptionRecipients []string
	if encryptionEnabled {
		if encryptionPublicKeyFile != "" {
			data, err := os.ReadFile(encryptionPublicKeyFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read ENCRYPTION_PUBLIC_KEY_FILE: %w", err)
			}
			for _, line := range splitNonEmptyLines(string(data)) {
				encryptionRecipients = append(encryptionRecipients, line)
			}
		} else {
			encryptionRecipients = splitNonEmptyLines(encryptionPublicKey)
		}
		if len(encryptionRecipients) == 0 {
			return nil, errors.New("ENCRYPTION_ENABLED is true but no valid recipient keys were found")
		}
	}

	// --- Decryption configuration (restore only) ---
	decryptionPrivateKey := os.Getenv("DECRYPTION_PRIVATE_KEY")
	decryptionPrivateKeyFile := os.Getenv("DECRYPTION_PRIVATE_KEY_FILE")
	if decryptionPrivateKey != "" && decryptionPrivateKeyFile != "" {
		return nil, errors.New("DECRYPTION_PRIVATE_KEY and DECRYPTION_PRIVATE_KEY_FILE are mutually exclusive")
	}

	return &Config{
		DBType:                   DatabaseType(dbType),
		DBHost:                   dbHost,
		DBPort:                   dbPort,
		DBName:                   dbName,
		DBUser:                   dbUser,
		DBPassword:               dbPassword,
		S3Endpoint:               s3Endpoint,
		S3Region:                 s3Region,
		S3Bucket:                 s3Bucket,
		S3AccessKey:              s3AccessKey,
		S3SecretKey:              s3SecretKey,
		S3UseSSL:                 s3UseSSL,
		CronExpression:           cronExpression,
		KeepLast:                 keepLast,
		BackupPrefix:             backupPrefix,
		EncryptionEnabled:        encryptionEnabled,
		EncryptionPublicKey:      encryptionPublicKey,
		EncryptionPublicKeyFile:  encryptionPublicKeyFile,
		EncryptionRecipients:     encryptionRecipients,
		DecryptionPrivateKey:     decryptionPrivateKey,
		DecryptionPrivateKeyFile: decryptionPrivateKeyFile,
	}, nil
}

// splitNonEmptyLines splits a string by newlines/commas and returns non-empty,
// trimmed lines. Lines starting with "#" are treated as comments and ignored.
func splitNonEmptyLines(s string) []string {
	var result []string
	for _, line := range strings.FieldsFunc(s, func(r rune) bool { return r == '\n' || r == ',' }) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		result = append(result, line)
	}
	return result
}
