package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Tahsin005/database-backup-tool/internal/config"
)

var editCmd = &cobra.Command{
	Use:   "edit <profile-name>",
	Short: "Edit backup directory, interval, or enabled state of a profile",
	Args:  cobra.ExactArgs(1),
	Run:   runEdit,
}

func init() {
	rootCmd.AddCommand(editCmd)
}

func runEdit(cmd *cobra.Command, args []string) {
	profileName := args[0]
	reader := bufio.NewReader(os.Stdin)

	// load the existing profile
	profile, err := config.LoadProfile(profileName)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// if daemon is running, ask user if they want to stop it
	if isAlreadyRunning(profileName) {
		fmt.Printf("Daemon for %q is currently running.\n", profileName)
		fmt.Print("Stop it and proceed with editing? (yes/no): ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))

		if answer != "yes" {
			fmt.Println("Edit cancelled.")
			os.Exit(0)
		}

		// stop the daemon
		if err := stopDaemon(profileName); err != nil {
			fmt.Printf("Failed to stop daemon: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Daemon stopped.")
		fmt.Println()
	}

	fmt.Printf("=== Edit Profile \"%s\" ===\n", profileName)
	fmt.Println("Press Enter to keep the current value.")
	fmt.Println()

	// backup directory
	newBackupDir := promptWithDefault(reader, "Backup directory", profile.BackupDir)

	// interval
	newIntervalStr := promptWithDefault(reader, "Backup interval (minutes)", strconv.Itoa(profile.Interval))
	newInterval, err := strconv.Atoi(newIntervalStr)
	if err != nil || newInterval < 1 {
		fmt.Println("Error: interval must be a positive number")
		os.Exit(1)
	}

	// enabled
	currentEnabledStr := "true"
	if !profile.Enabled {
		currentEnabledStr = "false"
	}
	newEnabledStr := promptWithDefault(reader, "Enabled (true/false)", currentEnabledStr)
	if newEnabledStr != "true" && newEnabledStr != "false" {
		fmt.Println("Error: enabled must be \"true\" or \"false\"")
		os.Exit(1)
	}
	newEnabled := newEnabledStr == "true"

	// apply changes
	profile.BackupDir = newBackupDir
	profile.Interval = newInterval
	profile.Enabled = newEnabled

	if err := config.SaveProfile(profile); err != nil {
		fmt.Printf("Error saving profile: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nProfile %q updated successfully.\n", profileName)

	if newEnabled {
		fmt.Printf("Run \"backuptool start %s\" to restart the daemon.\n", profileName)
	} else {
		fmt.Printf("Profile is now disabled. Daemon will not start for this profile.\n")
	}
}

// shared stop logic used by both edit and stop commands
func stopDaemon(profileName string) error {
	pidPath, err := pidFilePath(profileName)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(pidPath)
	if err != nil {
		return err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		os.Remove(pidPath)
		return fmt.Errorf("PID file corrupted")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	if err := process.Kill(); err != nil {
		return err
	}

	os.Remove(pidPath)
	return nil
}