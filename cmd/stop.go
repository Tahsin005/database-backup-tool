package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/Tahsin005/database-backup-tool/internal/config"
	"github.com/Tahsin005/database-backup-tool/internal/daemon"
)

var stopCmd = &cobra.Command{
	Use:   "stop <profile-name>",
	Short: "Stop the running backup daemon for a profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runStop,
}

func init() {
	rootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, args []string) error {
	profileName := args[0]

	_, err := config.LoadProfile(profileName)
	if err != nil {
		return fmt.Errorf("error loading profile: %w", err)
	}

	d := daemon.New(profileName, daemon.KindBackup)
	if running, _ := d.IsRunning(); !running {
		fmt.Printf("Daemon for %q is not running.\n", profileName)
		return nil
	}

	if err := d.Stop(5 * time.Second); err != nil {
		return fmt.Errorf("error stopping daemon: %w", err)
	}

	fmt.Printf("Backup daemon for %q stopped.\n", profileName)
	return nil
}