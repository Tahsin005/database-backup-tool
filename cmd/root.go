package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "backuptool",
	Short: "A CLI database backup utility",
	Long:  "backuptool lets you configure, schedule, and manage database backups from your terminal.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}