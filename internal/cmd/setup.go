package cmd

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	"github.com/namelens/namelens/internal/config"
)

// providerInfo describes a supported AI provider for the setup wizard.
type providerInfo struct {
	Slug         string // CLI slug: xai, openai, anthropic
	DisplayName  string // Human-readable name
	InstanceID   string // Config instance key (e.g. namelens-xai)
	AIProvider   string // ai_provider field value
	BaseURL      string
	DefaultModel string
	AuthHeader   string // "bearer" or "x-api-key"
	TestEndpoint bool   // Whether GET /models is supported for auth test
}

var providerTable = []providerInfo{
	{
		Slug:         "xai",
		DisplayName:  "xAI (Grok)",
		InstanceID:   "namelens-xai",
		AIProvider:   "xai",
		BaseURL:      "https://api.x.ai/v1",
		DefaultModel: "grok-4-1-fast-reasoning",
		AuthHeader:   "bearer",
		TestEndpoint: true,
	},
	{
		Slug:         "openai",
		DisplayName:  "OpenAI (GPT)",
		InstanceID:   "namelens-openai",
		AIProvider:   "openai",
		BaseURL:      "https://api.openai.com/v1",
		DefaultModel: "gpt-4o",
		AuthHeader:   "bearer",
		TestEndpoint: true,
	},
	{
		Slug:         "anthropic",
		DisplayName:  "Anthropic (Claude)",
		InstanceID:   "namelens-anthropic",
		AIProvider:   "anthropic",
		BaseURL:      "https://api.anthropic.com/v1",
		DefaultModel: "claude-sonnet-4-5-20250929",
		AuthHeader:   "x-api-key",
		TestEndpoint: false,
	},
}

var (
	setupProvider string
	setupAPIKey   string
	setupNoTest   bool
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure an AI backend for expert analysis",
	Long: `Interactive setup wizard for configuring an AI provider.

Guides you through selecting a provider (xAI, OpenAI, or Anthropic),
entering your API key, testing the connection, and writing the config file.

Non-interactive usage:
  namelens setup --provider xai --api-key YOUR_KEY
  namelens setup --provider anthropic --api-key YOUR_KEY --no-test`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := strings.TrimSpace(cfgFile)
		if configPath == "" {
			configPath = config.DefaultConfigPath()
		}
		if configPath == "" {
			return fmt.Errorf("cannot resolve config path")
		}
		return runSetup(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), os.Stdin, configPath)
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)

	setupCmd.Flags().StringVar(&setupProvider, "provider", "", "Provider slug: xai, openai, anthropic")
	setupCmd.Flags().StringVar(&setupAPIKey, "api-key", "", "API key (non-interactive)")
	setupCmd.Flags().BoolVar(&setupNoTest, "no-test", false, "Skip connection test")
}

func runSetup(ctx context.Context, stdout io.Writer, stderr io.Writer, stdin io.Reader, configPath string) error {
	// Create a shared buffered reader so piped input doesn't lose data
	// between selectProvider and getAPIKey reads.
	reader := bufio.NewReader(stdin)

	// Step 1: Detect existing config
	existingProvider := detectExistingProvider(configPath)
	if existingProvider != "" {
		_, _ = fmt.Fprintf(stdout, "Current AI provider: %s\n", existingProvider)
		if setupProvider == "" && setupAPIKey == "" {
			_, _ = fmt.Fprintln(stdout, "Reconfiguring will update the AI backend settings.")
			_, _ = fmt.Fprintln(stdout, "")
		}
	}

	// Step 2: Select provider
	provider, err := selectProvider(stdout, reader)
	if err != nil {
		return err
	}

	// Step 3: Get API key
	apiKey, err := getAPIKey(stdout, stdin, reader, provider)
	if err != nil {
		return err
	}

	// Step 4: Connection test
	if !setupNoTest {
		_, _ = fmt.Fprintln(stdout, "")
		_, _ = fmt.Fprintln(stdout, "Testing connection...")
		err := runSetupConnectionTest(ctx, stdout, provider, apiKey)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "Connection test failed: %s\n", err)
			_, _ = fmt.Fprintln(stderr, "Config will still be written. Run 'namelens doctor ailink connectivity' to debug.")
			_, _ = fmt.Fprintln(stderr, "")
		} else {
			_, _ = fmt.Fprintln(stdout, "Connection test passed.")
		}
	}

	// Step 5: Write config
	_, _ = fmt.Fprintln(stdout, "")
	err = writeSetupConfig(configPath, provider, apiKey)
	if err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	// Step 6: Success message
	_, _ = fmt.Fprintf(stdout, "Config written to %s\n", configPath)
	_, _ = fmt.Fprintln(stdout, "")
	_, _ = fmt.Fprintf(stdout, "Provider:  %s\n", provider.DisplayName)
	_, _ = fmt.Fprintf(stdout, "Model:     %s\n", provider.DefaultModel)
	_, _ = fmt.Fprintf(stdout, "API key:   %s\n", maskKey(apiKey))
	_, _ = fmt.Fprintln(stdout, "")
	_, _ = fmt.Fprintln(stdout, "Next steps:")
	_, _ = fmt.Fprintln(stdout, "  namelens check <name> --expert    Run an expert analysis")
	_, _ = fmt.Fprintln(stdout, "  namelens doctor                   Verify installation")
	_, _ = fmt.Fprintln(stdout, "")

	return nil
}

// selectProvider resolves the provider from --provider flag or interactive menu.
func selectProvider(stdout io.Writer, reader *bufio.Reader) (*providerInfo, error) {
	if setupProvider != "" {
		return lookupProvider(setupProvider)
	}

	// Interactive menu
	_, _ = fmt.Fprintln(stdout, "Select an AI provider:")
	_, _ = fmt.Fprintln(stdout, "")
	for i, p := range providerTable {
		_, _ = fmt.Fprintf(stdout, "  %d) %s\n", i+1, p.DisplayName)
	}
	_, _ = fmt.Fprintln(stdout, "")
	_, _ = fmt.Fprint(stdout, "Enter choice [1-3]: ")

	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("read input: %w", err)
	}
	line = strings.TrimSpace(line)

	idx, err := strconv.Atoi(line)
	if err != nil || idx < 1 || idx > len(providerTable) {
		return nil, fmt.Errorf("invalid choice: %q (enter 1-%d)", line, len(providerTable))
	}

	p := providerTable[idx-1]
	return &p, nil
}

// lookupProvider finds a provider by slug. Returns error for unknown slugs.
func lookupProvider(slug string) (*providerInfo, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	for _, p := range providerTable {
		if p.Slug == slug {
			return &p, nil
		}
	}
	valid := make([]string, 0, len(providerTable))
	for _, p := range providerTable {
		valid = append(valid, p.Slug)
	}
	return nil, fmt.Errorf("unknown provider %q (valid: %s)", slug, strings.Join(valid, ", "))
}

// getAPIKey gets the API key from --api-key flag or secure interactive input.
// stdin is the raw reader (for terminal detection), reader is the shared
// buffered reader (for piped input fallback).
func getAPIKey(stdout io.Writer, stdin io.Reader, reader *bufio.Reader, provider *providerInfo) (string, error) {
	if setupAPIKey != "" {
		return setupAPIKey, nil
	}

	_, _ = fmt.Fprintf(stdout, "\nEnter your %s API key: ", provider.DisplayName)

	// Try secure no-echo input if stdin is a terminal
	if f, ok := stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		key, err := term.ReadPassword(int(f.Fd()))
		_, _ = fmt.Fprintln(stdout) // newline after hidden input
		if err != nil {
			return "", fmt.Errorf("read API key: %w", err)
		}
		k := strings.TrimSpace(string(key))
		if k == "" {
			return "", fmt.Errorf("API key cannot be empty")
		}
		return k, nil
	}

	// Fallback for non-terminal (piped input) — use the shared reader
	// so data buffered from selectProvider isn't lost.
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read API key: %w", err)
	}
	k := strings.TrimSpace(line)
	if k == "" {
		return "", fmt.Errorf("API key cannot be empty")
	}
	return k, nil
}

// runSetupConnectionTest performs layered DNS → TCP → TLS → HTTP auth checks.
func runSetupConnectionTest(ctx context.Context, stdout io.Writer, provider *providerInfo, apiKey string) error {
	timeout := 10 * time.Second

	u, err := url.Parse(provider.BaseURL)
	if err != nil {
		return fmt.Errorf("parse base URL: %w", err)
	}
	host := u.Hostname()
	port := 443
	if u.Port() != "" {
		if p, err := strconv.Atoi(u.Port()); err == nil {
			port = p
		}
	}

	// DNS
	_, _ = fmt.Fprintf(stdout, "  DNS resolve %s... ", host)
	dnsCtx, dnsCancel := context.WithTimeout(ctx, timeout)
	defer dnsCancel()
	ips, err := net.DefaultResolver.LookupIPAddr(dnsCtx, host)
	if err != nil {
		_, _ = fmt.Fprintln(stdout, "FAIL")
		return fmt.Errorf("DNS: %w", err)
	}
	_, _ = fmt.Fprintf(stdout, "ok (%d addresses)\n", len(ips))

	// TCP
	_, _ = fmt.Fprintf(stdout, "  TCP connect %s:%d... ", host, port)
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		_, _ = fmt.Fprintln(stdout, "FAIL")
		return fmt.Errorf("TCP: %w", err)
	}
	_, _ = fmt.Fprintln(stdout, "ok")

	// TLS
	_, _ = fmt.Fprintf(stdout, "  TLS handshake %s... ", host)
	_ = conn.SetDeadline(time.Now().Add(timeout))
	tlsConn := tls.Client(conn, &tls.Config{ServerName: host})
	err = tlsConn.HandshakeContext(ctx)
	if err != nil {
		_ = conn.Close()
		_, _ = fmt.Fprintln(stdout, "FAIL")
		return fmt.Errorf("TLS: %w", err)
	}
	_ = tlsConn.Close()
	_, _ = fmt.Fprintln(stdout, "ok")

	// HTTP auth (skip for providers that don't support GET /models)
	if !provider.TestEndpoint {
		_, _ = fmt.Fprintf(stdout, "  HTTP auth... skipped (%s has no test endpoint)\n", provider.AIProvider)
		return nil
	}

	_, _ = fmt.Fprint(stdout, "  HTTP auth check... ")
	modelsURL := strings.TrimRight(provider.BaseURL, "/") + "/models"
	httpCtx, httpCancel := context.WithTimeout(ctx, timeout)
	defer httpCancel()

	req, err := http.NewRequestWithContext(httpCtx, http.MethodGet, modelsURL, nil)
	if err != nil {
		_, _ = fmt.Fprintln(stdout, "FAIL")
		return fmt.Errorf("HTTP: %w", err)
	}

	if provider.AuthHeader == "x-api-key" {
		req.Header.Set("x-api-key", apiKey)
	} else {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		_, _ = fmt.Fprintln(stdout, "FAIL")
		return fmt.Errorf("HTTP: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == 200 {
		_, _ = fmt.Fprintln(stdout, "ok")
		return nil
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		_, _ = fmt.Fprintln(stdout, "FAIL")
		return fmt.Errorf("HTTP auth: %s (check your API key)", resp.Status)
	}
	_, _ = fmt.Fprintln(stdout, "FAIL")
	return fmt.Errorf("HTTP: %s", resp.Status)
}

// detectExistingProvider reads the config file and extracts the current default provider.
func detectExistingProvider(configPath string) string {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return ""
	}
	ailinkRaw, ok := raw["ailink"]
	if !ok {
		return ""
	}
	ailinkMap, ok := ailinkRaw.(map[string]any)
	if !ok {
		return ""
	}
	dp, ok := ailinkMap["default_provider"]
	if !ok {
		return ""
	}
	s, ok := dp.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

// writeSetupConfig writes or merges provider config into the config file.
func writeSetupConfig(configPath string, provider *providerInfo, apiKey string) error {
	var raw map[string]any

	// Read existing config if present
	data, err := os.ReadFile(configPath)
	if err == nil {
		if err := yaml.Unmarshal(data, &raw); err != nil {
			// Invalid YAML — start fresh
			raw = nil
		}
	}
	if raw == nil {
		raw = make(map[string]any)
	}

	// Build ailink section
	ailinkMap := getOrCreateMap(raw, "ailink")
	ailinkMap["default_provider"] = provider.InstanceID

	providersMap := getOrCreateMap(ailinkMap, "providers")

	// Merge into existing provider entry to preserve custom model tiers
	// and other provider-specific settings (e.g. fast, reasoning models).
	providerEntry := getOrCreateMap(providersMap, provider.InstanceID)
	providerEntry["enabled"] = true
	providerEntry["ai_provider"] = provider.AIProvider
	providerEntry["base_url"] = provider.BaseURL

	// Merge models: set default but preserve existing custom tiers
	modelsMap := getOrCreateMap(providerEntry, "models")
	modelsMap["default"] = provider.DefaultModel

	// Replace credentials with the new key
	providerEntry["credentials"] = []any{
		map[string]any{
			"label":    "default",
			"priority": 0,
			"api_key":  apiKey,
			"enabled":  true,
		},
	}

	// Set expert section
	expertMap := getOrCreateMap(raw, "expert")
	expertMap["enabled"] = true
	if _, ok := expertMap["default_prompt"]; !ok {
		expertMap["default_prompt"] = "name-availability"
	}

	// Ensure config directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	// Marshal and write
	out, err := yaml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(configPath, out, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// getOrCreateMap returns an existing map[string]any at key, or creates one.
func getOrCreateMap(parent map[string]any, key string) map[string]any {
	if existing, ok := parent[key]; ok {
		if m, ok := existing.(map[string]any); ok {
			return m
		}
	}
	m := make(map[string]any)
	parent[key] = m
	return m
}
