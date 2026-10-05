package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/Tahsin005/database-backup-tool/internal/config"
	"github.com/Tahsin005/database-backup-tool/internal/daemon"
)

var editCmd = &cobra.Command{
	Use:   "edit <profile-name>",
	Short: "Edit backup directory, interval, or enabled state of a profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runEdit,
}

func init() {
	rootCmd.AddCommand(editCmd)
}

func runEdit(cmd *cobra.Command, args []string) error {
	profileName := args[0]
	reader := bufio.NewReader(os.Stdin)

	// load the existing profile
	profile, err := config.LoadProfile(profileName)
	if err != nil {
		return fmt.Errorf("error loading profile: %w", err)
	}

	d := daemon.New(profileName, daemon.KindBackup)

	// if daemon is running, ask user if they want to stop it
	if running, _ := d.IsRunning(); running {
		fmt.Printf("Daemon for %q is currently running.\n", profileName)
		fmt.Print("Stop it and proceed with editing? (yes/no): ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))

		if answer != "yes" {
			fmt.Println("Edit cancelled.")
			return nil
		}

		// stop the daemon
		if err := d.Stop(5 * time.Second); err != nil {
			return fmt.Errorf("failed to stop daemon: %w", err)
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
		return fmt.Errorf("interval must be a positive number")
	}

	// enabled
	currentEnabledStr := "true"
	if !profile.Enabled {
		currentEnabledStr = "false"
	}
	newEnabledStr := promptWithDefault(reader, "Enabled (true/false)", currentEnabledStr)
	if newEnabledStr != "true" && newEnabledStr != "false" {
		return fmt.Errorf("enabled must be \"true\" or \"false\"")
	}
	newEnabled := newEnabledStr == "true"

	// apply changes
	profile.BackupDir = newBackupDir
	profile.Interval = newInterval
	profile.Enabled = newEnabled

	if err := config.SaveProfile(profile); err != nil {
		return fmt.Errorf("error saving profile: %w", err)
	}

	fmt.Printf("\nProfile %q updated successfully.\n", profileName)

	if newEnabled {
		fmt.Printf("Run \"backuptool start %s\" to restart the daemon.\n", profileName)
	} else {
		fmt.Printf("Profile is now disabled. Daemon will not start for this profile.\n")
	}

	return nil
}