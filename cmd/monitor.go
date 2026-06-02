package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/Tahsin005/database-backup-tool/internal/config"
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
	Run:   runMonitorAdd,
}

func runMonitorAdd(cmd *cobra.Command, args []string) {
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
					Name:     chosen.Name,
					Type:     chosen.Type,
					Host:     chosen.Host,
					Port:     chosen.Port,
					Username: chosen.Username,
					Password: chosen.Password,
					DBName:   chosen.DBName,
				}
				fmt.Printf("\nImported connection details from %q.\n\n", chosen.Name)
			} else {
				fmt.Println("Invalid choice. Switching to manual entry.")
				profile = collectMonitorConnectionDetails(reader)
			}
		} else {
			profile = collectMonitorConnectionDetails(reader)
		}
	} else {
		profile = collectMonitorConnectionDetails(reader)
	}

	// check if monitor profile name already exists
	exists, _ := config.MonitorProfileExists(profile.Name)
	if exists {
		fmt.Printf("Error: a monitor profile named %q already exists.\n", profile.Name)
		os.Exit(1)
	}

	// monitor interval
	intervalStr := promptWithDefault(reader, "Monitor interval (minutes)", "5")
	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval < 1 {
		fmt.Println("Error: interval must be a positive number")
		os.Exit(1)
	}
	profile.MonitorInterval = interval

	// discord webhook URL
	webhookURL := prompt(reader, "Discord webhook URL: ")
	if webhookURL == "" {
		fmt.Println("Error: webhook URL cannot be empty")
		os.Exit(1)
	}
	profile.WebhookURL = webhookURL
	profile.Enabled = true

	// test connection before saving
	fmt.Println("\nTesting connection...")
	pg := db.NewPostgres(profile.Host, profile.Port, profile.Username, profile.Password, profile.DBName)
	if err := pg.Ping(); err != nil {
		fmt.Printf("Connection failed: %v\n", err)
		fmt.Println("Profile not saved.")
		os.Exit(1)
	}
	fmt.Println("Connection successful!")

	if err := config.SaveMonitorProfile(profile); err != nil {
		fmt.Printf("Error saving monitor profile: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nMonitor profile %q saved.\n", profile.Name)
	fmt.Printf("Run \"backuptool monitor start %s\" to begin monitoring.\n", profile.Name)
}

// runs the manual entry wizard
func collectMonitorConnectionDetails(reader *bufio.Reader) config.MonitorProfile {
	name := prompt(reader, "Profile name: ")
	if name == "" {
		fmt.Println("Error: name cannot be empty")
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
	password := prompt(reader, "Password: ")
	dbName := prompt(reader, "Database name: ")

	return config.MonitorProfile{
		Name:     name,
		Type:     "postgres",
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		DBName:   dbName,
	}
}

// monitor start

var monitorDaemonMode bool

var monitorStartCmd = &cobra.Command{
	Use:   "start <profile-name>",
	Short: "Start monitoring daemon for a profile",
	Args:  cobra.ExactArgs(1),
	Run:   runMonitorStart,
}

func runMonitorStart(cmd *cobra.Command, args []string) {
	profileName := args[0]

	profile, err := config.LoadMonitorProfile(profileName)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if !profile.Enabled {
		fmt.Printf("Error: monitor profile %q is disabled.\n", profileName)
		os.Exit(1)
	}

	if !monitorDaemonMode {
		// foreground — check if already running
		if isMonitorRunning(profileName) {
			fmt.Printf("Monitor daemon for %q is already running.\n", profileName)
			os.Exit(1)
		}

		// re-launch as background daemon
		self := os.Args[0]
		child := exec.Command(self, "monitor", "start", profileName, "--daemon")
		child.Stdout = nil
		child.Stderr = nil
		child.Stdin = nil

		if err := child.Start(); err != nil {
			fmt.Printf("Failed to start monitor daemon: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Monitor daemon started for %q (PID: %d)\n", profileName, child.Process.Pid)
		fmt.Println("Run \"backuptool monitor status\" to check its state.")
		os.Exit(0)
	}

	// daemon mode — write PID and start loop
	if err := writeMonitorPIDFile(profileName); err != nil {
		os.Exit(1)
	}
	defer deleteMonitorPIDFile(profileName)

	monitor.StartMonitor(profile)
}

// monitor stop

var monitorStopCmd = &cobra.Command{
	Use:   "stop <profile-name>",
	Short: "Stop the monitor daemon for a profile",
	Args:  cobra.ExactArgs(1),
	Run:   runMonitorStop,
}

func runMonitorStop(cmd *cobra.Command, args []string) {
	profileName := args[0]

	if !isMonitorRunning(profileName) {
		fmt.Printf("Monitor daemon for %q is not running.\n", profileName)
		os.Exit(0)
	}

	if err := stopMonitorDaemon(profileName); err != nil {
		fmt.Printf("Error stopping monitor daemon: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Monitor daemon for %q stopped.\n", profileName)
}

// monitor status

var monitorStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show monitor daemon state for all profiles",
	Args:  cobra.NoArgs,
	Run:   runMonitorStatus,
}

func runMonitorStatus(cmd *cobra.Command, args []string) {
	profiles, err := config.LoadAllMonitorProfiles()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if len(profiles) == 0 {
		fmt.Println("No monitor profiles configured yet.")
		fmt.Println("Run \"backuptool monitor add\" to add one.")
		return
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

		if isMonitorRunning(p.Name) {
			pid := readMonitorPID(p.Name)
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
}

// monitor remove

var monitorRemoveCmd = &cobra.Command{
	Use:   "remove <profile-name>",
	Short: "Remove a monitor profile",
	Args:  cobra.ExactArgs(1),
	Run:   runMonitorRemove,
}

func runMonitorRemove(cmd *cobra.Command, args []string) {
	profileName := args[0]
	reader := bufio.NewReader(os.Stdin)

	_, err := config.LoadMonitorProfile(profileName)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if isMonitorRunning(profileName) {
		fmt.Printf("Error: monitor daemon for %q is still running.\n", profileName)
		fmt.Printf("Run \"backuptool monitor stop %s\" first.\n", profileName)
		os.Exit(1)
	}

	fmt.Printf("Are you sure you want to remove monitor profile %q? (y/n): ", profileName)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer != "y" {
		fmt.Println("Cancelled.")
		os.Exit(0)
	}

	if err := config.RemoveMonitorProfile(profileName); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Monitor profile %q removed.\n", profileName)
}

// Monitor PID file helpers

func monitorPIDFilePath(profileName string) (string, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, profileName+".monitor.pid"), nil
}

func writeMonitorPIDFile(profileName string) error {
	path, err := monitorPIDFilePath(profileName)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0600)
}

func deleteMonitorPIDFile(profileName string) {
	path, _ := monitorPIDFilePath(profileName)
	os.Remove(path)
}

func isMonitorRunning(profileName string) bool {
	path, err := monitorPIDFilePath(profileName)
	if err != nil {
		return false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	return process.Signal(syscall.Signal(0)) == nil
}

func readMonitorPID(profileName string) int {
	path, _ := monitorPIDFilePath(profileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	return pid
}

func stopMonitorDaemon(profileName string) error {
	path, err := monitorPIDFilePath(profileName)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		os.Remove(path)
		return fmt.Errorf("PID file corrupted")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	if err := process.Kill(); err != nil {
		return err
	}

	os.Remove(path)
	return nil
}

func monitorLogPath(dbName string) (string, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, dbName+".monitor.log"), nil
}