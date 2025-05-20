package backup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/nilsmarti/go-dbdumper/config"
	"github.com/nilsmarti/go-dbdumper/storage"
)

// Service handles database backup operations
type Service struct {
	cfg       *config.Config
	s3Client  *storage.S3Client
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

// PerformBackup performs a database backup and uploads it to S3
func (s *Service) PerformBackup() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	fmt.Printf("Starting backup of %s database %s at %s\n", 
		s.cfg.DBType, s.cfg.DBName, time.Now().Format(time.RFC3339))

	// Create a pipe to stream the dump directly to S3
	pr, pw := io.Pipe()

	// Start the upload process in a goroutine
	var uploadErr error
	var objName string
	go func() {
		defer pw.Close()
		
		// Execute the appropriate dump command based on database type
		var cmd *exec.Cmd
		switch s.cfg.DBType {
		case config.MySQL:
			cmd = s.createMySQLDumpCmd()
		case config.PostgreSQL:
			cmd = s.createPgDumpCmd()
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
	objName, uploadErr = s.s3Client.UploadBackup(ctx, pr, s.cfg.DBName, string(s.cfg.DBType))
	if uploadErr != nil {
		return fmt.Errorf("failed to upload backup: %w", uploadErr)
	}

	fmt.Printf("Backup completed successfully: %s\n", objName)
	return nil
}

// createMySQLDumpCmd creates a command to dump a MySQL database
func (s *Service) createMySQLDumpCmd() *exec.Cmd {
	// Build mysqldump command
	cmd := exec.Command("mysqldump",
		"--host", s.cfg.DBHost,
		"--port", s.cfg.DBPort,
		"--user", s.cfg.DBUser,
		"--password=" + s.cfg.DBPassword, // Note: This is secure in this context as we're not exposing it to shell history
		"--single-transaction",
		"--quick",
		"--lock-tables=false",
		s.cfg.DBName,
	)

	return cmd
}

// createPgDumpCmd creates a command to dump a PostgreSQL database
func (s *Service) createPgDumpCmd() *exec.Cmd {
	// Build pg_dump command
	cmd := exec.Command("pg_dump",
		"--host", s.cfg.DBHost,
		"--port", s.cfg.DBPort,
		"--username", s.cfg.DBUser,
		"--dbname", s.cfg.DBName,
		"--format", "plain",
		"--no-owner",
		"--no-acl",
	)

	// Set PGPASSWORD environment variable
	cmd.Env = append(cmd.Env, "PGPASSWORD="+s.cfg.DBPassword)

	return cmd
}
