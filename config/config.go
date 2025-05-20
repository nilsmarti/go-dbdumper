package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
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
	DBType       DatabaseType
	DBHost       string
	DBPort       string
	DBName       string
	DBUser       string
	DBPassword   string

	// S3 configuration
	S3Endpoint   string
	S3Region     string
	S3Bucket     string
	S3AccessKey  string
	S3SecretKey  string
	S3UseSSL     bool

	// Backup configuration
	CronExpression string
	KeepLast       int
	BackupPrefix   string
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

	return &Config{
		DBType:         DatabaseType(dbType),
		DBHost:         dbHost,
		DBPort:         dbPort,
		DBName:         dbName,
		DBUser:         dbUser,
		DBPassword:     dbPassword,
		S3Endpoint:     s3Endpoint,
		S3Region:       s3Region,
		S3Bucket:       s3Bucket,
		S3AccessKey:    s3AccessKey,
		S3SecretKey:    s3SecretKey,
		S3UseSSL:       s3UseSSL,
		CronExpression: cronExpression,
		KeepLast:       keepLast,
		BackupPrefix:   backupPrefix,
	}, nil
}
