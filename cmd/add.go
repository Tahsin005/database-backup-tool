package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Tahsin005/database-backup-tool/internal/config"
	"github.com/Tahsin005/database-backup-tool/internal/db"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new database profile interactively",
	RunE:  runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== Add New Database Profile ===")
	fmt.Println()

	// profile name
	name := prompt(reader, "Profile name (e.g. my-local-pg): ")
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}

	// check if name already exists
	exists, err := config.ProfileExists(name)
	if err != nil {
		return fmt.Errorf("error checking config: %w", err)
	}
	if exists {
		return fmt.Errorf("a profile named %q already exists", name)
	}

	// connection details
	connCfg, err := promptDBConnection(reader)
	if err != nil {
		return err
	}

	// storage type
	fmt.Println("Storage type:")
	fmt.Println("  [1] Local")
	storageInput := promptWithDefault(reader, "Choose", "1")
	if storageInput != "1" {
		return fmt.Errorf("only local storage is supported right now")
	}
	storage := "local"

	// backup directory
	defaultDir := filepath.Join(os.Getenv("HOME"), "backups")
	backupDir := promptWithDefault(reader, "Backup directory", defaultDir)

	// backup interval
	intervalStr := promptWithDefault(reader, "Backup interval (minutes)", "60")
	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval < 1 {
		return fmt.Errorf("interval must be a positive number")
	}

	// test the connection before saving
	fmt.Println()
	fmt.Println("Testing connection...")

	pg := db.NewPostgresFromConfig(connCfg)
	if err := pg.Ping(); err != nil {
		return fmt.Errorf("connection failed: %w (profile not saved, please check credentials)", err)
	}

	fmt.Println("Connection successful!")

	// save the profile
	profile := config.DBProfile{
		Name:         name,
		DBConnConfig: connCfg,
		Storage:      storage,
		BackupDir:    backupDir,
		Interval:     interval,
		Enabled:      true,
	}

	if err := config.SaveProfile(profile); err != nil {
		return fmt.Errorf("error saving profile: %w", err)
	}

	fmt.Printf("\nProfile %q saved successfully!\n", name)
	fmt.Printf("Run \"backuptool start %s\" to start backing up.\n", name)
	return nil
}