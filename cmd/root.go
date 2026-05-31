package cmd

import (
	"fmt"
	"os"

	"github.com/Tahsin005/database-backup-tool/internal/backup"
	"github.com/Tahsin005/database-backup-tool/internal/db"
	"github.com/spf13/cobra"
)

var (
	host string
	username string
	password string
	port int 
	dbName string
)

var rootCmd = &cobra.Command{
	Use: "backuptool",
	Short: "PostgreSQL backup utility",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Testing connection to PostgreSQL...")

		pg := db.NewPostgres(host, port, username, password, dbName)
		if err := pg.Ping(); err != nil {
			fmt.Printf("Connection failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Connection successful! Starting backup scheduler...")

		backup.StartScheduler(pg)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVar(&host, "host", "localhost", "Database host")
	rootCmd.Flags().StringVar(&username, "username", "", "Database username")
	rootCmd.Flags().StringVar(&password, "password", "", "Database password")
	rootCmd.Flags().StringVar(&dbName, "dbname", "", "Database name")
	rootCmd.Flags().IntVar(&port, "port", 5432, "Database port")

	rootCmd.MarkFlagRequired("username")
	rootCmd.MarkFlagRequired("password")
	rootCmd.MarkFlagRequired("dbname")
}