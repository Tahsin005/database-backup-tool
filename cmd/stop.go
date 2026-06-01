package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/Tahsin005/database-backup-tool/internal/config"
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

	if !isAlreadyRunning(profileName) {
		fmt.Printf("Daemon for %q is not running.\n", profileName)
		os.Exit(0)
	}

	if err := stopDaemon(profileName); err != nil {
		fmt.Printf("Error stopping daemon: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Backup daemon for %q stopped.\n", profileName)
}