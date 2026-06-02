package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type MonitorProfile struct {
	Name            string
	Type            string
	Host            string
	Port            int
	Username        string
	Password        string
	DBName          string
	MonitorInterval int    // in minutes
	WebhookURL      string
	Enabled         bool
}

// returns ~/.backuptool/monitorsettings.conf
func MonitorConfigFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "monitorsettings.conf"), nil
}

// writes a monitor profile to monitorsettings.conf
func SaveMonitorProfile(profile MonitorProfile) error {
	if err := EnsureConfigDir(); err != nil {
		return err
	}

	existing, err := LoadAllMonitorProfiles()
	if err != nil {
		existing = map[string]MonitorProfile{}
	}

	existing[profile.Name] = profile
	return writeAllMonitorProfiles(existing)
}

// reads monitorsettings.conf
func LoadAllMonitorProfiles() (map[string]MonitorProfile, error) {
	path, err := MonitorConfigFile()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]MonitorProfile{}, nil
		}
		return nil, fmt.Errorf("could not read monitor config: %w", err)
	}

	return parseMonitorConfig(string(data))
}

// loads a single monitor profile by name
func LoadMonitorProfile(name string) (MonitorProfile, error) {
	all, err := LoadAllMonitorProfiles()
	if err != nil {
		return MonitorProfile{}, err
	}

	profile, ok := all[name]
	if !ok {
		return MonitorProfile{}, fmt.Errorf("no monitor profile named %q found", name)
	}

	return profile, nil
}

// checks if a monitor profile name is taken
func MonitorProfileExists(name string) (bool, error) {
	all, err := LoadAllMonitorProfiles()
	if err != nil {
		return false, err
	}
	_, ok := all[name]
	return ok, nil
}

// deletes a monitor profile by name
func RemoveMonitorProfile(name string) error {
	all, err := LoadAllMonitorProfiles()
	if err != nil {
		return err
	}

	if _, ok := all[name]; !ok {
		return fmt.Errorf("no monitor profile named %q found", name)
	}

	delete(all, name)
	return writeAllMonitorProfiles(all)
}

func writeAllMonitorProfiles(profiles map[string]MonitorProfile) error {
	path, err := MonitorConfigFile()
	if err != nil {
		return err
	}

	var sb strings.Builder
	for _, p := range profiles {
		sb.WriteString(fmt.Sprintf("[%s]\n", p.Name))
		sb.WriteString(fmt.Sprintf("type            = %s\n", p.Type))
		sb.WriteString(fmt.Sprintf("host            = %s\n", p.Host))
		sb.WriteString(fmt.Sprintf("port            = %d\n", p.Port))
		sb.WriteString(fmt.Sprintf("username        = %s\n", p.Username))
		sb.WriteString(fmt.Sprintf("password        = %s\n", p.Password))
		sb.WriteString(fmt.Sprintf("dbname          = %s\n", p.DBName))
		sb.WriteString(fmt.Sprintf("monitorinterval = %d\n", p.MonitorInterval))
		sb.WriteString(fmt.Sprintf("webhookurl      = %s\n", p.WebhookURL))
		sb.WriteString(fmt.Sprintf("enabled         = %t\n", p.Enabled))
		sb.WriteString("\n")
	}

	return os.WriteFile(path, []byte(sb.String()), 0600)
}

func parseMonitorConfig(content string) (map[string]MonitorProfile, error) {
	profiles := map[string]MonitorProfile{}
	var current *MonitorProfile

	for _, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimSpace(rawLine)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if current != nil {
				profiles[current.Name] = *current
			}
			name := line[1 : len(line)-1]
			current = &MonitorProfile{Name: name}
			continue
		}

		if current == nil {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "type":
			current.Type = val
		case "host":
			current.Host = val
		case "port":
			port, err := strconv.Atoi(val)
			if err == nil {
				current.Port = port
			}
		case "username":
			current.Username = val
		case "password":
			current.Password = val
		case "dbname":
			current.DBName = val
		case "monitorinterval":
			interval, err := strconv.Atoi(val)
			if err == nil {
				current.MonitorInterval = interval
			}
		case "webhookurl":
			current.WebhookURL = val
		case "enabled":
			current.Enabled = val == "true"
		}
	}

	if current != nil {
		profiles[current.Name] = *current
	}

	return profiles, nil
}