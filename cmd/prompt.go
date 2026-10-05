package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Tahsin005/database-backup-tool/internal/config"
)

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

// prompts the user interactively for database connection details
func promptDBConnection(reader *bufio.Reader) config.DBConnConfig {
	fmt.Println("Database type:")
	fmt.Println("  [1] PostgreSQL")
	dbTypeInput := prompt(reader, "Choose (1): ")
	if dbTypeInput == "" {
		dbTypeInput = "1"
	}
	if dbTypeInput != "1" {
		fmt.Println("Error: only PostgreSQL is supported right now")
		os.Exit(1)
	}

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

	sslMode := promptWithDefault(reader, "SSL mode (disable/require/verify-full)", "disable")

	return config.DBConnConfig{
		Type:     "postgres",
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		DBName:   dbName,
		SSLMode:  sslMode,
	}
}
