package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type DBProfile struct {
	Name string
	Type string
	Host string
	Port int
	Username string
	Password string
	DBName string
	Storage string
	BackupDir string
	Interval int
	Enabled bool
}

// returns ~/.backuptool
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not find home directory: %w", err)
	}
	return filepath.Join(home, ".backuptool"), nil
}

// returns ~/.backuptool/settings.conf
func ConfigFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.conf"), nil
}

// creates the config directory if it does not exist
func EnsureConfigDir() error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0700) // 0700 = only owner can read/write
}

// reads settings.conf and returns all profiles
func LoadAllProfiles() (map[string]DBProfile, error) {
	path, err := ConfigFile()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]DBProfile{}, nil // no config yet, not an error
		}
		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	return parseConfig(string(data))
}

// loads a single profile by name
func LoadProfile(name string) (DBProfile, error) {
	all, err := LoadAllProfiles()
	if err != nil {
		return DBProfile{}, err
	}

	profile, ok := all[name]
	if !ok {
		return DBProfile{}, fmt.Errorf("no database profile named %q found", name)
	}

	return profile, nil
}

func ProfileExists(name string) (bool, error) {
	all, err := LoadAllProfiles()
	if err != nil {
		return false, err
	}
	_, ok := all[name]
	return ok, nil
}


// writes a profile to settings.conf
func SaveProfile(profile DBProfile) error {
	if err := EnsureConfigDir(); err != nil {
		return err
	}

	existing, err := LoadAllProfiles()
	if err != nil {
		existing = map[string]DBProfile{}
	}

	existing[profile.Name] = profile

	return writeAllProfiles(existing)
}

// serializes all profiles back to settings.conf
func writeAllProfiles(profiles map[string]DBProfile) error {
	path, err := ConfigFile()
	if err != nil {
		return err
	}

	var sb strings.Builder
	for _, p := range profiles {
		sb.WriteString(fmt.Sprintf("[%s]\n", p.Name))
		sb.WriteString(fmt.Sprintf("type     = %s\n", p.Type))
		sb.WriteString(fmt.Sprintf("host     = %s\n", p.Host))
		sb.WriteString(fmt.Sprintf("port     = %d\n", p.Port))
		sb.WriteString(fmt.Sprintf("username = %s\n", p.Username))
		sb.WriteString(fmt.Sprintf("password = %s\n", p.Password))
		sb.WriteString(fmt.Sprintf("dbname   = %s\n", p.DBName))
		sb.WriteString(fmt.Sprintf("storage   = %s\n", p.Storage))
		sb.WriteString(fmt.Sprintf("backupdir = %s\n", p.BackupDir))
		sb.WriteString(fmt.Sprintf("interval  = %d\n", p.Interval))
		sb.WriteString(fmt.Sprintf("enabled   = %t\n", p.Enabled)) 
		sb.WriteString("\n")
	}

	// 0600 = only owner can read/write
	return os.WriteFile(path, []byte(sb.String()), 0600)
}

// parses the settings.conf into a map of profiles
func parseConfig(content string) (map[string]DBProfile, error) {
	profiles := map[string]DBProfile{}
	var current *DBProfile

	for _, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimSpace(rawLine)

		if line == "" || strings.HasPrefix(line, "#") {
			continue // skip empty lines and comments
		}

		// section header like [mydb]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if current != nil {
				profiles[current.Name] = *current // save previous profile
			}
			name := line[1 : len(line)-1]
			current = &DBProfile{Name: name}
			continue
		}

		// Key = value line
		if current == nil {
			continue // no section started yet, skip
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
		case "storage":                         
		    current.Storage = val
		case "backupdir":                        
		    current.BackupDir = val
		case "interval":
		    interval, err := strconv.Atoi(val)
		    if err == nil {
		        current.Interval = interval
		    }
		case "enabled":
			current.Enabled = val == "true"
		}
	}

	// save the last profile (no trailing section to trigger the save)
	if current != nil {
		profiles[current.Name] = *current
	}

	return profiles, nil
}

func RemoveProfile(name string) error {
	all, err := LoadAllProfiles()
	if err != nil {
		return err
	}

	if _, ok := all[name]; !ok {
		return fmt.Errorf("no profile named %q found", name)
	}

	delete(all, name)
	return writeAllProfiles(all)
}