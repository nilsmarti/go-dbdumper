package backup

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/nilsmarti/go-dbdumper/config"
)

// TestCreateMySQLDumpCmd tests the creation of MySQL dump command
func TestCreateMySQLDumpCmd(t *testing.T) {
	// Create a test configuration. Use a distinctive password so we can
	// assert it never leaks onto the command line.
	cfg := &config.Config{
		DBType:     config.MySQL,
		DBHost:     "localhost",
		DBPort:     "3306",
		DBName:     "testdb",
		DBUser:     "user",
		DBPassword: "s3cr3t-pw-123",
	}

	// Create a service with the test configuration
	svc := &Service{cfg: cfg}

	// Write the defaults-extra-file and clean it up after the test.
	defaultsFile, err := svc.writeMySQLDefaultsFile()
	if err != nil {
		t.Fatalf("Failed to write defaults file: %v", err)
	}
	defer os.Remove(defaultsFile)

	// Create the MySQL dump command
	cmd := svc.createMySQLDumpCmd(context.Background(), defaultsFile)

	// Verify the command
	if cmd.Path == "" {
		t.Error("Expected command path to be set")
	}

	// Check that the command has the right arguments. The password must NOT
	// appear on the command line; it is read from the defaults-extra-file.
	args := cmd.Args
	expectedArgs := []string{
		"mysqldump",
		"--defaults-extra-file=" + defaultsFile,
		"--single-transaction",
		"--quick",
		"--lock-tables=false",
		"--default-auth=mysql_native_password",
		"testdb",
	}

	if len(args) != len(expectedArgs) {
		t.Errorf("Expected %d arguments, got %d: %v", len(expectedArgs), len(args), args)
	}

	// Check each argument
	for i, expected := range expectedArgs {
		if i < len(args) && args[i] != expected {
			t.Errorf("Expected argument %d to be '%s', got '%s'", i, expected, args[i])
		}
	}

	// Ensure no argument leaks the actual password value.
	for _, a := range args {
		if strings.Contains(a, cfg.DBPassword) {
			t.Errorf("password must not appear on the command line, found in arg: %s", a)
		}
	}
}

// TestCreatePgDumpCmd tests the creation of PostgreSQL dump command
func TestCreatePgDumpCmd(t *testing.T) {
	// Create a test configuration
	cfg := &config.Config{
		DBType:     config.PostgreSQL,
		DBHost:     "localhost",
		DBPort:     "5432",
		DBName:     "testdb",
		DBUser:     "user",
		DBPassword: "password",
	}

	// Create a service with the test configuration
	svc := &Service{cfg: cfg}

	// Write the pgpass file and clean it up after the test.
	pgpassFile, err := svc.writePgPassFile()
	if err != nil {
		t.Fatalf("Failed to write pgpass file: %v", err)
	}
	defer os.Remove(pgpassFile)

	// Create the PostgreSQL dump command
	cmd := svc.createPgDumpCmd(context.Background(), pgpassFile)

	// Verify the command
	if cmd.Path == "" {
		t.Error("Expected command path to be set")
	}

	// Check that the command has the right arguments
	args := cmd.Args
	expectedArgs := []string{
		"pg_dump",
		"--host", "localhost",
		"--port", "5432",
		"--username", "user",
		"--dbname", "testdb",
		"--format", "plain",
		"--no-owner",
		"--no-acl",
	}

	if len(args) != len(expectedArgs) {
		t.Errorf("Expected %d arguments, got %d: %v", len(expectedArgs), len(args), args)
	}

	// Check each argument
	for i, expected := range expectedArgs {
		if i < len(args) && args[i] != expected {
			t.Errorf("Expected argument %d to be '%s', got '%s'", i, expected, args[i])
		}
	}

	// Check that PGPASSFILE environment variable is set (not PGPASSWORD).
	pgpassFound := false
	pgpasswordFound := false
	for _, env := range cmd.Env {
		if strings.HasPrefix(env, "PGPASSFILE=") {
			pgpassFound = true
		}
		if strings.HasPrefix(env, "PGPASSWORD=") {
			pgpasswordFound = true
		}
	}

	if !pgpassFound {
		t.Error("Expected PGPASSFILE environment variable to be set")
	}
	if pgpasswordFound {
		t.Error("PGPASSWORD must not be set in the command environment")
	}
}

// TestWriteSecretTempFile verifies that secret files are created with mode
// 0600 and the expected content.
func TestWriteSecretTempFile(t *testing.T) {
	path, err := writeSecretTempFile("test-secret-", "supersecret")
	if err != nil {
		t.Fatalf("writeSecretTempFile failed: %v", err)
	}
	defer os.Remove(path)

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	// The 0600 permission is only meaningful on Unix; on Windows os.Chmod
	// only toggles the read-only bit and Perm() does not reflect Unix modes.
	// Production runs in a Linux container where the mode is enforced.
	if runtime.GOOS != "windows" {
		if mode := info.Mode().Perm(); mode != 0o600 {
			t.Errorf("Expected file mode 0600, got %o", mode)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(data) != "supersecret" {
		t.Errorf("Expected content 'supersecret', got %q", string(data))
	}
}
