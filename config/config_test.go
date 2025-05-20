package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Set up test environment variables
	os.Setenv("DB_TYPE", "mysql")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_NAME", "testdb")
	os.Setenv("DB_USER", "user")
	os.Setenv("DB_PASSWORD", "password")
	os.Setenv("S3_ENDPOINT", "localhost:9000")
	os.Setenv("S3_BUCKET", "backups")
	os.Setenv("S3_ACCESS_KEY", "accesskey")
	os.Setenv("S3_SECRET_KEY", "secretkey")
	os.Setenv("KEEP_LAST", "3")

	// Load configuration
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Verify configuration values
	if cfg.DBType != MySQL {
		t.Errorf("Expected DBType to be %s, got %s", MySQL, cfg.DBType)
	}

	if cfg.DBHost != "localhost" {
		t.Errorf("Expected DBHost to be localhost, got %s", cfg.DBHost)
	}

	if cfg.DBPort != "3306" {
		t.Errorf("Expected DBPort to be 3306, got %s", cfg.DBPort)
	}

	if cfg.DBName != "testdb" {
		t.Errorf("Expected DBName to be testdb, got %s", cfg.DBName)
	}

	if cfg.DBUser != "user" {
		t.Errorf("Expected DBUser to be user, got %s", cfg.DBUser)
	}

	if cfg.DBPassword != "password" {
		t.Errorf("Expected DBPassword to be password, got %s", cfg.DBPassword)
	}

	if cfg.S3Endpoint != "localhost:9000" {
		t.Errorf("Expected S3Endpoint to be localhost:9000, got %s", cfg.S3Endpoint)
	}

	if cfg.S3Bucket != "backups" {
		t.Errorf("Expected S3Bucket to be backups, got %s", cfg.S3Bucket)
	}

	if cfg.S3AccessKey != "accesskey" {
		t.Errorf("Expected S3AccessKey to be accesskey, got %s", cfg.S3AccessKey)
	}

	if cfg.S3SecretKey != "secretkey" {
		t.Errorf("Expected S3SecretKey to be secretkey, got %s", cfg.S3SecretKey)
	}

	if cfg.KeepLast != 3 {
		t.Errorf("Expected KeepLast to be 3, got %d", cfg.KeepLast)
	}

	// Test default values
	if cfg.CronExpression != "0 0 * * *" {
		t.Errorf("Expected CronExpression to be '0 0 * * *', got %s", cfg.CronExpression)
	}

	if cfg.BackupPrefix != "backup" {
		t.Errorf("Expected BackupPrefix to be 'backup', got %s", cfg.BackupPrefix)
	}
}

func TestLoadInvalidKeepLast(t *testing.T) {
	// Set up test environment variables with invalid KEEP_LAST
	os.Setenv("DB_TYPE", "mysql")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_NAME", "testdb")
	os.Setenv("DB_USER", "user")
	os.Setenv("DB_PASSWORD", "password")
	os.Setenv("S3_ENDPOINT", "localhost:9000")
	os.Setenv("S3_BUCKET", "backups")
	os.Setenv("S3_ACCESS_KEY", "accesskey")
	os.Setenv("S3_SECRET_KEY", "secretkey")
	os.Setenv("KEEP_LAST", "invalid")

	// Load configuration should fail
	_, err := Load()
	if err == nil {
		t.Fatal("Expected error for invalid KEEP_LAST, got nil")
	}
}

func TestLoadInvalidDBType(t *testing.T) {
	// Set up test environment variables with invalid DB_TYPE
	os.Setenv("DB_TYPE", "invalid")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_NAME", "testdb")
	os.Setenv("DB_USER", "user")
	os.Setenv("DB_PASSWORD", "password")
	os.Setenv("S3_ENDPOINT", "localhost:9000")
	os.Setenv("S3_BUCKET", "backups")
	os.Setenv("S3_ACCESS_KEY", "accesskey")
	os.Setenv("S3_SECRET_KEY", "secretkey")

	// Load configuration should fail
	_, err := Load()
	if err == nil {
		t.Fatal("Expected error for invalid DB_TYPE, got nil")
	}
}
