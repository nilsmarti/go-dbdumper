package restore

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"filippo.io/age"
	"github.com/nilsmarti/go-dbdumper/config"
	"github.com/nilsmarti/go-dbdumper/encryption"
	"github.com/nilsmarti/go-dbdumper/storage"
)

// Service handles database restore operations.
type Service struct {
	cfg      *config.Config
	s3Client *storage.S3Client
}

// NewService creates a new restore service.
func NewService(cfg *config.Config) (*Service, error) {
	s3Client, err := storage.NewS3Client(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize S3 client: %w", err)
	}

	return &Service{
		cfg:      cfg,
		s3Client: s3Client,
	}, nil
}

// ListBackups returns a list of all backup object names in S3.
func (s *Service) ListBackups(ctx context.Context) ([]string, error) {
	return s.s3Client.ListBackups(ctx)
}

// RestoreOptions controls which backup to restore and how.
type RestoreOptions struct {
	// ObjectName is the specific S3 object to restore. If empty, the latest
	// backup for the configured database is used.
	ObjectName string
	// DryRun, if true, downloads and decrypts (if needed) the backup but does
	// not pipe it into the database. The plaintext is written to stdout
	// instead, useful for verifying backups without modifying a database.
	DryRun bool
}

// Restore downloads a backup from S3, decrypts it if needed, and streams it
// into the appropriate database restore command (mysql or psql).
func (s *Service) Restore(ctx context.Context, opts RestoreOptions) error {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.BackupTimeout)
	defer cancel()

	objectName := opts.ObjectName
	if objectName == "" {
		fmt.Printf("No object specified, finding latest backup for %s/%s...\n",
			s.cfg.DBName, s.cfg.DBType)
		var err error
		objectName, err = s.s3Client.GetLatestBackup(ctx, s.cfg.DBName, string(s.cfg.DBType))
		if err != nil {
			return fmt.Errorf("failed to find latest backup: %w", err)
		}
	}

	fmt.Printf("Restoring backup: %s\n", objectName)

	// Determine the transformations applied based on the file suffix.
	// Suffixes: .sql (plain), .sql.gz (compressed), .sql.age (encrypted),
	// .sql.gz.age (compressed + encrypted).
	encrypted := strings.HasSuffix(objectName, ".age")
	compressed := strings.Contains(objectName, ".gz")

	// If encrypted, load the decryption identity.
	var identities []age.Identity
	if encrypted {
		identity, err := s.loadIdentity()
		if err != nil {
			return fmt.Errorf("failed to load decryption key: %w", err)
		}
		identities = append(identities, identity)
	}

	// Download the backup from S3.
	downloadReader, err := s.s3Client.DownloadBackup(ctx, objectName)
	if err != nil {
		return fmt.Errorf("failed to download backup: %w", err)
	}
	defer downloadReader.Close()

	// Build the decompression/decryption pipeline in the reverse order of how
	// the backup was created (encrypt was last, so decrypt is first):
	//   S3 → [age decrypt] → [gzip decompress] → mysql/psql stdin
	var plaintextReader io.Reader = downloadReader
	if encrypted {
		decReader, err := encryption.Decrypt(downloadReader, identities)
		if err != nil {
			return fmt.Errorf("failed to create age decryptor: %w", err)
		}
		plaintextReader = decReader
	}
	if compressed {
		gzReader, err := gzip.NewReader(plaintextReader)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		plaintextReader = gzReader
	}

	// If dry-run, pipe plaintext to stdout and exit.
	if opts.DryRun {
		if _, err := io.Copy(os.Stdout, plaintextReader); err != nil {
			return fmt.Errorf("failed to write plaintext to stdout: %w", err)
		}
		fmt.Println("\nDry run completed. No changes were made to the database.")
		return nil
	}

	// Prepare credentials in temp files (same approach as backup).
	var credsFile string
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

	// Create the restore command.
	var cmd *exec.Cmd
	switch s.cfg.DBType {
	case config.MySQL:
		cmd = s.createMySQLRestoreCmd(ctx, credsFile)
	case config.PostgreSQL:
		cmd = s.createPsqlRestoreCmd(ctx, credsFile)
	default:
		return fmt.Errorf("unsupported database type: %s", s.cfg.DBType)
	}

	// Pipe the plaintext into the command's stdin.
	cmd.Stdin = plaintextReader

	// Capture stderr for error reporting.
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	fmt.Printf("Streaming restore to %s database %s...\n", s.cfg.DBType, s.cfg.DBName)
	if err := cmd.Run(); err != nil {
		errOutput := stderr.String()
		return fmt.Errorf("database restore failed: %w (stderr: %s)", err, errOutput)
	}

	fmt.Printf("Restore completed successfully from: %s\n", objectName)
	return nil
}

// loadIdentity loads the age private key (identity) from config.
func (s *Service) loadIdentity() (age.Identity, error) {
	if s.cfg.DecryptionPrivateKey != "" {
		return encryption.ParseIdentity(s.cfg.DecryptionPrivateKey)
	}
	if s.cfg.DecryptionPrivateKeyFile != "" {
		return encryption.LoadIdentityFromFile(s.cfg.DecryptionPrivateKeyFile)
	}
	return nil, fmt.Errorf("backup is encrypted but no decryption key is configured (set DECRYPTION_PRIVATE_KEY or DECRYPTION_PRIVATE_KEY_FILE)")
}

// writeMySQLDefaultsFile writes a mysql client --defaults-extra-file to a temp
// file with mode 0600 and returns its path. The caller must remove the file.
func (s *Service) writeMySQLDefaultsFile() (string, error) {
	content := fmt.Sprintf("[client]\nhost=%s\nport=%s\nuser=%s\npassword=%s\n",
		s.cfg.DBHost, s.cfg.DBPort, s.cfg.DBUser, s.cfg.DBPassword)
	return writeSecretTempFile("mysql-restore-defaults-", content)
}

// writePgPassFile writes a libpq pgpass file to a temp file with mode 0600 and
// returns its path. The caller must remove the file.
func (s *Service) writePgPassFile() (string, error) {
	content := fmt.Sprintf("%s:%s:%s:%s:%s\n",
		s.cfg.DBHost, s.cfg.DBPort, s.cfg.DBName, s.cfg.DBUser, s.cfg.DBPassword)
	return writeSecretTempFile("pgpass-restore-", content)
}

// writeSecretTempFile writes content to a new temp file with mode 0600 and
// returns its path.
func writeSecretTempFile(prefix, content string) (string, error) {
	f, err := os.CreateTemp("", prefix)
	if err != nil {
		return "", fmt.Errorf("failed to create temp credentials file: %w", err)
	}
	path := f.Name()
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

// createMySQLRestoreCmd creates a mysql command to restore a SQL dump.
// Credentials are read from the given defaults-extra-file.
func (s *Service) createMySQLRestoreCmd(ctx context.Context, defaultsFile string) *exec.Cmd {
	return exec.CommandContext(ctx, "mysql",
		"--defaults-extra-file="+defaultsFile,
		s.cfg.DBName,
	)
}

// createPsqlRestoreCmd creates a psql command to restore a plain-text SQL
// dump (as produced by pg_dump --format plain). Credentials are read from the
// pgpass file referenced by PGPASSFILE.
func (s *Service) createPsqlRestoreCmd(ctx context.Context, pgpassFile string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "psql",
		"--host", s.cfg.DBHost,
		"--port", s.cfg.DBPort,
		"--username", s.cfg.DBUser,
		"--dbname", s.cfg.DBName,
		"--quiet",
	)
	cmd.Env = append(os.Environ(), "PGPASSFILE="+pgpassFile)
	return cmd
}
