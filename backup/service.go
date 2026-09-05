package backup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/nilsmarti/go-dbdumper/config"
	"github.com/nilsmarti/go-dbdumper/storage"
)

// Service handles database backup operations
type Service struct {
	cfg      *config.Config
	s3Client *storage.S3Client
}

// NewService creates a new backup service
func NewService(cfg *config.Config) (*Service, error) {
	// Initialize S3 client
	s3Client, err := storage.NewS3Client(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize S3 client: %w", err)
	}

	return &Service{
		cfg:      cfg,
		s3Client: s3Client,
	}, nil
}

// PerformBackup performs a database backup and uploads it to S3.
// The provided context governs the lifetime of the dump command and the S3
// upload; cancelling it aborts an in-flight backup cleanly (the multipart
// upload is aborted by minio, leaving no partial object in S3).
func (s *Service) PerformBackup(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	fmt.Printf("Starting backup of %s database %s at %s\n",
		s.cfg.DBType, s.cfg.DBName, time.Now().Format(time.RFC3339))

	// Create a pipe to stream the dump directly to S3
	pr, pw := io.Pipe()

	// Prepare credentials in temp files (mode 0600) so they never appear on
	// the command line (visible via ps/proc) or in the process environment
	// (visible via /proc/<pid>/environ).
	var credsFile string
	var err error
	switch s.cfg.DBType {
	case config.MySQL:
		credsFile, err = s.writeMySQLDefaultsFile()
	case config.PostgreSQL:
		credsFile, err = s.writePgPassFile()
	default:
		return fmt.Errorf("unsupported database type: %s", s.cfg.DBType)
	}
	if err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}
	defer os.Remove(credsFile)

	// Start the dump in a goroutine so it streams into the pipe while the
	// upload reads from the other end.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer pw.Close()

		var cmd *exec.Cmd
		switch s.cfg.DBType {
		case config.MySQL:
			cmd = s.createMySQLDumpCmd(ctx, credsFile)
		case config.PostgreSQL:
			cmd = s.createPgDumpCmd(ctx, credsFile)
		default:
			pw.CloseWithError(fmt.Errorf("unsupported database type: %s", s.cfg.DBType))
			return
		}

		// Create a buffer to capture stderr
		var stderr bytes.Buffer

		// Set the output to the pipe writer and capture stderr
		cmd.Stdout = pw
		cmd.Stderr = &stderr

		// Run the command
		if err := cmd.Run(); err != nil {
			errOutput := stderr.String()
			fmt.Printf("Database dump error output: %s\n", errOutput)
			pw.CloseWithError(fmt.Errorf("database dump failed: %w (stderr: %s)", err, errOutput))
		}
	}()

	// Upload the backup to S3
	objName, uploadErr := s.s3Client.UploadBackup(ctx, pr, s.cfg.DBName, string(s.cfg.DBType))
	// Wait for the dump goroutine to finish before returning so the temp
	// credentials file is not removed while the command might still read it.
	wg.Wait()
	if uploadErr != nil {
		return fmt.Errorf("failed to upload backup: %w", uploadErr)
	}

	fmt.Printf("Backup completed successfully: %s\n", objName)
	return nil
}

// writeMySQLDefaultsFile writes a mysqldump --defaults-extra-file to a temp
// file with mode 0600 and returns its path. The caller must remove the file.
func (s *Service) writeMySQLDefaultsFile() (string, error) {
	content := fmt.Sprintf("[client]\nhost=%s\nport=%s\nuser=%s\npassword=%s\n",
		s.cfg.DBHost, s.cfg.DBPort, s.cfg.DBUser, s.cfg.DBPassword)
	return writeSecretTempFile("mysqldump-defaults-", content)
}

// writePgPassFile writes a libpq pgpass file to a temp file with mode 0600 and
// returns its path. The caller must remove the file.
func (s *Service) writePgPassFile() (string, error) {
	// Format: hostname:port:database:username:password
	content := fmt.Sprintf("%s:%s:%s:%s:%s\n",
		s.cfg.DBHost, s.cfg.DBPort, s.cfg.DBName, s.cfg.DBUser, s.cfg.DBPassword)
	return writeSecretTempFile("pgpass-", content)
}

// writeSecretTempFile writes content to a new temp file with mode 0600 and
// returns its path.
func writeSecretTempFile(prefix, content string) (string, error) {
	f, err := os.CreateTemp("", prefix)
	if err != nil {
		return "", fmt.Errorf("failed to create temp credentials file: %w", err)
	}
	path := f.Name()
	// Ensure the file is only readable by the owner. CreateTemp may create the
	// file with 0600 already on Unix, but be explicit and also handle the case
	// where the umask loosened it.
	if err := os.Chmod(path, 0o600); err != nil {
		f.Close()
		os.Remove(path)
		return "", fmt.Errorf("failed to set permissions on credentials file: %w", err)
	}
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(path)
		return "", fmt.Errorf("failed to write credentials file: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return "", fmt.Errorf("failed to close credentials file: %w", err)
	}
	return path, nil
}

// createMySQLDumpCmd creates a command to dump a MySQL database. Credentials
// are read from the given defaults-extra-file instead of the command line.
func (s *Service) createMySQLDumpCmd(ctx context.Context, defaultsFile string) *exec.Cmd {
	// Build mysqldump command. --defaults-extra-file must be the first option
	// to be honoured correctly.
	cmd := exec.CommandContext(ctx, "mysqldump",
		"--defaults-extra-file="+defaultsFile,
		"--single-transaction",
		"--quick",
		"--lock-tables=false",
		"--default-auth=mysql_native_password", // Use native password authentication for MySQL 8+ compatibility
		s.cfg.DBName,
	)

	return cmd
}

// createPgDumpCmd creates a command to dump a PostgreSQL database. The
// password is read from the pgpass file referenced by PGPASSFILE.
func (s *Service) createPgDumpCmd(ctx context.Context, pgpassFile string) *exec.Cmd {
	// Build pg_dump command
	cmd := exec.CommandContext(ctx, "pg_dump",
		"--host", s.cfg.DBHost,
		"--port", s.cfg.DBPort,
		"--username", s.cfg.DBUser,
		"--dbname", s.cfg.DBName,
		"--format", "plain",
		"--no-owner",
		"--no-acl",
	)

	// Use a pgpass file instead of PGPASSWORD so the password is not visible
	// in the process environment.
	cmd.Env = append(os.Environ(), "PGPASSFILE="+pgpassFile)

	return cmd
}
