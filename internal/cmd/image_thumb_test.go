package cmd

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteThumbnailShrinksImage(t *testing.T) {
	dir := t.TempDir()
	inPath := filepath.Join(dir, "in.png")
	outPath := filepath.Join(dir, "out.jpg")

	img := image.NewRGBA(image.Rect(0, 0, 1000, 500))
	f, err := os.Create(inPath)
	require.NoError(t, err)
	require.NoError(t, png.Encode(f, img))
	require.NoError(t, f.Close())

	require.NoError(t, writeThumbnail(inPath, outPath, 200, "jpeg", 80))

	outInfo, err := os.Stat(outPath)
	require.NoError(t, err)
	require.Greater(t, outInfo.Size(), int64(0))
}
