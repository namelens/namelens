package daemon

import (
	"fmt"
	"strings"
	"time"

	"github.com/3leaps/sysprims/bindings/go/sysprims"
)

// ProcessInfo contains information about a running server process.
type ProcessInfo struct {
	PID       uint32
	Port      int
	Name      string
	Cmdline   string
	StartTime time.Time
	Running   bool
}

// FindProcessByPID checks if a process with the given PID is running.
// Returns process info if found, nil if not running.
func FindProcessByPID(pid uint32) (*ProcessInfo, error) {
	proc, err := sysprims.ProcessGet(pid)
	if err != nil {
		// Process not found - not an error, just means it's not running
		return nil, nil
	}

	info := &ProcessInfo{
		PID:     proc.PID,
		Name:    proc.Name,
		Cmdline: strings.Join(proc.Cmdline, " "),
		Running: true,
	}

	if proc.StartTimeUnixMS != nil {
		info.StartTime = time.UnixMilli(int64(*proc.StartTimeUnixMS))
	}

	return info, nil
}

// FindProcessOnPort finds the process listening on the given port.
// Returns the process info if found, nil if no process is listening.
func FindProcessOnPort(port int) (*ProcessInfo, error) {
	localPort := uint16(port)
	filter := &sysprims.PortFilter{
		LocalPort: &localPort,
	}

	snapshot, err := sysprims.ListeningPorts(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list listening ports: %w", err)
	}

	if len(snapshot.Bindings) == 0 {
		return nil, nil
	}

	// Get the first binding (there may be multiple for IPv4/IPv6)
	binding := snapshot.Bindings[0]

	if binding.PID == nil {
		// Port is bound but we can't determine the PID (permission issue)
		return &ProcessInfo{
			Port:    port,
			Running: true,
		}, nil
	}

	pid := *binding.PID

	// If process info is embedded, use it
	if binding.Process != nil {
		info := &ProcessInfo{
			PID:     pid,
			Port:    port,
			Name:    binding.Process.Name,
			Cmdline: strings.Join(binding.Process.Cmdline, " "),
			Running: true,
		}
		if binding.Process.StartTimeUnixMS != nil {
			info.StartTime = time.UnixMilli(int64(*binding.Process.StartTimeUnixMS))
		}
		return info, nil
	}

	// Otherwise fetch process details separately
	proc, err := sysprims.ProcessGet(pid)
	if err != nil {
		// Process found on port but can't get details
		return &ProcessInfo{
			PID:     pid,
			Port:    port,
			Running: true,
		}, nil
	}

	info := &ProcessInfo{
		PID:     proc.PID,
		Port:    port,
		Name:    proc.Name,
		Cmdline: strings.Join(proc.Cmdline, " "),
		Running: true,
	}

	if proc.StartTimeUnixMS != nil {
		info.StartTime = time.UnixMilli(int64(*proc.StartTimeUnixMS))
	}

	return info, nil
}

// IsProcessRunning checks if the process with the given PID is still running.
func IsProcessRunning(pid uint32) bool {
	proc, _ := FindProcessByPID(pid)
	return proc != nil && proc.Running
}

// TerminateProcess sends a termination signal to the process.
// Returns nil if the process terminates successfully.
func TerminateProcess(pid uint32) error {
	return sysprims.Terminate(pid)
}

// ForceKillProcess forcefully kills the process.
// This should only be used if graceful termination fails.
func ForceKillProcess(pid uint32) error {
	return sysprims.ForceKill(pid)
}

// WaitForProcessExit waits for a process to exit within the timeout.
// Returns true if the process exited, false if timeout was reached.
func WaitForProcessExit(pid uint32, timeout time.Duration) (bool, error) {
	result, err := sysprims.WaitPID(pid, timeout)
	if err != nil {
		return false, fmt.Errorf("failed waiting for process: %w", err)
	}

	return result.Exited, nil
}
