package backup

import (
	"compress/gzip"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Tahsin005/database-backup-tool/internal/config"
	"github.com/Tahsin005/database-backup-tool/internal/db"
)

func RunBackup(pg *db.Postgres, backupDir string, logger *os.File) error {
	// ensure backup directory exists
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("backup_%s_%s.sql.gz", pg.DBName, timestamp)
	fullPath := filepath.Join(backupDir, filename) // use backupDir

	outFile, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer outFile.Close()

	gzWriter, err := gzip.NewWriterLevel(outFile, gzip.BestCompression)
	if err != nil {
		return fmt.Errorf("failed to create gzip writer: %w", err)
	}
	defer gzWriter.Close()

	cmd := exec.Command("pg_dump",
		"-h", pg.Host,
		"-p", fmt.Sprintf("%d", pg.Port),
		"-U", pg.Username,
		"-d", pg.DBName,
		"-F", "p",
	)

	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", pg.Password))
	cmd.Stdout = gzWriter
	cmd.Stderr = logger

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump failed: %w", err)
	}

	fmt.Fprintf(logger, "[%s] Backup saved: %s\n", time.Now().Format("15:04:05"), fullPath)
	return nil
}

func StartScheduler(pg *db.Postgres, backupDir string, intervalMinutes int) {
	logFile, err := openLogFile(pg.DBName)
	if err != nil {
		os.Exit(1)
	}
	defer logFile.Close()

	fmt.Fprintf(logFile, "[%s] Scheduler started. Interval: %d min. Dir: %s\n",
		time.Now().Format("15:04:05"), intervalMinutes, backupDir)

	// run once immediately
	if err := RunBackup(pg, backupDir, logFile); err != nil {
		fmt.Fprintf(logFile, "[%s] Backup error: %v\n", time.Now().Format("15:04:05"), err)
	}

	ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if err := RunBackup(pg, backupDir, logFile); err != nil {
			fmt.Fprintf(logFile, "[%s] Backup error: %v\n", time.Now().Format("15:04:05"), err)
		}
	}
}

func openLogFile(dbName string) (*os.File, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return nil, err
	}
	logPath := filepath.Join(dir, dbName+".log")
	return os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
}