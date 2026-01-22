package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/namelens/namelens/internal/ailink"
	"github.com/namelens/namelens/internal/ailink/driver"
	"github.com/namelens/namelens/internal/config"
	"github.com/namelens/namelens/internal/observability"

	"go.uber.org/zap"
)

var markCmd = &cobra.Command{
	Use:   "mark <name>",
	Short: "Generate logo/mark directions for a name",
	Long:  "Generate early-stage brand mark/logo directions and images for a finalist name (opt-in).",
	Args:  cobra.ExactArgs(1),
	RunE:  runMark,
}

func init() {
	rootCmd.AddCommand(markCmd)

	markCmd.Flags().String("prompt", "brand-mark", "Prompt slug to use")
	markCmd.Flags().String("depth", "quick", "Generation depth: quick, deep")
	markCmd.Flags().Int("count", 3, "Number of mark images to generate")
	markCmd.Flags().String("size", "1024x1024", "Image size (e.g. 1024x1024)")
	markCmd.Flags().String("format", "png", "Output format: png, jpeg, webp")
	markCmd.Flags().String("quality", "auto", "Quality: auto, low, medium, high")
	markCmd.Flags().String("background", "auto", "Background: auto, transparent, opaque")
	markCmd.Flags().String("out-dir", "", "Write images to a directory (required)")
	markCmd.Flags().String("model", "", "Text model override")
	markCmd.Flags().String("image-provider-role", "brand-mark-image", "Role key used for image provider routing")
	markCmd.Flags().String("image-model", "", "Image model override (default from provider models.image, else models.default)")
}

func runMark(cmd *cobra.Command, args []string) error {
	name := strings.TrimSpace(args[0])
	if name == "" {
		return errors.New("name is required")
	}

	promptSlug, _ := cmd.Flags().GetString("prompt")
	depth, _ := cmd.Flags().GetString("depth")
	count, _ := cmd.Flags().GetInt("count")
	size, _ := cmd.Flags().GetString("size")
	format, _ := cmd.Flags().GetString("format")
	quality, _ := cmd.Flags().GetString("quality")
	background, _ := cmd.Flags().GetString("background")
	outDir, _ := cmd.Flags().GetString("out-dir")
	modelOverride, _ := cmd.Flags().GetString("model")
	imageProviderRole, _ := cmd.Flags().GetString("image-provider-role")
	imageModelOverride, _ := cmd.Flags().GetString("image-model")

	promptSlug = strings.TrimSpace(promptSlug)
	if promptSlug == "" {
		return errors.New("prompt is required")
	}

	outDir = strings.TrimSpace(outDir)
	if outDir == "" {
		return errors.New("--out-dir is required")
	}

	absOutDir, err := ensureOutDir(outDir)
	if err != nil {
		return err
	}
	if err := verifyDirWritable(absOutDir); err != nil {
		return err
	}

	ctx := cmd.Context()
	cfg, err := config.Load(ctx)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	registry, err := buildPromptRegistry(cfg)
	if err != nil {
		return fmt.Errorf("loading prompts: %w", err)
	}
	promptDef, err := registry.Get(promptSlug)
	if err != nil {
		return fmt.Errorf("prompt not found: %w", err)
	}

	providers := ailink.NewRegistry(cfg.AILink)
	resolvedText, err := providers.ResolveWithDepth(promptSlug, promptDef, modelOverride, depth)
	if err != nil {
		return fmt.Errorf("resolving text provider: %w", err)
	}

	imageProviderRole = strings.TrimSpace(imageProviderRole)
	if imageProviderRole == "" {
		imageProviderRole = "brand-mark-image"
	}
	resolvedImage, err := providers.ResolveWithDepth(imageProviderRole, promptDef, "", depth)
	if err != nil {
		return fmt.Errorf("resolving image provider: %w", err)
	}

	gen, ok := resolvedImage.Driver.(driver.ImageGenerator)
	if !ok {
		return fmt.Errorf("provider %q does not support image generation", resolvedImage.Driver.Name())
	}

	catalog, err := buildSchemaCatalog()
	if err != nil {
		return fmt.Errorf("loading schemas: %w", err)
	}
	service := &ailink.Service{Providers: providers, Registry: registry, Catalog: catalog}
	_ = service

	imageModel := strings.TrimSpace(imageModelOverride)
	if imageModel == "" {
		if cfg.AILink.Providers != nil {
			if providerCfg, ok := cfg.AILink.Providers[resolvedImage.ProviderID]; ok {
				if providerCfg.Models != nil {
					imageModel = strings.TrimSpace(providerCfg.Models["image"])
					if imageModel == "" {
						imageModel = strings.TrimSpace(providerCfg.Models["default"])
					}
				}
			}
		}
	}
	if imageModel == "" {
		imageModel = resolvedImage.Model
	}

	// Step 1: generate mark directions + per-image prompts (schema-validated).
	markJSON, genErr, _ := runReviewGenerate(ctx, cfg, nil, promptSlug, name, depth, resolvedText.Model, map[string]string{"name": name}, false)
	if genErr != nil {
		return fmt.Errorf("mark prompt failed: %s: %s", genErr.Code, genErr.Message)
	}
	if len(markJSON) == 0 {
		return errors.New("mark prompt returned empty response")
	}

	var parsed struct {
		Name  string `json:"name"`
		Marks []struct {
			Label       string `json:"label"`
			Description string `json:"description"`
			ImagePrompt string `json:"image_prompt"`
		} `json:"marks"`
	}
	if err := json.Unmarshal(markJSON, &parsed); err != nil {
		return fmt.Errorf("decode mark prompt response: %w", err)
	}
	if len(parsed.Marks) == 0 {
		return errors.New("mark prompt returned no marks")
	}

	// Step 2: generate images for top N marks.
	limit := count
	if limit <= 0 {
		limit = 1
	}
	if limit > len(parsed.Marks) {
		limit = len(parsed.Marks)
	}

	for i := 0; i < limit; i++ {
		mark := parsed.Marks[i]
		imgResp, err := gen.GenerateImage(ctx, &driver.ImageRequest{
			Model:        imageModel,
			Prompt:       mark.ImagePrompt,
			Count:        1,
			Size:         size,
			OutputFormat: format,
			Quality:      quality,
			Background:   background,
			PromptSlug:   promptSlug,
		})
		if err != nil {
			return fmt.Errorf("image generation failed: %w", err)
		}
		if imgResp == nil || len(imgResp.Images) == 0 {
			return errors.New("image generation returned no images")
		}

		filename := fmt.Sprintf("%s_brand-mark_%02d.%s", sanitizeFilename(name), i+1, strings.ToLower(strings.TrimSpace(format)))
		path := filepath.Join(absOutDir, filename)
		block := imgResp.Images[0]
		if len(block.Data) == 0 {
			return errors.New("image response missing bytes")
		}
		if err := os.WriteFile(path, block.Data, 0644); err != nil {
			return fmt.Errorf("write image: %w", err)
		}

		observability.CLILogger.Info("Wrote brand mark image", zap.String("path", path), zap.String("label", mark.Label))
	}

	return nil
}

func verifyDirWritable(dir string) error {
	probe := filepath.Join(dir, ".namelens-write-test")
	f, err := os.OpenFile(probe, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("output directory not writable: %w", err)
	}
	if err := f.Close(); err != nil {
		return err
	}
	_ = os.Remove(probe)
	return nil
}
