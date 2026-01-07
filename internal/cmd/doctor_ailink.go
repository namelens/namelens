package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/namelens/namelens/internal/ailink"
	"github.com/namelens/namelens/internal/ailink/prompt"
	"github.com/namelens/namelens/internal/config"
	"github.com/namelens/namelens/internal/observability"
)

var (
	doctorAILinkRole  string
	doctorAILinkModel string
)

var doctorAILinkCmd = &cobra.Command{
	Use:   "ailink [prompt-slug]",
	Short: "Inspect AILink provider resolution",
	Long:  "Resolve an AILink role/prompt to a provider instance and show credential selection.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cmd.Context())
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		promptSlug := ""
		if len(args) > 0 {
			promptSlug = strings.TrimSpace(args[0])
		}
		if promptSlug == "" {
			promptSlug = strings.TrimSpace(cfg.Expert.DefaultPrompt)
		}
		if promptSlug == "" {
			promptSlug = "name-availability"
		}

		role := strings.TrimSpace(doctorAILinkRole)
		if role == "" {
			role = promptSlug
		}

		promptRegistry, err := buildPromptRegistry(cfg)
		if err != nil {
			return fmt.Errorf("load prompt registry: %w", err)
		}

		promptDef, err := promptRegistry.Get(promptSlug)
		if err != nil {
			return fmt.Errorf("prompt not found: %w", err)
		}

		providers := ailink.NewRegistry(cfg.AILink)
		resolved, err := providers.Resolve(role, promptDef, doctorAILinkModel)
		if err != nil {
			return fmt.Errorf("resolve provider: %w", err)
		}

		providerCfg := resolved.Provider
		resolutionSource, routingTarget := describeAILinkResolution(cfg, role)

		observability.CLILogger.Info("AILink Resolution")
		observability.CLILogger.Info(fmt.Sprintf("  Role:        %s", role))
		observability.CLILogger.Info(fmt.Sprintf("  Prompt:       %s", promptSlug))
		observability.CLILogger.Info(fmt.Sprintf("  Source:       %s", resolutionSource))
		if routingTarget != "" {
			observability.CLILogger.Info(fmt.Sprintf("  Routing:     %s -> %s", role, routingTarget))
		}
		observability.CLILogger.Info(fmt.Sprintf("  Provider ID:  %s", resolved.ProviderID))
		observability.CLILogger.Info(fmt.Sprintf("  ai_provider:  %s", providerCfg.AIProvider))
		observability.CLILogger.Info(fmt.Sprintf("  base_url:     %s", providerCfg.BaseURL))

		configuredModel := ""
		if providerCfg.Models != nil {
			configuredModel = strings.TrimSpace(providerCfg.Models["default"])
		}
		promptPreferred := firstPreferredModel(promptDef)

		modelSource := "unknown"
		switch {
		case strings.TrimSpace(doctorAILinkModel) != "":
			modelSource = "cli_override"
		case promptPreferred != "":
			modelSource = "prompt_preferred_models"
		case configuredModel != "":
			modelSource = "provider.models.default"
		}

		observability.CLILogger.Info(fmt.Sprintf("  model:        %s", resolved.Model))
		observability.CLILogger.Info(fmt.Sprintf("  model_source: %s", modelSource))
		if modelSource == "prompt_preferred_models" {
			observability.CLILogger.Info(fmt.Sprintf("  prompt.preferred_models[0]: %s", promptPreferred))
			if configuredModel != "" {
				observability.CLILogger.Info(fmt.Sprintf("  provider.models.default:    %s", configuredModel))
			}
		}
		observability.CLILogger.Info("")

		policy := strings.TrimSpace(providerCfg.SelectionPolicy)
		if policy == "" {
			policy = "priority"
		}
		observability.CLILogger.Info("Credential Selection")
		observability.CLILogger.Info(fmt.Sprintf("  selection_policy:   %s", policy))
		if strings.TrimSpace(providerCfg.DefaultCredential) != "" {
			observability.CLILogger.Info(fmt.Sprintf("  default_credential: %s", providerCfg.DefaultCredential))
		}
		observability.CLILogger.Info(fmt.Sprintf("  selected.label:     %s", resolved.Credential.Label))
		observability.CLILogger.Info(fmt.Sprintf("  selected.priority:  %d", resolved.Credential.Priority))
		if strings.TrimSpace(resolved.Credential.APIKey) != "" {
			observability.CLILogger.Info("  selected.api_key:   (set)")
		} else {
			observability.CLILogger.Info("  selected.api_key:   (not set)")
			observability.CLILogger.Warn("Selected credential has no API key", zap.String("provider", resolved.ProviderID))
		}

		return nil
	},
}

func describeAILinkResolution(cfg *config.Config, role string) (source string, routingTarget string) {
	if cfg == nil {
		return "config missing", ""
	}

	role = strings.TrimSpace(role)
	if role != "" && cfg.AILink.Routing != nil {
		routingTarget = strings.TrimSpace(cfg.AILink.Routing[role])
		if routingTarget != "" {
			return "routing", routingTarget
		}
	}

	for _, providerCfg := range cfg.AILink.Providers {
		if !providerCfg.Enabled {
			continue
		}
		for _, r := range providerCfg.Roles {
			if strings.EqualFold(strings.TrimSpace(r), role) {
				return "roles", ""
			}
		}
	}

	if strings.TrimSpace(cfg.AILink.DefaultProvider) != "" {
		return "default_provider", ""
	}

	enabledCount := 0
	for _, providerCfg := range cfg.AILink.Providers {
		if providerCfg.Enabled {
			enabledCount++
		}
	}
	if enabledCount == 1 {
		return "only_enabled_provider", ""
	}

	return "unknown", ""
}

func firstPreferredModel(promptDef *prompt.Prompt) string {
	if promptDef == nil {
		return ""
	}

	value, ok := promptDef.Config.ProviderHints["preferred_models"]
	if !ok || value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []string:
		if len(typed) == 0 {
			return ""
		}
		return strings.TrimSpace(typed[0])
	case []any:
		for _, item := range typed {
			if s, ok := item.(string); ok {
				if candidate := strings.TrimSpace(s); candidate != "" {
					return candidate
				}
			}
		}
		return ""
	default:
		return ""
	}
}

func init() {
	doctorCmd.AddCommand(doctorAILinkCmd)

	doctorAILinkCmd.Flags().StringVar(&doctorAILinkRole, "role", "", "Role to resolve (defaults to prompt slug)")
	doctorAILinkCmd.Flags().StringVar(&doctorAILinkModel, "model", "", "Model override (defaults to prompt/provider config)")
}
