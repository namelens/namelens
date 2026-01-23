package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestThumbnailPath(t *testing.T) {
	require.Equal(t, "/out/name.thumbnail.jpg", thumbnailPath("/out", "name.png", "thumbnail", "jpeg"))
	require.Equal(t, "/out/name.thumbnail.png", thumbnailPath("/out", "name.jpeg", "thumbnail", "png"))
}
