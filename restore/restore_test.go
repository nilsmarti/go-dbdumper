package restore

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/nilsmarti/go-dbdumper/config"
)

func TestCreateMySQLRestoreCmd(t *testing.T) {
	cfg := &config.Config{
		DBType:     config.MySQL,
		DBHost:     "localhost",
		DBPort:     "3306",
		DBName:     "testdb",
		DBUser:     "user",
		DBPassword: "s3cr3t-pw-123",
	}

	svc := &Service{cfg: cfg}

	defaultsFile, err := svc.writeMySQLDefaultsFile()
	if err != nil {
		t.Fatalf("Failed to write defaults file: %v", err)
	}
	defer os.Remove(defaultsFile)

	cmd := svc.createMySQLRestoreCmd(context.Background(), defaultsFile)

	if cmd.Path == "" {
		t.Error("Expected command path to be set")
	}

	args := cmd.Args
	expectedArgs := []string{
		"mysql",
		"--defaults-extra-file=" + defaultsFile,
		"testdb",
	}

	if len(args) != len(expectedArgs) {
		t.Errorf("Expected %d arguments, got %d: %v", len(expectedArgs), len(args), args)
	}

	for i, expected := range expectedArgs {
		if i < len(args) && args[i] != expected {
			t.Errorf("Expected argument %d to be '%s', got '%s'", i, expected, args[i])
		}
	}

	// Ensure no argument leaks the password.
	for _, a := range args {
		if strings.Contains(a, cfg.DBPassword) {
			t.Errorf("password must not appear on the command line, found in arg: %s", a)
		}
	}
}

func TestCreatePsqlRestoreCmd(t *testing.T) {
	cfg := &config.Config{
		DBType:     config.PostgreSQL,
		DBHost:     "localhost",
		DBPort:     "5432",
		DBName:     "testdb",
		DBUser:     "user",
		DBPassword: "s3cr3t-pw-123",
	}

	svc := &Service{cfg: cfg}

	pgpassFile, err := svc.writePgPassFile()
	if err != nil {
		t.Fatalf("Failed to write pgpass file: %v", err)
	}
	defer os.Remove(pgpassFile)

	cmd := svc.createPsqlRestoreCmd(context.Background(), pgpassFile)

	if cmd.Path == "" {
		t.Error("Expected command path to be set")
	}

	args := cmd.Args
	expectedArgs := []string{
		"psql",
		"--host", "localhost",
		"--port", "5432",
		"--username", "user",
		"--dbname", "testdb",
		"--quiet",
	}

	if len(args) != len(expectedArgs) {
		t.Errorf("Expected %d arguments, got %d: %v", len(expectedArgs), len(args), args)
	}

	for i, expected := range expectedArgs {
		if i < len(args) && args[i] != expected {
			t.Errorf("Expected argument %d to be '%s', got '%s'", i, expected, args[i])
		}
	}

	// Check PGPASSFILE is set, PGPASSWORD is not.
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

func TestWriteSecretTempFile(t *testing.T) {
	path, err := writeSecretTempFile("test-restore-secret-", "supersecret")
	if err != nil {
		t.Fatalf("writeSecretTempFile failed: %v", err)
	}
	defer os.Remove(path)

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
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

func TestLoadIdentityMissing(t *testing.T) {
	cfg := &config.Config{
		DecryptionPrivateKey:     "",
		DecryptionPrivateKeyFile: "",
	}
	svc := &Service{cfg: cfg}

	_, err := svc.loadIdentity()
	if err == nil {
		t.Fatal("Expected error when no decryption key is configured")
	}
}
