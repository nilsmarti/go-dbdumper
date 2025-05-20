package storage

import (
	"context"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/nilsmarti/go-dbdumper/config"
)

// S3Client handles interactions with S3 compatible storage
type S3Client struct {
	client     *minio.Client
	bucketName string
	prefix     string
	keepLast   int
}

// NewS3Client creates a new S3 client
func NewS3Client(cfg *config.Config) (*S3Client, error) {
	// Initialize minio client
	client, err := minio.New(cfg.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""),
		Secure: cfg.S3UseSSL,
		Region: cfg.S3Region,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Check if bucket exists
	exists, err := client.BucketExists(context.Background(), cfg.S3Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if !exists {
		return nil, fmt.Errorf("bucket %s does not exist", cfg.S3Bucket)
	}

	return &S3Client{
		client:     client,
		bucketName: cfg.S3Bucket,
		prefix:     cfg.BackupPrefix,
		keepLast:   cfg.KeepLast,
	}, nil
}

// UploadBackup uploads a backup to S3
func (s *S3Client) UploadBackup(ctx context.Context, reader io.Reader, dbName, dbType string) (string, error) {
	// Create a timestamp for the backup filename
	timestamp := time.Now().UTC().Format("20060102-150405")

	// Create the object name with format: prefix/dbname-dbtype-timestamp.sql
	objName := fmt.Sprintf("%s/%s-%s-%s.sql", s.prefix, dbName, dbType, timestamp)

	// Upload the backup
	_, err := s.client.PutObject(ctx, s.bucketName, objName, reader, -1,
		minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return "", fmt.Errorf("failed to upload backup: %w", err)
	}

	// Clean up old backups
	if err := s.cleanupOldBackups(ctx, dbName, dbType); err != nil {
		// Just log the error but don't fail the backup
		fmt.Printf("Warning: failed to cleanup old backups: %v\n", err)
	}

	return objName, nil
}

// cleanupOldBackups removes old backups based on the keepLast setting
func (s *S3Client) cleanupOldBackups(ctx context.Context, dbName, dbType string) error {
	// List all objects with the prefix
	prefix := fmt.Sprintf("%s/%s-%s-", s.prefix, dbName, dbType)

	objectCh := s.client.ListObjects(ctx, s.bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	// Collect all backup objects
	var backups []minio.ObjectInfo
	for object := range objectCh {
		if object.Err != nil {
			return fmt.Errorf("error listing objects: %w", object.Err)
		}
		backups = append(backups, object)
	}

	// Sort backups by last modified time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].LastModified.After(backups[j].LastModified)
	})

	// Keep only the latest N backups
	if len(backups) > s.keepLast {
		for i := s.keepLast; i < len(backups); i++ {
			objName := backups[i].Key
			err := s.client.RemoveObject(ctx, s.bucketName, objName, minio.RemoveObjectOptions{})
			if err != nil {
				return fmt.Errorf("failed to remove old backup %s: %w", objName, err)
			}
			fmt.Printf("Removed old backup: %s\n", objName)
		}
	}

	return nil
}

// ListBackups lists all backups in the bucket with the given prefix
func (s *S3Client) ListBackups(ctx context.Context) ([]string, error) {
	objectCh := s.client.ListObjects(ctx, s.bucketName, minio.ListObjectsOptions{
		Prefix:    s.prefix,
		Recursive: true,
	})

	var backups []string
	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("error listing objects: %w", object.Err)
		}
		backups = append(backups, object.Key)
	}

	return backups, nil
}
