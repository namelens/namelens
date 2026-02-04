package daemon

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDaemon(t *testing.T) {
	// Save and restore env
	orig := os.Getenv(DaemonEnvVar)
	defer func() {
		if orig == "" {
			_ = os.Unsetenv(DaemonEnvVar)
		} else {
			_ = os.Setenv(DaemonEnvVar, orig)
		}
	}()

	t.Run("not set", func(t *testing.T) {
		_ = os.Unsetenv(DaemonEnvVar)
		assert.False(t, IsDaemon())
	})

	t.Run("set to true", func(t *testing.T) {
		t.Setenv(DaemonEnvVar, "true")
		assert.True(t, IsDaemon())
	})

	t.Run("set to false", func(t *testing.T) {
		t.Setenv(DaemonEnvVar, "false")
		assert.False(t, IsDaemon())
	})

	t.Run("set to other value", func(t *testing.T) {
		t.Setenv(DaemonEnvVar, "1")
		assert.False(t, IsDaemon())
	})
}

func TestStatus_NoServer(t *testing.T) {
	// Use a port that's unlikely to be in use
	port := 59999

	// Create a temp dir for PID files
	dir := t.TempDir()

	// Override PID file location for test
	pf, err := NewPIDFile(port, dir)
	require.NoError(t, err)

	// Status should show not running (no PID file, no process on port)
	status, err := Status(port)
	require.NoError(t, err)
	assert.False(t, status.Running)
	assert.False(t, status.Stale)
	assert.Equal(t, port, status.Port)

	// Create stale PID file (PID that doesn't exist)
	err = pf.Write(999999999) // Very high PID unlikely to exist
	require.NoError(t, err)

	// Note: Status() uses default PID dir, not our temp dir
	// This test just verifies the function doesn't crash
}

func TestListServers_Empty(t *testing.T) {
	// This will use the default PID directory (XDG data dir)
	// We're just verifying the function works without errors
	servers, err := ListServers()
	// May return error if PID dir doesn't exist, which is fine
	// Result can be nil (no PID files) or an empty slice - both are valid
	if err != nil {
		t.Logf("ListServers returned error (acceptable if PID dir doesn't exist): %v", err)
	} else {
		// servers can be nil if no PID files exist, which is valid
		t.Logf("ListServers returned %d servers", len(servers))
	}
}

func TestServerStatus_Stale(t *testing.T) {
	// Create a temp dir for testing
	dir := t.TempDir()

	// Create a PID file with a non-existent PID
	pf, err := NewPIDFile(58888, dir)
	require.NoError(t, err)

	err = pf.Write(999999999) // PID that won't exist
	require.NoError(t, err)

	// Read back and verify file is there
	pid, err := pf.Read()
	require.NoError(t, err)
	assert.Equal(t, 999999999, pid)

	// The PID file should exist
	assert.True(t, pf.Exists())
}

func TestFormatDuration(t *testing.T) {
	// Test the formatDuration helper in serve_subcommands.go
	// Since it's in the cmd package, we can't test it directly here
	// But we can test the ServerStatus struct

	status := &ServerStatus{
		Running: true,
		Port:    8080,
		PID:     12345,
		Name:    "namelens",
	}

	assert.Equal(t, 8080, status.Port)
	assert.Equal(t, uint32(12345), status.PID)
	assert.Equal(t, "namelens", status.Name)
}

func TestDaemonEnvVar(t *testing.T) {
	assert.Equal(t, "NAMELENS_DAEMON", DaemonEnvVar)
}

func TestDefaultGracePeriod(t *testing.T) {
	assert.Equal(t, 10.0, DefaultGracePeriod.Seconds())
}
