package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/Tahsin005/database-backup-tool/internal/config"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon state and last backup info for all profiles",
	Args:  cobra.NoArgs,
	Run:   runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) {
	profiles, err := config.LoadAllProfiles()
	if err != nil {
		fmt.Printf("Error loading profiles: %v\n", err)
		os.Exit(1)
	}

	if len(profiles) == 0 {
		fmt.Println("No profiles configured yet.")
		fmt.Println("Run \"backuptool add\" to add one.")
		return
	}

	fmt.Println()

	for _, p := range profiles {
		fmt.Printf("Profile  : %s\n", p.Name)
		fmt.Printf("Database : %s (%s)\n", p.DBName, p.Type)
		fmt.Printf("Host     : %s:%d\n", p.Host, p.Port)
		fmt.Printf("Storage    : %s\n", p.Storage)
		fmt.Printf("Backup dir : %s\n", p.BackupDir)
		fmt.Printf("Interval   : every %d min\n", p.Interval)

		enabled := "yes"
		if !p.Enabled {
			enabled = "no"
		}
		fmt.Printf("Enabled  : %s\n", enabled)

		// daemon status
		if isAlreadyRunning(p.Name) {
			pid := readPID(p.Name)
			fmt.Printf("Daemon   : running (PID: %d)\n", pid)
		} else {
			fmt.Printf("Daemon   : stopped\n")
		}

		// last backup info
		lastFile, lastTime, err := findLastBackup(p.DBName, p.BackupDir)
		if err != nil || lastFile == "" {
			fmt.Printf("Last backup: none yet\n")
		} else {
			ago := time.Since(lastTime)
			fmt.Printf("Last backup: %s (%s ago)\n", lastFile, formatDuration(ago))
		}

		// log file location
		logPath, _ := logFilePath(p.DBName)
		if _, err := os.Stat(logPath); err == nil {
			fmt.Printf("Log file : %s\n", logPath)
		}

		fmt.Println(strings.Repeat("-", 45))
	}

	fmt.Println()
}

// reads the PID from the PID file
func readPID(profileName string) int {
	path, err := pidFilePath(profileName)
	if err != nil {
		return 0
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	pid := 0
	fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &pid)
	return pid
}

// looks in the backup directory for the most recent
// backup file matching backup_<dbname>_*.sql.gz
func findLastBackup(dbName string, backupDir string) (string, time.Time, error) {
	pattern := filepath.Join(backupDir, fmt.Sprintf("backup_%s_*.sql.gz", dbName))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", time.Time{}, err
	}
	if len(matches) == 0 {
		return "", time.Time{}, nil
	}

	// sort alphabetically
	// (backup_testdb_2026-05-31_22-36-26.sql.gz) this gives us chronological order 
	sort.Strings(matches)
	latest := matches[len(matches)-1]

	info, err := os.Stat(latest)
	if err != nil {
		return "", time.Time{}, err
	}

	return latest, info.ModTime(), nil
}

// returns ~/.backuptool/<dbname>.log
func logFilePath(dbName string) (string, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, dbName+".log"), nil
}

// formatDuration converts a duration into a human readable string
// e.g. "2h 15m" or "45s"
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)

	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}