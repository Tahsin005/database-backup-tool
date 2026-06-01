package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Tahsin005/database-backup-tool/internal/config"
)

var removeCmd = &cobra.Command{
	Use:   "remove <profile-name>",
	Short: "Remove a database profile from config",
	Args:  cobra.ExactArgs(1),
	Run:   runRemove,
}

func init() {
	rootCmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) {
	profileName := args[0]
	reader := bufio.NewReader(os.Stdin)

	// make sure profile exists
	_, err := config.LoadProfile(profileName)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// refuse if daemon is running
	if isAlreadyRunning(profileName) {
		fmt.Printf("Error: daemon for %q is still running.\n", profileName)
		fmt.Printf("Run \"backuptool stop %s\" first.\n", profileName)
		os.Exit(1)
	}

	fmt.Printf("Are you sure you want to remove %q? (y/n): ", profileName)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer != "y" {
		fmt.Println("Cancelled.")
		os.Exit(0)
	}

	if err := config.RemoveProfile(profileName); err != nil {
		fmt.Printf("Error removing profile: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Profile %q removed.\n", profileName)
}