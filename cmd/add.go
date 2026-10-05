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
	Run:   runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== Add New Database Profile ===")
	fmt.Println()

	// profile name
	name := prompt(reader, "Profile name (e.g. my-local-pg): ")
	if name == "" {
		fmt.Println("Error: profile name cannot be empty")
		os.Exit(1)
	}

	// check if name already exists
	exists, err := config.ProfileExists(name)
	if err != nil {
		fmt.Printf("Error checking config: %v\n", err)
		os.Exit(1)
	}
	if exists {
		fmt.Printf("Error: a profile named %q already exists\n", name)
		os.Exit(1)
	}

	// connection details
	connCfg := promptDBConnection(reader)

	// storage type
	fmt.Println("Storage type:")
	fmt.Println("  [1] Local")
	storageInput := promptWithDefault(reader, "Choose", "1")
	if storageInput != "1" {
		fmt.Println("Error: only local storage is supported right now")
		os.Exit(1)
	}
	storage := "local"

	// backup directory
	defaultDir := filepath.Join(os.Getenv("HOME"), "backups")
	backupDir := promptWithDefault(reader, "Backup directory", defaultDir)

	// backup interval
	intervalStr := promptWithDefault(reader, "Backup interval (minutes)", "60")
	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval < 1 {
		fmt.Println("Error: interval must be a positive number")
		os.Exit(1)
	}

	// test the connection before saving
	fmt.Println()
	fmt.Println("Testing connection...")

	pg := db.NewPostgresFromConfig(connCfg)
	if err := pg.Ping(); err != nil {
		fmt.Printf("Connection failed: %v\n", err)
		fmt.Println("Profile not saved. Please check your credentials and try again.")
		os.Exit(1)
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
		fmt.Printf("Error saving profile: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nProfile %q saved successfully!\n", name)
	fmt.Printf("Run \"backuptool start %s\" to start backing up.\n", name)
}