package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/Tahsin005/database-backup-tool/internal/config"
	"github.com/Tahsin005/database-backup-tool/internal/daemon"
)

var stopCmd = &cobra.Command{
	Use:   "stop <profile-name>",
	Short: "Stop the running backup daemon for a profile",
	Args:  cobra.ExactArgs(1),
	Run:   runStop,
}

func init() {
	rootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, args []string) {
	profileName := args[0]

	_, err := config.LoadProfile(profileName)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	d := daemon.New(profileName, daemon.KindBackup)
	if running, _ := d.IsRunning(); !running {
		fmt.Printf("Daemon for %q is not running.\n", profileName)
		os.Exit(0)
	}

	if err := d.Stop(5 * time.Second); err != nil {
		fmt.Printf("Error stopping daemon: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Backup daemon for %q stopped.\n", profileName)
}