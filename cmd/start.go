package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/Tahsin005/database-backup-tool/internal/backup"
	"github.com/Tahsin005/database-backup-tool/internal/config"
	"github.com/Tahsin005/database-backup-tool/internal/db"
)

var daemonMode bool

var startCmd = &cobra.Command{
	Use:   "start <profile-name>",
	Short: "Start backup scheduler for a database profile",
	Args:  cobra.ExactArgs(1), // exactly one argument required
	Run:   runStart,
}

func init() {
	startCmd.Flags().BoolVar(&daemonMode, "daemon", false, "Run as background daemon (used internally)")
	startCmd.Flags().MarkHidden("daemon")
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) {
	profileName := args[0]

	// make sure the profile exists before doing anything
	profile, err := config.LoadProfile(profileName)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if !daemonMode {
		// -------------------------------------------------------
		// FOREGROUND MODE
		// We are the parent. Re-launch ourselves as a background
		// child process, then exit so the terminal is freed.
		// -------------------------------------------------------

		// check if already running
		if isAlreadyRunning(profileName) {
			fmt.Printf("Backup daemon for %q is already running.\n", profileName)
			fmt.Printf("Run \"backuptool stop %s\" to stop it first.\n", profileName)
			os.Exit(1)
		}

		// build the child command: same binary, same args, plus --daemon
		self := os.Args[0] // path to the current executable
		childArgs := []string{"start", profileName, "--daemon"}

		child := exec.Command(self, childArgs...)

		// detach from terminal completely
		child.Stdout = nil
		child.Stderr = nil
		child.Stdin = nil

		if err := child.Start(); err != nil {
			fmt.Printf("Failed to start daemon: %v\n", err)
			os.Exit(1)
		}

		// parent exits here — terminal is freed
		fmt.Printf("Backup daemon started for profile %q (PID: %d)\n", profileName, child.Process.Pid)
		fmt.Println("Run \"backuptool status\" to check its state.")
		os.Exit(0)
	}

	// -------------------------------------------------------
	// DAEMON MODE
	// We are the background child. Do the real work.
	// -------------------------------------------------------

	// write our PID to ~/.backuptool/<profileName>.pid
	if err := writePIDFile(profileName); err != nil {
		os.Exit(1)
	}

	// clean up PID file when we exit
	defer deletePIDFile(profileName)

	// connect to the database
	pg := db.NewPostgres(
		profile.Host,
		profile.Port,
		profile.Username,
		profile.Password,
		profile.DBName,
	)

	// start the backup scheduler — this blocks forever
	backup.StartScheduler(pg, profile.BackupDir, profile.Interval)
}

// -------------------------------------------------------
// PID file helpers
// -------------------------------------------------------

func pidFilePath(profileName string) (string, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, profileName+".pid"), nil
}

func writePIDFile(profileName string) error {
	path, err := pidFilePath(profileName)
	if err != nil {
		return err
	}

	pid := os.Getpid()
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0600)
}

func deletePIDFile(profileName string) {
	path, _ := pidFilePath(profileName)
	os.Remove(path)
}

func isAlreadyRunning(profileName string) bool {
	path, err := pidFilePath(profileName)
	if err != nil {
		return false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Signal 0 = don't kill, just check if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}