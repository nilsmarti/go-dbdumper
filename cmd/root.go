package cmd

import (
	"fmt"
	"os"

	"github.com/nilsmarti/go-dbdumper/backup"
	"github.com/nilsmarti/go-dbdumper/config"
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

		// Initialize scheduler
		scheduler := scheduler.New(cfg.CronExpression, backupSvc.PerformBackup)

		// Start the scheduler
		if err := scheduler.Start(); err != nil {
			fmt.Printf("Error starting scheduler: %v\n", err)
			os.Exit(1)
		}
		defer scheduler.Stop()

		fmt.Printf("DB Dumper started with cron expression: %s\n", cfg.CronExpression)
		fmt.Println("Press Ctrl+C to exit.")

		// Wait for interrupt signal
		select {}
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

		// Perform backup
		if err := backupSvc.PerformBackup(); err != nil {
			fmt.Printf("Error performing backup: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Backup completed successfully.")
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
}
