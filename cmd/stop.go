package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

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

	// make sure the profile actually exists in config
	_, err := config.LoadProfile(profileName)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// find the PID file
	pidPath, err := pidFilePath(profileName)
	if err != nil {
		fmt.Printf("Error resolving PID file path: %v\n", err)
		os.Exit(1)
	}

	// read the PID
	data, err := os.ReadFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Daemon for %q is not running.\n", profileName)
			os.Exit(0)
		}
		fmt.Printf("Error reading PID file: %v\n", err)
		os.Exit(1)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		fmt.Printf("Error: PID file is corrupted\n")
		os.Remove(pidPath) // clean it up
		os.Exit(1)
	}

	// find the process and kill it
	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Printf("Error finding process %d: %v\n", pid, err)
		os.Remove(pidPath)
		os.Exit(1)
	}

	if err := process.Kill(); err != nil {
		fmt.Printf("Error killing process %d: %v\n", pid, err)
		os.Exit(1)
	}

	// clean up the PID file
	os.Remove(pidPath)

	fmt.Printf("Backup daemon for %q stopped (PID: %d)\n", profileName, pid)
}