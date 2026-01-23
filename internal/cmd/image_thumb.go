package cmd

import (
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/image/draw"
)

var imageThumbCmd = &cobra.Command{
	Use:   "thumb",
	Short: "Generate thumbnails for images",
	Long:  "Generate smaller thumbnail images (png/jpeg) for easier review and agent ingestion.",
	RunE:  runImageThumb,
}

func init() {
	imageCmd.AddCommand(imageThumbCmd)

	imageThumbCmd.Flags().String("in-dir", "", "Input directory containing images")
	imageThumbCmd.Flags().String("out-dir", "", "Output directory for thumbnails (defaults to in-dir)")
	imageThumbCmd.Flags().Int("max-size", 256, "Max thumbnail dimension (64-1024)")
	imageThumbCmd.Flags().String("format", "jpeg", "Thumbnail format: jpeg or png")
	imageThumbCmd.Flags().Int("jpeg-quality", 80, "JPEG quality (1-100)")
	imageThumbCmd.Flags().String("suffix", "thumbnail", "Filename suffix (e.g. 'thumbnail' -> name.thumbnail.jpg)")
}

func runImageThumb(cmd *cobra.Command, _ []string) error {
	inDir, _ := cmd.Flags().GetString("in-dir")
	outDir, _ := cmd.Flags().GetString("out-dir")
	maxSize, _ := cmd.Flags().GetInt("max-size")
	format, _ := cmd.Flags().GetString("format")
	jpegQuality, _ := cmd.Flags().GetInt("jpeg-quality")
	suffix, _ := cmd.Flags().GetString("suffix")

	inDir = strings.TrimSpace(inDir)
	outDir = strings.TrimSpace(outDir)
	format = strings.ToLower(strings.TrimSpace(format))
	suffix = strings.TrimSpace(suffix)

	if inDir == "" {
		return errors.New("--in-dir is required")
	}
	if outDir == "" {
		outDir = inDir
	}
	if maxSize < 64 || maxSize > 1024 {
		return errors.New("--max-size must be between 64 and 1024")
	}
	if suffix == "" {
		suffix = "thumbnail"
	}

	absIn, err := filepath.Abs(inDir)
	if err != nil {
		absIn = inDir
	}
	absOut, err := ensureOutDir(outDir)
	if err != nil {
		return err
	}
	if err := verifyDirWritable(absOut); err != nil {
		return err
	}

	entries, err := os.ReadDir(absIn)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		lower := strings.ToLower(name)
		if !strings.HasSuffix(lower, ".png") && !strings.HasSuffix(lower, ".jpg") && !strings.HasSuffix(lower, ".jpeg") {
			continue
		}
		if strings.HasSuffix(lower, "."+strings.ToLower(suffix)+".png") ||
			strings.HasSuffix(lower, "."+strings.ToLower(suffix)+".jpg") ||
			strings.HasSuffix(lower, "."+strings.ToLower(suffix)+".jpeg") {
			continue
		}

		inPath := filepath.Join(absIn, name)
		outPath := thumbnailPath(absOut, name, suffix, format)
		if err := writeThumbnail(inPath, outPath, maxSize, format, jpegQuality); err != nil {
			return fmt.Errorf("thumbnail %s: %w", name, err)
		}
	}

	return nil
}

func thumbnailPath(outDir, filename, suffix, format string) string {
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	ext := "jpg"
	if format == "png" {
		ext = "png"
	}
	return filepath.Join(outDir, fmt.Sprintf("%s.%s.%s", base, suffix, ext))
}

func writeThumbnail(inPath, outPath string, maxSize int, format string, jpegQuality int) error {
	inFile, err := os.Open(inPath)
	if err != nil {
		return err
	}
	defer inFile.Close() // nolint:errcheck

	srcImg, _, err := image.Decode(inFile)
	if err != nil {
		return err
	}

	bounds := srcImg.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return errors.New("invalid image dimensions")
	}

	scale := float64(maxSize) / float64(max(width, height))
	if scale > 1 {
		scale = 1
	}
	newW := int(float64(width) * scale)
	newH := int(float64(height) * scale)
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), srcImg, bounds, draw.Over, nil)

	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close() // nolint:errcheck

	return encodeImage(outFile, dst, format, jpegQuality)
}

func encodeImage(w io.Writer, img image.Image, format string, jpegQuality int) error {
	switch format {
	case "png":
		return png.Encode(w, img)
	case "jpeg", "jpg", "":
		q := jpegQuality
		if q < 1 {
			q = 1
		}
		if q > 100 {
			q = 100
		}
		return jpeg.Encode(w, img, &jpeg.Options{Quality: q})
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
