package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nilsmarti/go-dbdumper/backup"
	"github.com/nilsmarti/go-dbdumper/config"
	"github.com/nilsmarti/go-dbdumper/restore"
	"github.com/nilsmarti/go-dbdumper/scheduler"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "go-dbdumper",
	Short: "A tool to backup databases to S3",
	Long: `go-dbdumper is a tool that creates database dumps and uploads them directly to S3 compatible storage.

It supports both MySQL and PostgreSQL databases and can be configured via environment variables.`,
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the backup scheduler",
	Long:  `Run the backup scheduler which will perform backups according to the configured cron schedule.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
			os.Exit(1)
		}

		// Initialize backup service
		backupSvc, err := backup.NewService(cfg)
		if err != nil {
			fmt.Printf("Error initializing backup service: %v\n", err)
			os.Exit(1)
		}

		// Set up a context that is cancelled on SIGINT/SIGTERM so that
		// in-flight backups are aborted cleanly and the scheduler shuts down
		// orderly instead of being killed mid-stream.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		// Initialize scheduler
		scheduler := scheduler.New(cfg.CronExpression, backupSvc.PerformBackup)

		// Start the scheduler
		if err := scheduler.Start(ctx); err != nil {
			fmt.Printf("Error starting scheduler: %v\n", err)
			os.Exit(1)
		}
		defer scheduler.Stop()

		fmt.Printf("DB Dumper started with cron expression: %s\n", cfg.CronExpression)
		fmt.Println("Press Ctrl+C to exit.")

		// Wait for an interrupt signal
		<-ctx.Done()
		fmt.Println("Shutdown signal received, stopping...")
	},
}

var backupNowCmd = &cobra.Command{
	Use:   "backup-now",
	Short: "Run a backup immediately",
	Long:  `Run a backup immediately without waiting for the scheduled time.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
			os.Exit(1)
		}

		// Initialize backup service
		backupSvc, err := backup.NewService(cfg)
		if err != nil {
			fmt.Printf("Error initializing backup service: %v\n", err)
			os.Exit(1)
		}

		// Allow Ctrl+C to abort a manual backup.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		// Perform backup
		if err := backupSvc.PerformBackup(ctx); err != nil {
			fmt.Printf("Error performing backup: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Backup completed successfully.")
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore a backup from S3 to a database",
	Long: `Download a backup from S3, decrypt it if needed, and stream it into the
target database. By default the latest backup for the configured database is
restored; use --object to restore a specific backup object. Use --dry-run to
verify a backup (decrypt and print to stdout) without modifying a database.

This command is intended to run on a dedicated restore host that has the age
private key (DECRYPTION_PRIVATE_KEY or DECRYPTION_PRIVATE_KEY_FILE). The
private key should never be present on the app server that creates backups.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
			os.Exit(1)
		}

		// Initialize restore service
		restoreSvc, err := restore.NewService(cfg)
		if err != nil {
			fmt.Printf("Error initializing restore service: %v\n", err)
			os.Exit(1)
		}

		// Allow Ctrl+C to abort a restore.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		objectName, _ := cmd.Flags().GetString("object")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		if err := restoreSvc.Restore(ctx, restore.RestoreOptions{
			ObjectName: objectName,
			DryRun:     dryRun,
		}); err != nil {
			fmt.Printf("Error restoring backup: %v\n", err)
			os.Exit(1)
		}
	},
}

var listBackupsCmd = &cobra.Command{
	Use:   "list-backups",
	Short: "List available backups in S3",
	Long:  `List all backup objects in the configured S3 bucket and prefix.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
			os.Exit(1)
		}

		// Initialize restore service (reuses the S3 client)
		restoreSvc, err := restore.NewService(cfg)
		if err != nil {
			fmt.Printf("Error initializing service: %v\n", err)
			os.Exit(1)
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		backups, err := restoreSvc.ListBackups(ctx)
		if err != nil {
			fmt.Printf("Error listing backups: %v\n", err)
			os.Exit(1)
		}

		if len(backups) == 0 {
			fmt.Println("No backups found.")
			return
		}

		fmt.Printf("Found %d backup(s):\n", len(backups))
		for _, b := range backups {
			fmt.Printf("  %s\n", b)
		}
	},
}

// Execute executes the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(backupNowCmd)
	rootCmd.AddCommand(restoreCmd)
	rootCmd.AddCommand(listBackupsCmd)

	restoreCmd.Flags().String("object", "", "Specific S3 object name to restore (default: latest)")
	restoreCmd.Flags().Bool("dry-run", false, "Download and decrypt only; print plaintext to stdout without modifying the database")
}
