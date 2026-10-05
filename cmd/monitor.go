package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/Tahsin005/database-backup-tool/internal/config"
	"github.com/Tahsin005/database-backup-tool/internal/daemon"
	"github.com/Tahsin005/database-backup-tool/internal/db"
	"github.com/Tahsin005/database-backup-tool/internal/monitor"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Manage database monitoring",
}

func init() {
	rootCmd.AddCommand(monitorCmd)
	monitorCmd.AddCommand(monitorAddCmd)
	monitorCmd.AddCommand(monitorStartCmd)
	monitorCmd.AddCommand(monitorStopCmd)
	monitorCmd.AddCommand(monitorStatusCmd)
	monitorCmd.AddCommand(monitorRemoveCmd)

	monitorStartCmd.Flags().BoolVar(&monitorDaemonMode, "daemon", false, "")
	monitorStartCmd.Flags().MarkHidden("daemon")
}

// monitor add
var monitorAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new monitor profile",
	Args:  cobra.NoArgs,
	RunE:  runMonitorAdd,
}

func runMonitorAdd(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== Add Monitor Profile ===")
	fmt.Println()

	// ask if user wants to import from existing backup profile
	backupProfiles, _ := config.LoadAllProfiles()

	var profile config.MonitorProfile

	if len(backupProfiles) > 0 {
		fmt.Print("Import connection details from an existing backup profile? (yes/no): ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))

		if answer == "yes" {
			// list available backup profiles
			fmt.Println("\nAvailable backup profiles:")
			names := make([]string, 0, len(backupProfiles))
			for name := range backupProfiles {
				names = append(names, name)
			}
			for i, name := range names {
				p := backupProfiles[name]
				fmt.Printf("  [%d] %-20s (%s @ %s)\n", i+1, name, p.DBName, p.Host)
			}

			fmt.Print("\nChoose a number: ")
			choiceStr, _ := reader.ReadString('\n')
			choiceStr = strings.TrimSpace(choiceStr)
			choice, err := strconv.Atoi(choiceStr)

			if err == nil && choice >= 1 && choice <= len(names) {
				chosen := backupProfiles[names[choice-1]]

				// copy connection details over
				profile = config.MonitorProfile{
					Name:         chosen.Name,
					DBConnConfig: chosen.DBConnConfig,
				}
				fmt.Printf("\nImported connection details from %q.\n\n", chosen.Name)
			} else {
				fmt.Println("Invalid choice. Switching to manual entry.")
				var err error
				profile, err = collectMonitorConnectionDetails(reader)
				if err != nil {
					return err
				}
			}
		} else {
			var err error
			profile, err = collectMonitorConnectionDetails(reader)
			if err != nil {
				return err
			}
		}
	} else {
		var err error
		profile, err = collectMonitorConnectionDetails(reader)
		if err != nil {
			return err
		}
	}

	// check if monitor profile name already exists
	exists, _ := config.MonitorProfileExists(profile.Name)
	if exists {
		return fmt.Errorf("a monitor profile named %q already exists", profile.Name)
	}

	// monitor interval
	intervalStr := promptWithDefault(reader, "Monitor interval (minutes)", "5")
	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval < 1 {
		return fmt.Errorf("interval must be a positive number")
	}
	profile.MonitorInterval = interval

	// discord webhook URL
	webhookURL := prompt(reader, "Discord webhook URL: ")
	if webhookURL == "" {
		return fmt.Errorf("webhook URL cannot be empty")
	}
	profile.WebhookURL = webhookURL
	profile.Enabled = true

	// test connection before saving
	fmt.Println("\nTesting connection...")
	pg := db.NewPostgresFromConfig(profile.DBConnConfig)
	if err := pg.Ping(); err != nil {
		fmt.Printf("Connection failed: %v\n", err)
		return fmt.Errorf("profile not saved: connection test failed: %w", err)
	}
	fmt.Println("Connection successful!")

	if err := config.SaveMonitorProfile(profile); err != nil {
		return fmt.Errorf("saving monitor profile: %w", err)
	}

	fmt.Printf("\nMonitor profile %q saved.\n", profile.Name)
	fmt.Printf("Run \"backuptool monitor start %s\" to begin monitoring.\n", profile.Name)
	return nil
}

// runs the manual entry wizard
func collectMonitorConnectionDetails(reader *bufio.Reader) (config.MonitorProfile, error) {
	name := prompt(reader, "Profile name: ")
	if name == "" {
		return config.MonitorProfile{}, fmt.Errorf("profile name cannot be empty")
	}

	connCfg, err := promptDBConnection(reader)
	if err != nil {
		return config.MonitorProfile{}, err
	}

	return config.MonitorProfile{
		Name:         name,
		DBConnConfig: connCfg,
	}, nil
}

// monitor start

var monitorDaemonMode bool

var monitorStartCmd = &cobra.Command{
	Use:   "start <profile-name>",
	Short: "Start monitoring daemon for a profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runMonitorStart,
}

func runMonitorStart(cmd *cobra.Command, args []string) error {
	profileName := args[0]

	profile, err := config.LoadMonitorProfile(profileName)
	if err != nil {
		return err
	}

	if !profile.Enabled {
		return fmt.Errorf("monitor profile %q is disabled", profileName)
	}

	d := daemon.New(profileName, daemon.KindMonitor)

	if !monitorDaemonMode {
		// foreground check if already running
		if running, _ := d.IsRunning(); running {
			return fmt.Errorf("monitor daemon for %q is already running", profileName)
		}

		// re-launch as background daemon
		child, err := d.LaunchBackground("monitor", "start", profileName, "--daemon")
		if err != nil {
			return fmt.Errorf("starting monitor daemon: %w", err)
		}

		fmt.Printf("Monitor daemon started for %q (PID: %d)\n", profileName, child.Process.Pid)
		fmt.Println("Run \"backuptool monitor status\" to check its state.")
		return nil
	}

	// daemon mode: write pid and start loop
	if err := d.WritePID(); err != nil {
		return fmt.Errorf("writing PID: %w", err)
	}
	defer d.DeletePID()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	monitor.StartMonitor(ctx, profile)
	return nil
}

// monitor stop

var monitorStopCmd = &cobra.Command{
	Use:   "stop <profile-name>",
	Short: "Stop the monitor daemon for a profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runMonitorStop,
}

func runMonitorStop(cmd *cobra.Command, args []string) error {
	profileName := args[0]

	d := daemon.New(profileName, daemon.KindMonitor)
	if running, _ := d.IsRunning(); !running {
		fmt.Printf("Monitor daemon for %q is not running.\n", profileName)
		return nil
	}

	if err := d.Stop(5 * time.Second); err != nil {
		return fmt.Errorf("stopping monitor daemon: %w", err)
	}

	fmt.Printf("Monitor daemon for %q stopped.\n", profileName)
	return nil
}

// monitor status

var monitorStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show monitor daemon state for all profiles",
	Args:  cobra.NoArgs,
	RunE:  runMonitorStatus,
}

func runMonitorStatus(cmd *cobra.Command, args []string) error {
	profiles, err := config.LoadAllMonitorProfiles()
	if err != nil {
		return err
	}

	if len(profiles) == 0 {
		fmt.Println("No monitor profiles configured yet.")
		fmt.Println("Run \"backuptool monitor add\" to add one.")
		return nil
	}

	fmt.Println()

	for _, p := range profiles {
		fmt.Printf("Profile  : %s\n", p.Name)
		fmt.Printf("Database : %s (%s)\n", p.DBName, p.Type)
		fmt.Printf("Host     : %s:%d\n", p.Host, p.Port)
		fmt.Printf("Interval : every %d min\n", p.MonitorInterval)
		fmt.Printf("Webhook  : %s\n", p.WebhookURL)

		enabledStr := "yes"
		if !p.Enabled {
			enabledStr = "no"
		}
		fmt.Printf("Enabled  : %s\n", enabledStr)

		d := daemon.New(p.Name, daemon.KindMonitor)
		if running, pid := d.IsRunning(); running {
			fmt.Printf("Daemon   : running (PID: %d)\n", pid)
		} else {
			fmt.Printf("Daemon   : stopped\n")
		}

		// show last log entry
		logPath, _ := monitorLogPath(p.DBName)
		if _, err := os.Stat(logPath); err == nil {
			fmt.Printf("Log file : %s\n", logPath)
		}

		fmt.Println(strings.Repeat("-", 45))
	}

	fmt.Println()
	return nil
}

// monitor remove

var monitorRemoveCmd = &cobra.Command{
	Use:   "remove <profile-name>",
	Short: "Remove a monitor profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runMonitorRemove,
}

func runMonitorRemove(cmd *cobra.Command, args []string) error {
	profileName := args[0]
	reader := bufio.NewReader(os.Stdin)

	_, err := config.LoadMonitorProfile(profileName)
	if err != nil {
		return err
	}

	d := daemon.New(profileName, daemon.KindMonitor)
	if running, _ := d.IsRunning(); running {
		return fmt.Errorf("monitor daemon for %q is still running; run \"backuptool monitor stop %s\" first", profileName, profileName)
	}

	fmt.Printf("Are you sure you want to remove monitor profile %q? (y/n): ", profileName)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer != "y" {
		fmt.Println("Cancelled.")
		return nil
	}

	if err := config.RemoveMonitorProfile(profileName); err != nil {
		return fmt.Errorf("removing monitor profile: %w", err)
	}

	fmt.Printf("Monitor profile %q removed.\n", profileName)
	return nil
}

func monitorLogPath(dbName string) (string, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, dbName+".monitor.log"), nil
}