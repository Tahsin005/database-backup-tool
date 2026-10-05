package backup

import (
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
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
	fullPath := filepath.Join(backupDir, filename)
	tmpPath := fullPath + ".tmp"

	outFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temporary backup file: %w", err)
	}

	gzWriter, err := gzip.NewWriterLevel(outFile, gzip.BestCompression)
	if err != nil {
		_ = outFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to create gzip writer: %w", err)
	}

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

	dumpErr := cmd.Run()

	// explicitly close writer and file to flush trailers and release descriptors
	gzErr := gzWriter.Close()
	fileErr := outFile.Close()

	if dumpErr != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("pg_dump failed: %w", dumpErr)
	}
	if gzErr != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("gzip flush failed: %w", gzErr)
	}
	if fileErr != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("closing backup file failed: %w", fileErr)
	}

	// atomically finalize the backup
	if err := os.Rename(tmpPath, fullPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to finalize backup file: %w", err)
	}

	fmt.Fprintf(logger, "[%s] Backup saved: %s\n", time.Now().Format("15:04:05"), fullPath)
	return nil
}

func StartScheduler(ctx context.Context, pg *db.Postgres, backupDir string, intervalMinutes int) {
	logFile, err := openLogFile(pg.DBName)
	if err != nil {
		return
	}
	defer logFile.Close()

	fmt.Fprintf(logFile, "[%s] Scheduler started. Interval: %d min. Dir: %s\n",
		time.Now().Format("15:04:05"), intervalMinutes, backupDir)

	var isRunning atomic.Bool

	runSafe := func() {
		if !isRunning.CompareAndSwap(false, true) {
			fmt.Fprintf(logFile, "[%s] Warning: previous backup is still in progress; skipping scheduled run\n",
				time.Now().Format("15:04:05"))
			return
		}
		defer isRunning.Store(false)

		if err := RunBackup(pg, backupDir, logFile); err != nil {
			fmt.Fprintf(logFile, "[%s] Backup error: %v\n", time.Now().Format("15:04:05"), err)
		}
	}

	// run once immediately
	runSafe()

	ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Fprintf(logFile, "[%s] Scheduler stopped gracefully.\n", time.Now().Format("15:04:05"))
			return
		case <-ticker.C:
			runSafe()
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