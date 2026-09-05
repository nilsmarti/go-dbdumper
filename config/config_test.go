package config

import (
	"os"
	"testing"
	"time"
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

// setBaseEnv sets the minimum required env vars for a valid config.
func setBaseEnv(t *testing.T) {
	t.Helper()
	os.Setenv("DB_TYPE", "mysql")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_NAME", "testdb")
	os.Setenv("DB_USER", "user")
	os.Setenv("DB_PASSWORD", "password")
	os.Setenv("S3_ENDPOINT", "localhost:9000")
	os.Setenv("S3_BUCKET", "backups")
	os.Setenv("S3_ACCESS_KEY", "accesskey")
	os.Setenv("S3_SECRET_KEY", "secretkey")
	os.Unsetenv("DB_PORT")
	os.Unsetenv("KEEP_LAST")
	os.Unsetenv("CRON_EXPRESSION")
	os.Unsetenv("BACKUP_PREFIX")
	os.Unsetenv("S3_REGION")
	os.Unsetenv("S3_USE_SSL")
	os.Unsetenv("BACKUP_TIMEOUT")
	os.Unsetenv("COMPRESSION_ENABLED")
	os.Unsetenv("CLEANUP_ENABLED")
	os.Unsetenv("OBJECT_LOCK_MODE")
	os.Unsetenv("OBJECT_LOCK_RETAIN_UNTIL_DAYS")
}

// clearEncryptionEnv unsets all encryption-related env vars.
func clearEncryptionEnv() {
	os.Unsetenv("ENCRYPTION_ENABLED")
	os.Unsetenv("ENCRYPTION_PUBLIC_KEY")
	os.Unsetenv("ENCRYPTION_PUBLIC_KEY_FILE")
}

func TestLoadEncryptionDisabledByDefault(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}
	if cfg.EncryptionEnabled {
		t.Error("Expected EncryptionEnabled to be false by default")
	}
	if len(cfg.EncryptionRecipients) != 0 {
		t.Errorf("Expected no recipients when disabled, got %d", len(cfg.EncryptionRecipients))
	}
}

func TestLoadEncryptionEnabledWithoutKey(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()
	os.Setenv("ENCRYPTION_ENABLED", "true")

	_, err := Load()
	if err == nil {
		t.Fatal("Expected error when encryption enabled without a key, got nil")
	}
}

func TestLoadEncryptionEnabledWithKey(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()
	os.Setenv("ENCRYPTION_ENABLED", "true")
	// A dummy key string. Config validation only checks presence; age format
	// validation happens at backup time in the encryption package.
	os.Setenv("ENCRYPTION_PUBLIC_KEY", "age1dummykey")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config with encryption key: %v", err)
	}
	if !cfg.EncryptionEnabled {
		t.Error("Expected EncryptionEnabled to be true")
	}
	if len(cfg.EncryptionRecipients) != 1 {
		t.Errorf("Expected 1 recipient, got %d", len(cfg.EncryptionRecipients))
	}
	if cfg.EncryptionRecipients[0] != "age1dummykey" {
		t.Errorf("Expected recipient 'age1dummykey', got %q", cfg.EncryptionRecipients[0])
	}
}

func TestLoadEncryptionBothKeySources(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()
	os.Setenv("ENCRYPTION_ENABLED", "true")
	os.Setenv("ENCRYPTION_PUBLIC_KEY", "age1dummy")
	os.Setenv("ENCRYPTION_PUBLIC_KEY_FILE", "/tmp/dummy")

	_, err := Load()
	if err == nil {
		t.Fatal("Expected error when both key sources are set, got nil")
	}
}

func TestLoadEncryptionInvalidEnabled(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()
	os.Setenv("ENCRYPTION_ENABLED", "not-a-bool")

	_, err := Load()
	if err == nil {
		t.Fatal("Expected error for invalid ENCRYPTION_ENABLED value, got nil")
	}
}

// clearDecryptionEnv unsets all decryption-related env vars.
func clearDecryptionEnv() {
	os.Unsetenv("DECRYPTION_PRIVATE_KEY")
	os.Unsetenv("DECRYPTION_PRIVATE_KEY_FILE")
}

func TestLoadDecryptionBothKeySources(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()
	clearDecryptionEnv()
	os.Setenv("DECRYPTION_PRIVATE_KEY", "AGE-SECRET-KEY-1dummy")
	os.Setenv("DECRYPTION_PRIVATE_KEY_FILE", "/tmp/dummy")

	_, err := Load()
	if err == nil {
		t.Fatal("Expected error when both decryption key sources are set, got nil")
	}
}

func TestLoadDecryptionNoneRequired(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()
	clearDecryptionEnv()

	// Decryption keys are optional — only needed for restore, not for backup.
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config without decryption keys: %v", err)
	}
	if cfg.DecryptionPrivateKey != "" {
		t.Error("Expected DecryptionPrivateKey to be empty")
	}
	if cfg.DecryptionPrivateKeyFile != "" {
		t.Error("Expected DecryptionPrivateKeyFile to be empty")
	}
}

func TestLoadDecryptionKeySet(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()
	clearDecryptionEnv()
	os.Setenv("DECRYPTION_PRIVATE_KEY", "AGE-SECRET-KEY-1dummy")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config with decryption key: %v", err)
	}
	if cfg.DecryptionPrivateKey != "AGE-SECRET-KEY-1dummy" {
		t.Errorf("Expected DecryptionPrivateKey 'AGE-SECRET-KEY-1dummy', got %q", cfg.DecryptionPrivateKey)
	}
}

func TestLoadBackupTimeoutDefault(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()
	clearDecryptionEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if cfg.BackupTimeout != 30*time.Minute {
		t.Errorf("Expected default BackupTimeout 30m, got %v", cfg.BackupTimeout)
	}
}

func TestLoadBackupTimeoutCustom(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()
	clearDecryptionEnv()
	os.Setenv("BACKUP_TIMEOUT", "1h30m")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	expected := 90 * time.Minute
	if cfg.BackupTimeout != expected {
		t.Errorf("Expected BackupTimeout %v, got %v", expected, cfg.BackupTimeout)
	}
}

func TestLoadBackupTimeoutInvalid(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()
	clearDecryptionEnv()
	os.Setenv("BACKUP_TIMEOUT", "not-a-duration")

	_, err := Load()
	if err == nil {
		t.Fatal("Expected error for invalid BACKUP_TIMEOUT, got nil")
	}
}

func TestLoadBackupTimeoutZero(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()
	clearDecryptionEnv()
	os.Setenv("BACKUP_TIMEOUT", "0s")

	_, err := Load()
	if err == nil {
		t.Fatal("Expected error for zero BACKUP_TIMEOUT, got nil")
	}
}

func TestLoadCompressionDefaultEnabled(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()
	clearDecryptionEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if !cfg.CompressionEnabled {
		t.Error("Expected CompressionEnabled to be true by default")
	}
}

func TestLoadCompressionDisabled(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()
	clearDecryptionEnv()
	os.Setenv("COMPRESSION_ENABLED", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if cfg.CompressionEnabled {
		t.Error("Expected CompressionEnabled to be false")
	}
}

func TestLoadCompressionInvalid(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()
	clearDecryptionEnv()
	os.Setenv("COMPRESSION_ENABLED", "not-a-bool")

	_, err := Load()
	if err == nil {
		t.Fatal("Expected error for invalid COMPRESSION_ENABLED, got nil")
	}
}

func TestLoadCleanupEnabledByDefault(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()
	clearDecryptionEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if !cfg.CleanupEnabled {
		t.Error("Expected CleanupEnabled to be true by default")
	}
}

func TestLoadCleanupDisabled(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()
	clearDecryptionEnv()
	os.Setenv("CLEANUP_ENABLED", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if cfg.CleanupEnabled {
		t.Error("Expected CleanupEnabled to be false")
	}
}

func TestLoadCleanupInvalid(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()
	clearDecryptionEnv()
	os.Setenv("CLEANUP_ENABLED", "not-a-bool")

	_, err := Load()
	if err == nil {
		t.Fatal("Expected error for invalid CLEANUP_ENABLED, got nil")
	}
}

func TestLoadObjectLockGovernance(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()
	clearDecryptionEnv()
	os.Setenv("OBJECT_LOCK_MODE", "governance")
	os.Setenv("OBJECT_LOCK_RETAIN_UNTIL_DAYS", "30")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if cfg.ObjectLockMode != "GOVERNANCE" {
		t.Errorf("Expected ObjectLockMode 'GOVERNANCE', got %q", cfg.ObjectLockMode)
	}
	if cfg.ObjectLockRetainUntilDays != 30 {
		t.Errorf("Expected ObjectLockRetainUntilDays 30, got %d", cfg.ObjectLockRetainUntilDays)
	}
}

func TestLoadObjectLockCompliance(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()
	clearDecryptionEnv()
	os.Setenv("OBJECT_LOCK_MODE", "COMPLIANCE")
	os.Setenv("OBJECT_LOCK_RETAIN_UNTIL_DAYS", "90")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if cfg.ObjectLockMode != "COMPLIANCE" {
		t.Errorf("Expected ObjectLockMode 'COMPLIANCE', got %q", cfg.ObjectLockMode)
	}
}

func TestLoadObjectLockInvalidMode(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()
	clearDecryptionEnv()
	os.Setenv("OBJECT_LOCK_MODE", "INVALID")
	os.Setenv("OBJECT_LOCK_RETAIN_UNTIL_DAYS", "30")

	_, err := Load()
	if err == nil {
		t.Fatal("Expected error for invalid OBJECT_LOCK_MODE, got nil")
	}
}

func TestLoadObjectLockModeWithoutDays(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()
	clearDecryptionEnv()
	os.Setenv("OBJECT_LOCK_MODE", "GOVERNANCE")
	// No OBJECT_LOCK_RETAIN_UNTIL_DAYS set

	_, err := Load()
	if err == nil {
		t.Fatal("Expected error when OBJECT_LOCK_MODE set without retain days, got nil")
	}
}

func TestLoadObjectLockInvalidDays(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()
	clearDecryptionEnv()
	os.Setenv("OBJECT_LOCK_MODE", "GOVERNANCE")
	os.Setenv("OBJECT_LOCK_RETAIN_UNTIL_DAYS", "0")

	_, err := Load()
	if err == nil {
		t.Fatal("Expected error for zero OBJECT_LOCK_RETAIN_UNTIL_DAYS, got nil")
	}
}

func TestLoadObjectLockNotSetByDefault(t *testing.T) {
	setBaseEnv(t)
	clearEncryptionEnv()
	clearDecryptionEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if cfg.ObjectLockMode != "" {
		t.Errorf("Expected empty ObjectLockMode by default, got %q", cfg.ObjectLockMode)
	}
	if cfg.ObjectLockRetainUntilDays != 0 {
		t.Errorf("Expected 0 ObjectLockRetainUntilDays by default, got %d", cfg.ObjectLockRetainUntilDays)
	}
}
