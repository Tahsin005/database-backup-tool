package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Tahsin005/database-backup-tool/internal/config"
	"github.com/Tahsin005/database-backup-tool/internal/daemon"
)

var removeCmd = &cobra.Command{
	Use:   "remove <profile-name>",
	Short: "Remove a database profile from config",
	Args:  cobra.ExactArgs(1),
	RunE:  runRemove,
}

func init() {
	rootCmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) error {
	profileName := args[0]
	reader := bufio.NewReader(os.Stdin)

	// make sure profile exists
	_, err := config.LoadProfile(profileName)
	if err != nil {
		return err
	}

	d := daemon.New(profileName, daemon.KindBackup)

	// refuse if daemon is running
	if running, _ := d.IsRunning(); running {
		return fmt.Errorf("daemon for %q is still running; run \"backuptool stop %s\" first", profileName, profileName)
	}

	fmt.Printf("Are you sure you want to remove %q? (y/n): ", profileName)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer != "y" {
		fmt.Println("Cancelled.")
		return nil
	}

	if err := config.RemoveProfile(profileName); err != nil {
		return fmt.Errorf("removing profile: %w", err)
	}

	fmt.Printf("Profile %q removed.\n", profileName)
	return nil
}