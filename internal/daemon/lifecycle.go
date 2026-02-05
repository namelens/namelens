package daemon

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Common errors for lifecycle operations.
var (
	ErrServerNotRunning     = errors.New("server is not running")
	ErrServerAlreadyRunning = errors.New("server is already running on this port")
	ErrStopTimeout          = errors.New("server did not stop within timeout")
	ErrPortInUse            = errors.New("port is in use by another process")
	ErrDaemonStartupFailed  = errors.New("daemon process exited during startup")
)

// StartupVerifyDelay is the time to wait before checking if daemon is still running.
const StartupVerifyDelay = 500 * time.Millisecond

// StartupVerifyTimeout is the maximum time to wait for daemon startup verification.
const StartupVerifyTimeout = 3 * time.Second

// ServerStatus represents the status of a server instance.
type ServerStatus struct {
	Running   bool
	PID       uint32
	Port      int
	Name      string
	Cmdline   string
	StartTime time.Time
	Uptime    time.Duration
	Stale     bool // True if PID file exists but process is not running
	Managed   bool // True if this is a namelens server (has PID file)
}

// DaemonEnvVar is the environment variable set when running as a daemon.
const DaemonEnvVar = "NAMELENS_DAEMON"

// DefaultGracePeriod is the default time to wait for graceful shutdown.
const DefaultGracePeriod = 10 * time.Second

// Status checks the status of a server on the given port.
// It first checks the PID file, then validates the process is actually running.
func Status(port int) (*ServerStatus, error) {
	pidFile, err := NewPIDFile(port, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create PID file handle: %w", err)
	}

	status := &ServerStatus{
		Port: port,
	}

	// Check PID file
	pid, err := pidFile.Read()
	if err != nil {
		if errors.Is(err, ErrPIDFileNotFound) {
			// No PID file - check if something else is on the port
			proc, err := FindProcessOnPort(port)
			if err != nil {
				return nil, err
			}
			if proc != nil {
				status.Running = true
				status.PID = proc.PID
				status.Name = proc.Name
				status.Cmdline = proc.Cmdline
				status.StartTime = proc.StartTime
				if !proc.StartTime.IsZero() {
					status.Uptime = time.Since(proc.StartTime)
				}
			}
			return status, nil
		}
		return nil, err
	}

	status.PID = uint32(pid)

	// Validate process is running
	proc, err := FindProcessByPID(uint32(pid))
	if err != nil {
		return nil, err
	}

	if proc == nil || !proc.Running {
		// PID file exists but process is not running - stale
		status.Stale = true
		return status, nil
	}

	status.Running = true
	status.Managed = true // Has PID file, so it's a managed namelens server
	status.Name = proc.Name
	status.Cmdline = proc.Cmdline
	status.StartTime = proc.StartTime
	if !proc.StartTime.IsZero() {
		status.Uptime = time.Since(proc.StartTime)
	}

	return status, nil
}

// StartDaemon spawns the server as a background daemon process.
// It returns the PID of the spawned process.
// The function verifies the daemon starts successfully by checking it's still
// running after a brief delay.
func StartDaemon(executable string, args []string, port int) (int, error) {
	// Check if server is already running
	status, err := Status(port)
	if err != nil {
		return 0, err
	}

	if status.Running {
		if status.Managed {
			// A namelens server (with PID file) is running
			return 0, fmt.Errorf("%w (PID %d) - use 'namelens serve stop --port %d' first",
				ErrServerAlreadyRunning, status.PID, port)
		}
		// Some other process is using the port
		return 0, fmt.Errorf("%w: port %d is in use by %s (PID %d)",
			ErrPortInUse, port, status.Name, status.PID)
	}

	// Clean up stale PID file if present
	if status.Stale {
		pidFile, _ := NewPIDFile(port, "")
		_ = pidFile.Remove()
	}

	// Build command with daemon environment variable
	cmd := exec.Command(executable, args...)
	cmd.Env = append(os.Environ(), DaemonEnvVar+"=true")

	// Detach from parent process
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Start the process
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start daemon: %w", err)
	}

	pid := cmd.Process.Pid

	// Write PID file early so we can track the process
	pidFile, err := NewPIDFile(port, "")
	if err != nil {
		// Try to kill the orphaned process
		_ = cmd.Process.Kill()
		return 0, fmt.Errorf("failed to create PID file: %w", err)
	}

	if err := pidFile.Write(pid); err != nil {
		_ = cmd.Process.Kill()
		return 0, fmt.Errorf("failed to write PID file: %w", err)
	}

	// Detach from the process so we don't wait for it
	_ = cmd.Process.Release()

	// Verify the daemon starts successfully by checking it's still running
	// after a brief delay. This catches immediate startup failures.
	if err := verifyDaemonStartup(uint32(pid), port, pidFile); err != nil {
		return 0, err
	}

	return pid, nil
}

// verifyDaemonStartup waits briefly and verifies the daemon is still running.
// If the process exits during startup, it cleans up and returns an error.
func verifyDaemonStartup(pid uint32, port int, pidFile *PIDFile) error {
	// Wait a moment for initial startup
	time.Sleep(StartupVerifyDelay)

	// Check if process is still running
	if !IsProcessRunning(pid) {
		// Process died during startup - clean up PID file
		_ = pidFile.Remove()
		return fmt.Errorf("%w - check server logs or run without --daemon for details", ErrDaemonStartupFailed)
	}

	// Wait a bit longer and check again to catch slower startup failures
	deadline := time.Now().Add(StartupVerifyTimeout - StartupVerifyDelay)
	for time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)

		if !IsProcessRunning(pid) {
			_ = pidFile.Remove()
			return fmt.Errorf("%w - check server logs or run without --daemon for details", ErrDaemonStartupFailed)
		}

		// Try to verify the server is actually listening on the port
		proc, _ := FindProcessOnPort(port)
		if proc != nil && proc.PID == pid {
			// Server is running and listening - success!
			return nil
		}
	}

	// Process is still running but may not be listening yet
	// This is acceptable - the server might just be slow to start
	if IsProcessRunning(pid) {
		return nil
	}

	_ = pidFile.Remove()
	return fmt.Errorf("%w - check server logs or run without --daemon for details", ErrDaemonStartupFailed)
}

// Stop gracefully stops a running server.
// It first tries SIGTERM, then SIGKILL if the grace period expires.
func Stop(port int, gracePeriod time.Duration) error {
	if gracePeriod == 0 {
		gracePeriod = DefaultGracePeriod
	}

	status, err := Status(port)
	if err != nil {
		return err
	}

	// Clean up stale PID file
	if status.Stale {
		pidFile, _ := NewPIDFile(port, "")
		_ = pidFile.Remove()
		return ErrServerNotRunning
	}

	if !status.Running {
		return ErrServerNotRunning
	}

	pid := status.PID

	// Send SIGTERM
	if err := TerminateProcess(pid); err != nil {
		return fmt.Errorf("failed to send termination signal: %w", err)
	}

	// Wait for graceful shutdown
	exited, err := WaitForProcessExit(pid, gracePeriod)
	if err != nil {
		return err
	}

	if exited {
		// Process exited gracefully - clean up PID file
		pidFile, _ := NewPIDFile(port, "")
		_ = pidFile.Remove()
		return nil
	}

	// Grace period expired - force kill
	if err := ForceKillProcess(pid); err != nil {
		return fmt.Errorf("failed to force kill process: %w", err)
	}

	// Wait briefly for force kill to take effect
	exited, _ = WaitForProcessExit(pid, 2*time.Second)
	if !exited {
		return ErrStopTimeout
	}

	// Clean up PID file
	pidFile, _ := NewPIDFile(port, "")
	_ = pidFile.Remove()

	return nil
}

// ForceStop immediately kills a server without graceful shutdown.
func ForceStop(port int) error {
	status, err := Status(port)
	if err != nil {
		return err
	}

	// Clean up stale PID file
	if status.Stale {
		pidFile, _ := NewPIDFile(port, "")
		_ = pidFile.Remove()
		return ErrServerNotRunning
	}

	if !status.Running {
		return ErrServerNotRunning
	}

	if err := ForceKillProcess(status.PID); err != nil {
		return fmt.Errorf("failed to force kill process: %w", err)
	}

	// Clean up PID file
	pidFile, _ := NewPIDFile(port, "")
	_ = pidFile.Remove()

	return nil
}

// CleanupResult contains information about what was cleaned up.
type CleanupResult struct {
	ProcessKilled  bool
	PIDFileRemoved bool
	PID            uint32
}

// Cleanup kills any process on the given port and removes stale PID files.
// This is useful for cleaning up orphaned processes and stale state.
// Returns a CleanupResult indicating what actions were taken.
func Cleanup(port int, force bool) (*CleanupResult, error) {
	result := &CleanupResult{}

	// Check for stale PID file first
	pidFile, err := NewPIDFile(port, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create PID file handle: %w", err)
	}

	// Check if there's a process on the port
	proc, err := FindProcessOnPort(port)
	if err != nil {
		return nil, err
	}

	if proc != nil {
		result.PID = proc.PID
		if force {
			if err := ForceKillProcess(proc.PID); err != nil {
				return nil, fmt.Errorf("failed to force kill process: %w", err)
			}
		} else {
			if err := TerminateProcess(proc.PID); err != nil {
				return nil, fmt.Errorf("failed to terminate process: %w", err)
			}

			// Wait for termination
			exited, _ := WaitForProcessExit(proc.PID, DefaultGracePeriod)
			if !exited {
				// Force kill if graceful failed
				if err := ForceKillProcess(proc.PID); err != nil {
					return nil, fmt.Errorf("failed to force kill process: %w", err)
				}
			}
		}
		result.ProcessKilled = true
	}

	// Always clean up any PID file for this port (handles stale files)
	if pidFile.Exists() {
		_ = pidFile.Remove()
		result.PIDFileRemoved = true
	}

	return result, nil
}

// IsDaemon returns true if this process was started as a daemon.
func IsDaemon() bool {
	return os.Getenv(DaemonEnvVar) == "true"
}

// ListServers returns status information for all known server instances.
func ListServers() ([]*ServerStatus, error) {
	pidFiles, err := ListPIDFiles("")
	if err != nil {
		return nil, err
	}

	var servers []*ServerStatus
	for _, pf := range pidFiles {
		status, err := Status(pf.Port)
		if err != nil {
			continue // Skip problematic entries
		}
		servers = append(servers, status)
	}

	return servers, nil
}
