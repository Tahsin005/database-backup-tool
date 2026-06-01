package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/Tahsin005/database-backup-tool/internal/config"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured database profiles",
	Args:  cobra.NoArgs,
	Run:   runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) {
	profiles, err := config.LoadAllProfiles()
	if err != nil {
		fmt.Printf("Error loading profiles: %v\n", err)
		os.Exit(1)
	}

	if len(profiles) == 0 {
		fmt.Println("No profiles configured yet.")
		fmt.Println("Run \"backuptool add\" to add one.")
		return
	}

	// header
	fmt.Printf("\n%-20s %-12s %-16s %-6s %-16s %-10s\n",
    	"NAME", "TYPE", "HOST", "PORT", "DATABASE", "ENABLED")
	fmt.Println("-----------------------------------------------------------------------------")
	
	for _, p := range profiles {
		// check if daemon is running for this profile
		status := "stopped"
		if isAlreadyRunning(p.Name) {
			status = "running"
		}

		enabled := "yes"
		if !p.Enabled {
		    enabled = "no"
		}
		
		fmt.Printf("%-20s %-12s %-16s %-6d %-16s %-10s [%s]\n",
		    p.Name,
		    p.Type,
		    p.Host,
		    p.Port,
		    p.DBName,
		    enabled,
		    status,
		)
	}

	fmt.Println()
}