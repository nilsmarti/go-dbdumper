package backup

import (
	"testing"

	"github.com/nilsmarti/go-dbdumper/config"
)

// TestCreateMySQLDumpCmd tests the creation of MySQL dump command
func TestCreateMySQLDumpCmd(t *testing.T) {
	// Create a test configuration
	cfg := &config.Config{
		DBType:     config.MySQL,
		DBHost:     "localhost",
		DBPort:     "3306",
		DBName:     "testdb",
		DBUser:     "user",
		DBPassword: "password",
	}

	// Create a service with the test configuration
	svc := &Service{cfg: cfg}

	// Create the MySQL dump command
	cmd := svc.createMySQLDumpCmd()

	// Verify the command
	if cmd.Path == "" {
		t.Error("Expected command path to be set")
	}

	// Check that the command has the right arguments
	args := cmd.Args
	expectedArgs := []string{
		"mysqldump",
		"--host", "localhost",
		"--port", "3306",
		"--user", "user",
		"--password=password",
		"--single-transaction",
		"--quick",
		"--lock-tables=false",
		"testdb",
	}

	if len(args) != len(expectedArgs) {
		t.Errorf("Expected %d arguments, got %d", len(expectedArgs), len(args))
	}

	// Check each argument
	for i, expected := range expectedArgs {
		if i < len(args) && args[i] != expected {
			t.Errorf("Expected argument %d to be '%s', got '%s'", i, expected, args[i])
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

	// Create the PostgreSQL dump command
	cmd := svc.createPgDumpCmd()

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
		t.Errorf("Expected %d arguments, got %d", len(expectedArgs), len(args))
	}

	// Check each argument
	for i, expected := range expectedArgs {
		if i < len(args) && args[i] != expected {
			t.Errorf("Expected argument %d to be '%s', got '%s'", i, expected, args[i])
		}
	}

	// Check that PGPASSWORD environment variable is set
	envs := cmd.Env
	pgPasswordFound := false
	for _, env := range envs {
		if env == "PGPASSWORD=password" {
			pgPasswordFound = true
			break
		}
	}

	if !pgPasswordFound {
		t.Error("Expected PGPASSWORD environment variable to be set")
	}
}
