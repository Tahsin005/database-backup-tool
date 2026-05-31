package backup

import (
	"compress/gzip"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/Tahsin005/database-backup-tool/internal/db"
)

// RunBackup calls pg_dump, compresses the output, saves to current directory
func RunBackup(pg *db.Postgres) error {
	// build filename with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("backup_%s_%s.sql.gz", pg.DBName, timestamp)

	// create the output file
	outFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer outFile.Close()

	// wrap with gzip writer
	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	// run pg_dump and pipe output into gzip
	cmd := exec.Command("pg_dump",
		"-h", pg.Host,
		"-p", fmt.Sprintf("%d", pg.Port),
		"-U", pg.Username,
		"-d", pg.DBName,
		"-F", "p", // plain SQL format
	)

	// Pass password via environment variable (safer than CLI arg)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", pg.Password))

	// Pipe pg_dump stdout → gzip writer → file
	cmd.Stdout = gzWriter
	cmd.Stderr = os.Stderr // print pg_dump errors to terminal

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump failed: %w", err)
	}

	fmt.Printf("[%s] Backup saved: %s\n", time.Now().Format("15:04:05"), filename)
	return nil
}

// StartScheduler runs a backup every 1 minute, blocks forever
func StartScheduler(pg *db.Postgres) {
	// run once immediately
	if err := RunBackup(pg); err != nil {
		fmt.Printf("Backup error: %v\n", err)
	}

	// run every 1 minute
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if err := RunBackup(pg); err != nil {
			fmt.Printf("Backup error: %v\n", err)
		}
	}
}