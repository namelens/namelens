// Package daemon provides server lifecycle management for daemon mode.
//
// This package handles:
//   - PID file management for tracking running server instances
//   - Process discovery via sysprims for port-based lookups
//   - Graceful start/stop/status operations
package daemon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/namelens/namelens/internal/config"
)

// Common errors for PID file operations.
var (
	ErrPIDFileNotFound = errors.New("PID file not found")
	ErrPIDFileStale    = errors.New("PID file exists but process is not running")
	ErrPIDFileLocked   = errors.New("PID file locked by another process")
)

// PIDFile manages a PID file for tracking a running server instance.
type PIDFile struct {
	// Path is the full path to the PID file.
	Path string
	// Port is the port number this PID file tracks.
	Port int
}

// DefaultPIDDir returns the default directory for PID files.
// Uses XDG_DATA_HOME/namelens/run/ following XDG Base Directory specification.
// Falls back to ~/.local/share/namelens/run/ on most systems.
func DefaultPIDDir() (string, error) {
	dataDir := config.DefaultDataDir()
	if dataDir == "" {
		// Fallback if config can't determine data dir
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		dataDir = filepath.Join(home, ".local", "share", "namelens")
	}
	return filepath.Join(dataDir, "run"), nil
}

// NewPIDFile creates a new PIDFile for the given port.
// If dir is empty, uses the default PID directory (XDG_DATA_HOME/namelens/run/).
func NewPIDFile(port int, dir string) (*PIDFile, error) {
	if dir == "" {
		var err error
		dir, err = DefaultPIDDir()
		if err != nil {
			return nil, err
		}
	}

	return &PIDFile{
		Path: filepath.Join(dir, fmt.Sprintf("namelens-%d.pid", port)),
		Port: port,
	}, nil
}

// Write writes the current process PID to the PID file.
// Creates the parent directory if it doesn't exist.
func (p *PIDFile) Write(pid int) error {
	dir := filepath.Dir(p.Path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create PID directory: %w", err)
	}

	content := fmt.Sprintf("%d\n", pid)
	if err := os.WriteFile(p.Path, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	return nil
}

// Read reads the PID from the PID file.
// Returns ErrPIDFileNotFound if the file doesn't exist.
func (p *PIDFile) Read() (int, error) {
	data, err := os.ReadFile(p.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, ErrPIDFileNotFound
		}
		return 0, fmt.Errorf("failed to read PID file: %w", err)
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %w", err)
	}

	return pid, nil
}

// Remove deletes the PID file.
// Returns nil if the file doesn't exist.
func (p *PIDFile) Remove() error {
	err := os.Remove(p.Path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}
	return nil
}

// Exists returns true if the PID file exists.
func (p *PIDFile) Exists() bool {
	_, err := os.Stat(p.Path)
	return err == nil
}

// ListPIDFiles returns all PID files in the given directory.
// If dir is empty, uses the default PID directory.
func ListPIDFiles(dir string) ([]*PIDFile, error) {
	if dir == "" {
		var err error
		dir, err = DefaultPIDDir()
		if err != nil {
			return nil, err
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read PID directory: %w", err)
	}

	var pidFiles []*PIDFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, "namelens-") || !strings.HasSuffix(name, ".pid") {
			continue
		}

		// Extract port from filename: namelens-8080.pid -> 8080
		portStr := strings.TrimPrefix(name, "namelens-")
		portStr = strings.TrimSuffix(portStr, ".pid")
		port, err := strconv.Atoi(portStr)
		if err != nil {
			continue // Skip malformed PID files
		}

		pidFiles = append(pidFiles, &PIDFile{
			Path: filepath.Join(dir, name),
			Port: port,
		})
	}

	return pidFiles, nil
}
