package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/Tahsin005/database-backup-tool/internal/backup"
	"github.com/Tahsin005/database-backup-tool/internal/config"
	"github.com/Tahsin005/database-backup-tool/internal/daemon"
	"github.com/Tahsin005/database-backup-tool/internal/db"
)

var daemonMode bool

var startCmd = &cobra.Command{
	Use:   "start <profile-name>",
	Short: "Start backup scheduler for a database profile",
	Args:  cobra.ExactArgs(1), // exactly one argument required
	Run:   runStart,
}

func init() {
	startCmd.Flags().BoolVar(&daemonMode, "daemon", false, "Run as background daemon (used internally)")
	startCmd.Flags().MarkHidden("daemon")
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) {
	profileName := args[0]

	// make sure the profile exists before doing anything
	profile, err := config.LoadProfile(profileName)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if !profile.Enabled {
		fmt.Printf("Error: profile %q is disabled.\n", profileName)
		fmt.Printf("Run \"backuptool edit %s\" to enable it.\n", profileName)
		os.Exit(1)
	}

	d := daemon.New(profileName, daemon.KindBackup)

	if !daemonMode {
		// foreground mode: re-launch as a background child process and exit
		if running, _ := d.IsRunning(); running {
			fmt.Printf("Backup daemon for %q is already running.\n", profileName)
			fmt.Printf("Run \"backuptool stop %s\" to stop it first.\n", profileName)
			os.Exit(1)
		}

		child, err := d.LaunchBackground("start", profileName, "--daemon")
		if err != nil {
			fmt.Printf("Failed to start daemon: %v\n", err)
			os.Exit(1)
		}

		// parent exits here — terminal is freed
		fmt.Printf("Backup daemon started for profile %q (PID: %d)\n", profileName, child.Process.Pid)
		fmt.Println("Run \"backuptool status\" to check its state.")
		os.Exit(0)
	}

	// daemon mode: background child execution
	if err := d.WritePID(); err != nil {
		fmt.Printf("Error writing PID: %v\n", err)
		os.Exit(1)
	}
	defer d.DeletePID()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// connect to the database
	pg := db.NewPostgresFromConfig(profile.DBConnConfig)

	// start the backup scheduler
	backup.StartScheduler(ctx, pg, profile.BackupDir, profile.Interval)
}