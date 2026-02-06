package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestProviderTableCompleteness(t *testing.T) {
	if len(providerTable) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(providerTable))
	}

	for _, p := range providerTable {
		t.Run(p.Slug, func(t *testing.T) {
			if p.Slug == "" {
				t.Error("slug is empty")
			}
			if p.DisplayName == "" {
				t.Error("display name is empty")
			}
			if p.InstanceID == "" {
				t.Error("instance ID is empty")
			}
			if p.AIProvider == "" {
				t.Error("ai_provider is empty")
			}
			if p.BaseURL == "" {
				t.Error("base_url is empty")
			}
			if p.DefaultModel == "" {
				t.Error("default model is empty")
			}
			if p.AuthHeader == "" {
				t.Error("auth header is empty")
			}
		})
	}
}

func TestLookupProvider(t *testing.T) {
	tests := []struct {
		slug    string
		wantErr bool
		wantAI  string
	}{
		{"xai", false, "xai"},
		{"openai", false, "openai"},
		{"anthropic", false, "anthropic"},
		{"XAI", false, "xai"},
		{"  openai  ", false, "openai"},
		{"invalid", true, ""},
		{"", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			p, err := lookupProvider(tt.slug)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.AIProvider != tt.wantAI {
				t.Errorf("ai_provider = %q, want %q", p.AIProvider, tt.wantAI)
			}
		})
	}
}

func TestDetectExistingProvider(t *testing.T) {
	t.Run("no file", func(t *testing.T) {
		result := detectExistingProvider(filepath.Join(t.TempDir(), "nonexistent.yaml"))
		if result != "" {
			t.Errorf("expected empty, got %q", result)
		}
	})

	t.Run("valid yaml with provider", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		content := "ailink:\n  default_provider: namelens-xai\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		result := detectExistingProvider(path)
		if result != "namelens-xai" {
			t.Errorf("expected namelens-xai, got %q", result)
		}
	})

	t.Run("valid yaml without ailink", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		content := "domain:\n  whois_fallback:\n    enabled: true\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		result := detectExistingProvider(path)
		if result != "" {
			t.Errorf("expected empty, got %q", result)
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		if err := os.WriteFile(path, []byte(":::invalid"), 0644); err != nil {
			t.Fatal(err)
		}
		result := detectExistingProvider(path)
		if result != "" {
			t.Errorf("expected empty, got %q", result)
		}
	})
}

func TestWriteSetupConfig_Fresh(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	provider := &providerInfo{
		Slug:         "xai",
		InstanceID:   "namelens-xai",
		AIProvider:   "xai",
		BaseURL:      "https://api.x.ai/v1",
		DefaultModel: "grok-4-1-fast-reasoning",
	}

	err := writeSetupConfig(path, provider, "sk-test-key-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file exists with 0600 permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("permissions = %o, want 0600", perm)
	}

	// Read back and verify structure
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Check ailink.default_provider
	ailinkMap, ok := raw["ailink"].(map[string]any)
	if !ok {
		t.Fatal("ailink section missing")
	}
	if dp := ailinkMap["default_provider"]; dp != "namelens-xai" {
		t.Errorf("default_provider = %v, want namelens-xai", dp)
	}

	// Check provider entry
	providersMap, ok := ailinkMap["providers"].(map[string]any)
	if !ok {
		t.Fatal("providers section missing")
	}
	provEntry, ok := providersMap["namelens-xai"].(map[string]any)
	if !ok {
		t.Fatal("namelens-xai provider missing")
	}
	if provEntry["ai_provider"] != "xai" {
		t.Errorf("ai_provider = %v, want xai", provEntry["ai_provider"])
	}
	if provEntry["base_url"] != "https://api.x.ai/v1" {
		t.Errorf("base_url = %v", provEntry["base_url"])
	}
}

func TestWriteSetupConfig_Merge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	// Write initial config with domain section and existing provider
	initial := `domain:
  whois_fallback:
    enabled: true
ailink:
  default_provider: namelens-xai
  providers:
    namelens-xai:
      enabled: true
      ai_provider: xai
      base_url: https://api.x.ai/v1
      credentials:
        - label: default
          api_key: old-key
store:
  driver: libsql
`
	if err := os.WriteFile(path, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	// Add OpenAI provider
	provider := &providerInfo{
		Slug:         "openai",
		InstanceID:   "namelens-openai",
		AIProvider:   "openai",
		BaseURL:      "https://api.openai.com/v1",
		DefaultModel: "gpt-4o",
	}

	err := writeSetupConfig(path, provider, "sk-openai-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Verify domain section preserved
	domainMap, ok := raw["domain"].(map[string]any)
	if !ok {
		t.Fatal("domain section was lost")
	}
	whoisMap, ok := domainMap["whois_fallback"].(map[string]any)
	if !ok {
		t.Fatal("whois_fallback section was lost")
	}
	if whoisMap["enabled"] != true {
		t.Error("domain.whois_fallback.enabled was lost")
	}

	// Verify store section preserved
	if _, ok := raw["store"]; !ok {
		t.Fatal("store section was lost")
	}

	// Verify old provider preserved
	ailinkMap := raw["ailink"].(map[string]any)
	providersMap := ailinkMap["providers"].(map[string]any)
	if _, ok := providersMap["namelens-xai"]; !ok {
		t.Error("existing xai provider was lost")
	}

	// Verify new provider added
	if _, ok := providersMap["namelens-openai"]; !ok {
		t.Error("new openai provider not added")
	}

	// Verify default_provider updated
	if dp := ailinkMap["default_provider"]; dp != "namelens-openai" {
		t.Errorf("default_provider = %v, want namelens-openai", dp)
	}
}

func TestWriteSetupConfig_Overwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	// Write initial config with xai including custom model tiers
	initial := `ailink:
  default_provider: namelens-xai
  providers:
    namelens-xai:
      enabled: true
      ai_provider: xai
      base_url: https://api.x.ai/v1
      models:
        default: grok-4-1-fast-reasoning
        fast: grok-3-mini-fast
        reasoning: grok-4-1-fast-reasoning
      credentials:
        - label: default
          api_key: old-key
`
	if err := os.WriteFile(path, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	// Overwrite same provider with new key
	provider := &providerInfo{
		Slug:         "xai",
		InstanceID:   "namelens-xai",
		AIProvider:   "xai",
		BaseURL:      "https://api.x.ai/v1",
		DefaultModel: "grok-4-1-fast-reasoning",
	}

	err := writeSetupConfig(path, provider, "sk-new-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	ailinkMap := raw["ailink"].(map[string]any)
	providersMap := ailinkMap["providers"].(map[string]any)
	provEntry := providersMap["namelens-xai"].(map[string]any)

	// Verify credentials updated
	creds, ok := provEntry["credentials"].([]any)
	if !ok || len(creds) == 0 {
		t.Fatal("credentials missing")
	}
	cred := creds[0].(map[string]any)
	if cred["api_key"] != "sk-new-key" {
		t.Errorf("api_key = %v, want sk-new-key", cred["api_key"])
	}

	// BUG-9 regression: verify custom model tiers preserved
	modelsMap, ok := provEntry["models"].(map[string]any)
	if !ok {
		t.Fatal("models section missing after overwrite")
	}
	if modelsMap["fast"] != "grok-3-mini-fast" {
		t.Errorf("models.fast = %v, want grok-3-mini-fast (custom tier lost)", modelsMap["fast"])
	}
	if modelsMap["reasoning"] != "grok-4-1-fast-reasoning" {
		t.Errorf("models.reasoning = %v, want grok-4-1-fast-reasoning (custom tier lost)", modelsMap["reasoning"])
	}
	if modelsMap["default"] != "grok-4-1-fast-reasoning" {
		t.Errorf("models.default = %v, want grok-4-1-fast-reasoning", modelsMap["default"])
	}
}

func TestWriteSetupConfig_ExpertEnabled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	provider := &providerInfo{
		Slug:         "anthropic",
		InstanceID:   "namelens-anthropic",
		AIProvider:   "anthropic",
		BaseURL:      "https://api.anthropic.com/v1",
		DefaultModel: "claude-sonnet-4-5-20250929",
	}

	err := writeSetupConfig(path, provider, "sk-ant-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	expertMap, ok := raw["expert"].(map[string]any)
	if !ok {
		t.Fatal("expert section missing")
	}
	if expertMap["enabled"] != true {
		t.Error("expert.enabled not true")
	}
	if expertMap["default_prompt"] != "name-availability" {
		t.Errorf("expert.default_prompt = %v, want name-availability", expertMap["default_prompt"])
	}
}

func TestRunSetup_UsesExplicitConfigPath(t *testing.T) {
	// BUG-8 regression: setup must write to the provided config path,
	// not always to DefaultConfigPath().
	dir := t.TempDir()
	customPath := filepath.Join(dir, "custom", "config.yaml")

	// Save/restore global flags
	oldProvider := setupProvider
	oldAPIKey := setupAPIKey
	oldNoTest := setupNoTest
	defer func() {
		setupProvider = oldProvider
		setupAPIKey = oldAPIKey
		setupNoTest = oldNoTest
	}()

	setupProvider = "openai"
	setupAPIKey = "sk-test-custom-path"
	setupNoTest = true

	var stdout, stderr bytes.Buffer
	err := runSetup(context.Background(), &stdout, &stderr, strings.NewReader(""), customPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was written to customPath, not DefaultConfigPath
	if _, err := os.Stat(customPath); err != nil {
		t.Fatalf("config not written to custom path: %v", err)
	}

	data, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	ailinkMap := raw["ailink"].(map[string]any)
	if dp := ailinkMap["default_provider"]; dp != "namelens-openai" {
		t.Errorf("default_provider = %v, want namelens-openai", dp)
	}
}

func TestRunSetup_PipedInput(t *testing.T) {
	// BUG-10 regression: piped input with provider selection + API key
	// must not lose data across reads.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	// Save/restore global flags
	oldProvider := setupProvider
	oldAPIKey := setupAPIKey
	oldNoTest := setupNoTest
	defer func() {
		setupProvider = oldProvider
		setupAPIKey = oldAPIKey
		setupNoTest = oldNoTest
	}()

	// Non-interactive provider but piped API key
	setupProvider = ""
	setupAPIKey = ""
	setupNoTest = true

	// Simulate piped input: "2\nsk-test-pipe-key\n"
	piped := strings.NewReader("2\nsk-test-pipe-key\n")

	var stdout, stderr bytes.Buffer
	err := runSetup(context.Background(), &stdout, &stderr, piped, path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Verify provider 2 (OpenAI) was selected
	ailinkMap := raw["ailink"].(map[string]any)
	if dp := ailinkMap["default_provider"]; dp != "namelens-openai" {
		t.Errorf("default_provider = %v, want namelens-openai", dp)
	}

	// Verify API key was captured (not lost to buffering)
	providersMap := ailinkMap["providers"].(map[string]any)
	provEntry := providersMap["namelens-openai"].(map[string]any)
	creds := provEntry["credentials"].([]any)
	cred := creds[0].(map[string]any)
	if cred["api_key"] != "sk-test-pipe-key" {
		t.Errorf("api_key = %v, want sk-test-pipe-key (data lost across reads)", cred["api_key"])
	}
}
