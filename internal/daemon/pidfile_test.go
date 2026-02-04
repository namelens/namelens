package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPIDFile(t *testing.T) {
	t.Run("with custom directory", func(t *testing.T) {
		dir := t.TempDir()
		pf, err := NewPIDFile(8080, dir)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(dir, "namelens-8080.pid"), pf.Path)
		assert.Equal(t, 8080, pf.Port)
	})

	t.Run("with default directory", func(t *testing.T) {
		pf, err := NewPIDFile(9000, "")
		require.NoError(t, err)

		// Should use XDG data dir + /run/
		assert.Contains(t, pf.Path, "namelens")
		assert.Contains(t, pf.Path, "run")
		assert.Contains(t, pf.Path, "namelens-9000.pid")
		assert.Equal(t, 9000, pf.Port)
	})
}

func TestPIDFile_WriteRead(t *testing.T) {
	dir := t.TempDir()
	pf, err := NewPIDFile(8080, dir)
	require.NoError(t, err)

	// Write PID
	err = pf.Write(12345)
	require.NoError(t, err)

	// Verify file exists
	assert.True(t, pf.Exists())

	// Read PID back
	pid, err := pf.Read()
	require.NoError(t, err)
	assert.Equal(t, 12345, pid)
}

func TestPIDFile_ReadNotFound(t *testing.T) {
	dir := t.TempDir()
	pf, err := NewPIDFile(8080, dir)
	require.NoError(t, err)

	// Should return ErrPIDFileNotFound
	_, err = pf.Read()
	assert.ErrorIs(t, err, ErrPIDFileNotFound)
}

func TestPIDFile_Remove(t *testing.T) {
	dir := t.TempDir()
	pf, err := NewPIDFile(8080, dir)
	require.NoError(t, err)

	// Write and verify
	err = pf.Write(12345)
	require.NoError(t, err)
	assert.True(t, pf.Exists())

	// Remove
	err = pf.Remove()
	require.NoError(t, err)
	assert.False(t, pf.Exists())

	// Removing again should not error
	err = pf.Remove()
	require.NoError(t, err)
}

func TestPIDFile_Exists(t *testing.T) {
	dir := t.TempDir()
	pf, err := NewPIDFile(8080, dir)
	require.NoError(t, err)

	// Should not exist initially
	assert.False(t, pf.Exists())

	// Create and check
	err = pf.Write(1)
	require.NoError(t, err)
	assert.True(t, pf.Exists())
}

func TestPIDFile_InvalidPID(t *testing.T) {
	dir := t.TempDir()
	pidPath := filepath.Join(dir, "namelens-8080.pid")

	// Write invalid content
	err := os.WriteFile(pidPath, []byte("not-a-number\n"), 0600)
	require.NoError(t, err)

	pf := &PIDFile{Path: pidPath, Port: 8080}
	_, err = pf.Read()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid PID")
}

func TestListPIDFiles(t *testing.T) {
	dir := t.TempDir()

	// Create some PID files
	for _, port := range []int{8080, 9000, 9001} {
		pf, err := NewPIDFile(port, dir)
		require.NoError(t, err)
		err = pf.Write(port) // Use port as PID for simplicity
		require.NoError(t, err)
	}

	// Create a non-PID file (should be ignored)
	err := os.WriteFile(filepath.Join(dir, "other.txt"), []byte("test"), 0600)
	require.NoError(t, err)

	// List PID files
	pidFiles, err := ListPIDFiles(dir)
	require.NoError(t, err)
	assert.Len(t, pidFiles, 3)

	// Verify ports
	ports := make(map[int]bool)
	for _, pf := range pidFiles {
		ports[pf.Port] = true
	}
	assert.True(t, ports[8080])
	assert.True(t, ports[9000])
	assert.True(t, ports[9001])
}

func TestListPIDFiles_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	pidFiles, err := ListPIDFiles(dir)
	require.NoError(t, err)
	assert.Empty(t, pidFiles)
}

func TestListPIDFiles_NonExistentDir(t *testing.T) {
	pidFiles, err := ListPIDFiles("/nonexistent/path/that/does/not/exist")
	require.NoError(t, err)
	assert.Nil(t, pidFiles)
}

func TestDefaultPIDDir(t *testing.T) {
	dir, err := DefaultPIDDir()
	require.NoError(t, err)

	// Should use XDG data dir pattern with /run/ subdirectory
	assert.Contains(t, dir, "namelens")
	assert.True(t, strings.HasSuffix(dir, "run"), "PID dir should end with /run")
}
