# Database Backup Tool

A command-line utility written in Go for backing up, restoring, and monitoring PostgreSQL databases. Supports automatic scheduled backups, gzip compression, per-profile configuration, background daemon management, and Discord alerts for database downtime.

> Repository: [https://github.com/Tahsin005/database-backup-tool](https://github.com/Tahsin005/database-backup-tool)

---

## Table of Contents

- [Features](#features)
- [Requirements](#requirements)
- [Installation](#installation)
- [Project Structure](#project-structure)
- [Configuration](#configuration)
- [Usage](#usage)
  - [add](#add)
  - [start](#start)
  - [stop](#stop)
  - [list](#list)
  - [status](#status)
  - [edit](#edit)
  - [remove](#remove)
  - [monitor add](#monitor-add)
  - [monitor start](#monitor-start)
  - [monitor stop](#monitor-stop)
  - [monitor status](#monitor-status)
  - [monitor remove](#monitor-remove)
- [How It Works](#how-it-works)
- [Roadmap](#roadmap)
- [License](#license)

---

## Features

- Add and manage multiple PostgreSQL database profiles
- Run scheduled full backups at a configurable interval
- Compress backup files using gzip (.sql.gz)
- Save backups to a user-specified local directory
- Daemonize the backup process — terminal stays free
- Stop, edit, or remove any profile at any time
- Enable or disable profiles without deleting them
- Monitor database availability on a separate interval
- Send Discord alerts when a database goes down or recovers
- Separate config files for backup and monitor profiles
- All activity logged to per-database log files under ~/.backuptool/

---

## Requirements

- PostgreSQL client tools (for `pg_dump`)
- A running PostgreSQL instance
- Discord webhook URL (for monitoring alerts)

### Install PostgreSQL client tools

**Arch Linux:**
```bash
sudo pacman -S postgresql
```

**Ubuntu / Debian:**
```bash
sudo apt install postgresql-client
```

Verify `pg_dump` is available:
```bash
pg_dump --version
```

---

## Installation

### Option 1 — Download prebuilt binary (recommended)

```bash
curl -L https://github.com/Tahsin005/database-backup-tool/releases/download/1.0.0/backuptool -o backuptool
chmod +x backuptool
```

Optionally move it to PATH so you can run it from anywhere:
```bash
sudo mv backuptool /usr/local/bin/
```

### Option 2 — Build from source

Requires Go 1.21+.

```bash
git clone https://github.com/Tahsin005/database-backup-tool
cd database-backup-tool
go mod tidy
go build -o backuptool .

# Optionally move to PATH
sudo mv backuptool /usr/local/bin/
```
---

## Project Structure

```
database-backup-tool/
├── main.go
├── cmd/
│   ├── root.go          # Base Cobra command
│   ├── add.go           # Add a backup profile
│   ├── start.go         # Start backup daemon
│   ├── stop.go          # Stop backup daemon
│   ├── list.go          # List all backup profiles
│   ├── status.go        # Show backup status
│   ├── edit.go          # Edit a backup profile
│   ├── remove.go        # Remove a backup profile
│   └── monitor.go       # All monitor subcommands
├── internal/
│   ├── config/
│   │   ├── config.go          # Backup profile read/write
│   │   └── monitorconfig.go   # Monitor profile read/write
│   ├── db/
│   │   └── postgres.go        # PostgreSQL connection + ping
│   ├── backup/
│   │   └── backup.go          # pg_dump + gzip + scheduler
│   └── monitor/
│       └── monitor.go         # Ping loop + Discord alerts
└── go.mod
```

---

## Configuration

All configuration is stored under `~/.backuptool/`:

```
~/.backuptool/
├── settings.conf           # Backup profiles
├── monitorsettings.conf    # Monitor profiles
├── <profile>.pid           # Backup daemon PID
├── <profile>.monitor.pid   # Monitor daemon PID
├── <dbname>.log            # Backup activity log
└── <dbname>.monitor.log    # Monitor activity log
```

### Backup profile (`settings.conf`)

```toml
[my-local-pg]
type      = postgres
host      = localhost
port      = 5432
username  = admin
password  = admin123
dbname    = testdb
storage   = local
backupdir = /home/user/backups
interval  = 60
enabled   = true
```

### Monitor profile (`monitorsettings.conf`)

```toml
[my-local-pg]
type            = postgres
host            = localhost
port            = 5432
username        = admin
password        = admin123
dbname          = testdb
monitorinterval = 5
webhookurl      = https://discord.com/api/webhooks/xxx/yyy
enabled         = true
```

---

## Usage

### add

Add a new backup profile interactively. Walks through a wizard, tests the connection, and saves the profile.

```bash
backuptool add
```

```
=== Add New Database Profile ===

Profile name (e.g. my-local-pg): my-local-pg
Database type:
  [1] PostgreSQL
Choose (1):
Host [localhost]:
Port [5432]:
Username: admin
Password: admin123
Database name: testdb
Storage type:
  [1] Local
Choose [1]:
Backup directory [/home/user/backups]:
Backup interval (minutes) [60]: 30

Testing connection...
Connection successful!

Profile "my-local-pg" saved successfully!
Run "backuptool start my-local-pg" to start backing up.
```

---

### start

Start the backup daemon for a profile. The process runs in the background — the terminal is freed immediately.

```bash
backuptool start <profile-name>
```

```
Backup daemon started for profile "my-local-pg" (PID: 12345)
Run "backuptool status" to check its state.
```

---

### stop

Stop the running backup daemon for a profile.

```bash
backuptool stop <profile-name>
```

```
Backup daemon for "my-local-pg" stopped.
```

---

### list

List all configured backup profiles with their live daemon status.

```bash
backuptool list
```

```
NAME                 TYPE         HOST             PORT   DATABASE         ENABLED   
-----------------------------------------------------------------------------
my-local-pg          postgres     localhost        5432   testdb           yes        [running]
```

---

### status

Show detailed status for all backup profiles including last backup file, time since last backup, and log file location.

```bash
backuptool status
```

```
Profile    : my-local-pg
Database   : testdb (postgres)
Host       : localhost:5432
Storage    : local
Backup dir : /home/user/backups
Interval   : every 30 min
Enabled    : yes
Daemon     : running (PID: 12345)
Last backup: backup_testdb_2026-06-01_10-22-00.sql.gz (14m 32s ago)
Log file   : /home/user/.backuptool/testdb.log
---------------------------------------------
```

---

### edit

Edit the backup directory, interval, or enabled state of an existing profile. If the daemon is running, you will be asked whether to stop it before editing. After editing you must restart the daemon manually.

```bash
backuptool edit <profile-name>
```

```
Daemon for "my-local-pg" is currently running.
Stop it and proceed with editing? (yes/no): yes
Daemon stopped.

=== Edit Profile "my-local-pg" ===
Press Enter to keep the current value.

Backup directory [/home/user/backups]: /home/user/pg-backups
Backup interval (minutes) [30]: 60
Enabled (true/false) [true]:

Profile "my-local-pg" updated successfully.
Run "backuptool start my-local-pg" to restart the daemon.
```

---

### remove

Remove a backup profile permanently. The daemon must be stopped first. Asks for confirmation before deleting.

```bash
backuptool remove <profile-name>
```

```
Are you sure you want to remove "my-local-pg"? (y/n): y
Profile "my-local-pg" removed.
```

---

### monitor add

Add a new monitor profile. You can import connection details from an existing backup profile or enter them manually.

```bash
backuptool monitor add
```

```
=== Add Monitor Profile ===

Import connection details from an existing backup profile? (yes/no): yes

Available backup profiles:
  [1] my-local-pg  (testdb @ localhost)

Choose a number: 1

Imported connection details from "my-local-pg".

Monitor interval (minutes) [5]: 2
Discord webhook URL: https://discord.com/api/webhooks/xxx/yyy

Testing connection...
Connection successful!

Monitor profile "my-local-pg" saved.
Run "backuptool monitor start my-local-pg" to begin monitoring.
```

---

### monitor start

Start the monitor daemon for a profile in the background.

```bash
backuptool monitor start <profile-name>
```

```
Monitor daemon started for "my-local-pg" (PID: 13456)
Run "backuptool monitor status" to check its state.
```

---

### monitor stop

Stop the running monitor daemon for a profile.

```bash
backuptool monitor stop <profile-name>
```

```
Monitor daemon for "my-local-pg" stopped.
```

---

### monitor status

Show the current state of all monitor profiles including daemon status, interval, webhook URL, and log file location.

```bash
backuptool monitor status
```

```
Profile  : my-local-pg
Database : testdb (postgres)
Host     : localhost:5432
Interval : every 2 min
Webhook  : https://discord.com/api/webhooks/xxx/yyy
Enabled  : yes
Daemon   : running (PID: 13456)
Log file : /home/user/.backuptool/testdb.monitor.log
---------------------------------------------
```

---

### monitor remove

Remove a monitor profile. The monitor daemon must be stopped first.

```bash
backuptool monitor remove <profile-name>
```

```
Are you sure you want to remove monitor profile "my-local-pg"? (y/n): y
Monitor profile "my-local-pg" removed.
```

---

## How It Works

### Backup daemon

When you run `backuptool start <name>`, the process detects it is running in the foreground. It re-launches itself as a background child process with a hidden `--daemon` flag, then exits — freeing the terminal. The child writes its PID to `~/.backuptool/<name>.pid`, then enters a ticker loop that calls `pg_dump` on the configured interval. The dump output is piped directly into a gzip writer and saved as a `.sql.gz` file in the configured backup directory. All activity is written to `~/.backuptool/<dbname>.log`.

### Monitor daemon

The monitor daemon follows the same daemonize pattern. Once running it pings the database on the configured interval. On the first failed ping it sends a **🔴 Database Down** alert to the Discord webhook. It continues pinging on each interval. When the database responds again it sends a **🟢 Database Recovered** alert. This means you are notified of both the outage and the recovery without any manual checking. All ping results are written to `~/.backuptool/<dbname>.monitor.log`.

### Discord alerts

Alerts are sent as plain HTTP POST requests to your Discord webhook URL with a JSON payload. No external library is required. Example down alert:

```
🔴 Database Down
Profile: my-local-pg
Database: testdb @ localhost:5432
Error: failed to ping: connection refused
Time: 2026-06-01 22:45:01
```

---

## Roadmap

- MySQL and MongoDB support
- Cloud storage backends (AWS S3, Google Cloud Storage)
- Restore command to recover from a backup file
- Cron expression support for backup scheduling
- Slack notification support alongside Discord
- Encrypted backup files

---

## License

MIT License. See [LICENSE](LICENSE) for details.