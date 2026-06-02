package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Tahsin005/database-backup-tool/internal/config"
	"github.com/Tahsin005/database-backup-tool/internal/db"
)

// runs the ping loop
func StartMonitor(profile config.MonitorProfile) {
	logFile, err := openMonitorLogFile(profile.DBName)
	if err != nil {
		os.Exit(1)
	}
	defer logFile.Close()

	fmt.Fprintf(logFile, "[%s] Monitor started for %q. Interval: %d min.\n",
		now(), profile.Name, profile.MonitorInterval)

	// track previous state so we know when DB recovers
	wasDown := false

	// ping immediately on start
	wasDown = runPing(profile, wasDown, logFile)

	ticker := time.NewTicker(time.Duration(profile.MonitorInterval) * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		wasDown = runPing(profile, wasDown, logFile)
	}
}

// performs one ping and handles alerting
// returns the new "wasDown" state
func runPing(profile config.MonitorProfile, wasDown bool, logFile *os.File) bool {
	pg := db.NewPostgres(
		profile.Host,
		profile.Port,
		profile.Username,
		profile.Password,
		profile.DBName,
	)

	err := pg.Ping()

	if err != nil {
		// DB is down
		fmt.Fprintf(logFile, "[%s] PING FAILED: %v\n", now(), err)

		if !wasDown {
			// first failure — send down alert
			msg := fmt.Sprintf(
				"🔴 **Database Down**\nProfile: `%s`\nDatabase: `%s` @ `%s:%d`\nError: %s\nTime: %s",
				profile.Name, profile.DBName, profile.Host, profile.Port,
				err.Error(), time.Now().Format("2006-01-02 15:04:05"),
			)
			sendDiscordAlert(profile.WebhookURL, msg, logFile)
		}

		return true // wasDown = true
	}

	// DB is up
	fmt.Fprintf(logFile, "[%s] PING OK\n", now())

	if wasDown {
		// DB just recovered — send recovery alert
		msg := fmt.Sprintf(
			"🟢 **Database Recovered**\nProfile: `%s`\nDatabase: `%s` @ `%s:%d`\nTime: %s",
			profile.Name, profile.DBName, profile.Host, profile.Port,
			time.Now().Format("2006-01-02 15:04:05"),
		)
		sendDiscordAlert(profile.WebhookURL, msg, logFile)
	}

	return false // wasDown = false
}

// POSTs a message to the Discord webhook URL
func sendDiscordAlert(webhookURL string, message string, logFile *os.File) {
	payload := map[string]string{
		"content": message,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintf(logFile, "[%s] Failed to build Discord payload: %v\n", now(), err)
		return
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Fprintf(logFile, "[%s] Failed to send Discord alert: %v\n", now(), err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Fprintf(logFile, "[%s] Discord alert sent.\n", now())
	} else {
		fmt.Fprintf(logFile, "[%s] Discord alert failed. Status: %d\n", now(), resp.StatusCode)
	}
}

func openMonitorLogFile(dbName string) (*os.File, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return nil, err
	}
	logPath := filepath.Join(dir, dbName+".monitor.log")
	return os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
}

func now() string {
	return time.Now().Format("15:04:05")
}