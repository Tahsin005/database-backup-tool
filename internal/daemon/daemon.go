package daemon

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Tahsin005/database-backup-tool/internal/config"
)

type Kind string

const (
	KindBackup  Kind = "backup"
	KindMonitor Kind = "monitor"
)

type Daemon struct {
	Name string
	Kind Kind
}

// creates a new daemon manager for a backup or monitor profile
func New(name string, kind Kind) *Daemon {
	return &Daemon{
		Name: name,
		Kind: kind,
	}
}

// returns the absolute path to ~/.backuptool/<name>[.monitor].pid
func (d *Daemon) PIDFilePath() (string, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}

	var filename string
	if d.Kind == KindMonitor {
		filename = fmt.Sprintf("%s.monitor.pid", d.Name)
	} else {
		filename = fmt.Sprintf("%s.pid", d.Name)
	}

	return filepath.Join(dir, filename), nil
}

// saves the current process's pid into the pid file with 0600 permissions
func (d *Daemon) WritePID() error {
	path, err := d.PIDFilePath()
	if err != nil {
		return err
	}

	pid := os.Getpid()
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0600)
}

// removes the daemon's pid file
func (d *Daemon) DeletePID() error {
	path, err := d.PIDFilePath()
	if err != nil {
		return err
	}

	err = os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// reads the pid from the pid file
func (d *Daemon) ReadPID() (int, error) {
	path, err := d.PIDFilePath()
	if err != nil {
		return 0, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("corrupted PID file: %w", err)
	}

	return pid, nil
}

// checks whether the daemon process is actively running and cleans up stale pid files
func (d *Daemon) IsRunning() (bool, int) {
	pid, err := d.ReadPID()
	if err != nil || pid <= 0 {
		return false, 0
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		_ = d.DeletePID()
		return false, 0
	}

	// signal 0 checks if process exists without sending a real signal
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		// process is dead, clean up stale pid file
		_ = d.DeletePID()
		return false, 0
	}

	return true, pid
}

// gracefully terminates the daemon process with sigterm, falling back to sigkill on timeout
func (d *Daemon) Stop(timeout time.Duration) error {
	running, pid := d.IsRunning()
	if !running || pid <= 0 {
		_ = d.DeletePID()
		return nil
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		_ = d.DeletePID()
		return nil
	}

	// send sigterm for graceful exit
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// if unable to send sigterm, attempt forceful kill
		_ = process.Kill()
		_ = d.DeletePID()
		return nil
	}

	// wait up to timeout for clean termination
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		if err := process.Signal(syscall.Signal(0)); err != nil {
			// process exited cleanly
			_ = d.DeletePID()
			return nil
		}
	}

	// timed out; force kill with sigkill
	if err := process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		_ = d.DeletePID()
		return fmt.Errorf("failed to kill process %d: %w", pid, err)
	}

	_ = d.DeletePID()
	return nil
}

// executes the current binary in the background detached from stdio
func (d *Daemon) LaunchBackground(args ...string) (*exec.Cmd, error) {
	self, err := os.Executable()
	if err != nil {
		self = os.Args[0]
	}

	cmd := exec.Command(self, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return cmd, nil
}
