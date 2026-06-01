package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Tahsin005/database-backup-tool/internal/config"
	"github.com/Tahsin005/database-backup-tool/internal/db"
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

	// Check if name already exists
	exists, err := config.ProfileExists(name)
	if err != nil {
		fmt.Printf("Error checking config: %v\n", err)
		os.Exit(1)
	}
	if exists {
		fmt.Printf("Error: a profile named %q already exists\n", name)
		os.Exit(1)
	}

	// Step 2: DB type (only postgres for now)
	fmt.Println("Database type:")
	fmt.Println("  [1] PostgreSQL")
	dbTypeInput := prompt(reader, "Choose (1): ")
	if dbTypeInput == "" {
		dbTypeInput = "1" // default to postgres
	}
	if dbTypeInput != "1" {
		fmt.Println("Error: only PostgreSQL is supported right now")
		os.Exit(1)
	}
	dbType := "postgres"

	// connection details
	host := promptWithDefault(reader, "Host", "localhost")

	portStr := promptWithDefault(reader, "Port", "5432")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		fmt.Println("Error: port must be a number")
		os.Exit(1)
	}

	username := prompt(reader, "Username: ")
	if username == "" {
		fmt.Println("Error: username cannot be empty")
		os.Exit(1)
	}

	password := prompt(reader, "Password: ")
	if password == "" {
		fmt.Println("Error: password cannot be empty")
		os.Exit(1)
	}

	dbName := prompt(reader, "Database name: ")
	if dbName == "" {
		fmt.Println("Error: database name cannot be empty")
		os.Exit(1)
	}

	// test the connection before saving
	fmt.Println()
	fmt.Println("Testing connection...")

	pg := db.NewPostgres(host, port, username, password, dbName)
	if err := pg.Ping(); err != nil {
		fmt.Printf("Connection failed: %v\n", err)
		fmt.Println("Profile not saved. Please check your credentials and try again.")
		os.Exit(1)
	}

	fmt.Println("Connection successful!")

	// save the profile
	profile := config.DBProfile{
		Name:     name,
		Type:     dbType,
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		DBName:   dbName,
	}

	if err := config.SaveProfile(profile); err != nil {
		fmt.Printf("Error saving profile: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nProfile %q saved successfully!\n", name)
	fmt.Printf("Run \"backuptool start %s\" to start backing up.\n", name)
}

// prints a label and reads a line from stdin
func prompt(reader *bufio.Reader, label string) string {
	fmt.Print(label)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// shows a default value and uses it if user hits enter
func promptWithDefault(reader *bufio.Reader, label, defaultVal string) string {
	fmt.Printf("%s [%s]: ", label, defaultVal)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}