package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerifyDirWritable(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, verifyDirWritable(root))

	locked := filepath.Join(root, "locked")
	require.NoError(t, os.MkdirAll(locked, 0500))
	err := verifyDirWritable(locked)
	require.Error(t, err)
}
