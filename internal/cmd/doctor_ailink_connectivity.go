package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fulmenhq/gofulmen/ascii"
	"github.com/spf13/cobra"

	"github.com/namelens/namelens/internal/ailink"
	"github.com/namelens/namelens/internal/config"
	"github.com/namelens/namelens/internal/output"
)

var (
	doctorAILinkConnectivityRole        string
	doctorAILinkConnectivityProviderID  string
	doctorAILinkConnectivityTimeout     time.Duration
	doctorAILinkConnectivityQuiet       bool
	doctorAILinkConnectivityShowSecrets bool
	doctorAILinkConnectivityOutputRaw   string
)

type connectivityReport struct {
	Version     string                 `json:"version,omitempty"`
	Timestamp   string                 `json:"timestamp"`
	Input       connectivityInput      `json:"input"`
	Resolution  connectivityResolution `json:"resolution"`
	Environment connectivityEnv        `json:"environment,omitempty"`
	Checks      []connectivityCheck    `json:"checks"`
	Summary     connectivitySummary    `json:"summary"`
}

type connectivityInput struct {
	PromptSlug         string `json:"prompt_slug,omitempty"`
	Role               string `json:"role,omitempty"`
	ProviderIDOverride string `json:"provider_id_override,omitempty"`
	TimeoutSeconds     int    `json:"timeout_seconds"`
	Output             string `json:"output"`
	Quiet              bool   `json:"quiet,omitempty"`
	ShowSecrets        bool   `json:"show_secrets,omitempty"`
}

type connectivityResolution struct {
	ProviderID        string                 `json:"provider_id"`
	ResolutionSource  string                 `json:"resolution_source"`
	RoutingTarget     string                 `json:"routing_target,omitempty"`
	AIProvider        string                 `json:"ai_provider"`
	BaseURL           string                 `json:"base_url"`
	Host              string                 `json:"host"`
	Port              int                    `json:"port"`
	Model             string                 `json:"model,omitempty"`
	ModelSource       string                 `json:"model_source,omitempty"`
	PromptPreferred   string                 `json:"prompt_preferred_model,omitempty"`
	ProviderDefault   string                 `json:"provider_default_model,omitempty"`
	Credential        connectivityCredential `json:"credential,omitempty"`
	AdditionalDetails map[string]any         `json:"details,omitempty"`
}

type connectivityCredential struct {
	SelectionPolicy   string `json:"selection_policy,omitempty"`
	DefaultCredential string `json:"default_credential,omitempty"`
	SelectedLabel     string `json:"selected_label,omitempty"`
	SelectedPriority  int    `json:"selected_priority,omitempty"`
	APIKeyPresent     bool   `json:"api_key_present"`
	APIKeyHint        string `json:"api_key_hint,omitempty"`
}

type connectivityEnv struct {
	HTTPProxySet   bool `json:"http_proxy_set"`
	HTTPSProxySet  bool `json:"https_proxy_set"`
	NoProxySet     bool `json:"no_proxy_set"`
	NoProxyMatches bool `json:"no_proxy_matches_host"`
}

type connectivityCheck struct {
	Name      string               `json:"name"`
	OK        bool                 `json:"ok,omitempty"`
	Skipped   bool                 `json:"skipped,omitempty"`
	LatencyMS int64                `json:"latency_ms,omitempty"`
	Details   map[string]any       `json:"details,omitempty"`
	Error     *connectivityErrInfo `json:"error,omitempty"`
}

type connectivityErrInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type connectivitySummary struct {
	OK             bool     `json:"ok"`
	FailureLayer   string   `json:"failure_layer,omitempty"`
	Classification string   `json:"classification,omitempty"`
	Hints          []string `json:"hints,omitempty"`
}

var doctorAILinkConnectivityCmd = &cobra.Command{
	Use:   "connectivity [prompt-slug]",
	Short: "Diagnose provider reachability and auth",
	Long:  "Runs layered DNS/TCP/TLS/HTTP checks for the AILink route resolved from a prompt/role.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cmd.Context())
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		format, err := output.ParseFormat(doctorAILinkConnectivityOutputRaw)
		if err != nil {
			return err
		}
		if format != output.FormatJSON && format != output.FormatTable {
			return fmt.Errorf("unsupported output format for connectivity: %s", format)
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

		role := strings.TrimSpace(doctorAILinkConnectivityRole)
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

		ailinkCfg := cfg.AILink
		providerOverride := strings.TrimSpace(doctorAILinkConnectivityProviderID)
		if providerOverride != "" {
			if ailinkCfg.Routing == nil {
				ailinkCfg.Routing = map[string]string{}
			}
			ailinkCfg.Routing[role] = providerOverride
		}

		providers := ailink.NewRegistry(ailinkCfg)
		resolved, err := providers.Resolve(role, promptDef, "")
		if err != nil {
			return fmt.Errorf("resolve provider: %w", err)
		}

		resolutionSource, routingTarget := describeAILinkResolution(cfg, role)
		if providerOverride != "" {
			resolutionSource = "provider_id_override"
			routingTarget = providerOverride
		}

		promptPreferred := firstPreferredModel(promptDef)
		report, err := runConnectivity(cmd.Context(), promptSlug, role, promptPreferred, resolved, resolutionSource, routingTarget, format)
		if err != nil {
			return err
		}

		if doctorAILinkConnectivityQuiet {
			if report.Summary.OK {
				return nil
			}
			return fmt.Errorf("connectivity check failed (%s)", report.Summary.Classification)
		}

		if format == output.FormatJSON {
			payload, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				return err
			}
			if err := validateConnectivityReport(payload); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(os.Stdout, string(payload)); err != nil {
				return err
			}
			return nil
		}

		renderConnectivityReportTable(report)
		return nil
	},
}

func runConnectivity(ctx context.Context, promptSlug string, role string, promptPreferred string, resolved *ailink.ResolvedProvider, resolutionSource string, routingTarget string, format output.Format) (*connectivityReport, error) {
	if resolved == nil {
		return nil, fmt.Errorf("provider not resolved")
	}

	timeout := doctorAILinkConnectivityTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	baseURL := strings.TrimSpace(resolved.BaseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(resolved.Provider.BaseURL)
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return nil, fmt.Errorf("base_url has no host")
	}

	port := 0
	if u.Port() != "" {
		p, err := strconv.Atoi(u.Port())
		if err == nil {
			port = p
		}
	}
	if port == 0 {
		if strings.EqualFold(u.Scheme, "http") {
			port = 80
		} else {
			port = 443
		}
	}

	proxyEnv := collectProxyEnv(host)

	report := &connectivityReport{
		Version:   versionInfo.Version,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Input: connectivityInput{
			PromptSlug:         promptSlug,
			Role:               role,
			ProviderIDOverride: strings.TrimSpace(doctorAILinkConnectivityProviderID),
			TimeoutSeconds:     int(timeout.Seconds()),
			Output:             string(format),
			Quiet:              doctorAILinkConnectivityQuiet,
			ShowSecrets:        doctorAILinkConnectivityShowSecrets,
		},
		Resolution: connectivityResolution{
			ProviderID:       resolved.ProviderID,
			ResolutionSource: resolutionSource,
			RoutingTarget:    routingTarget,
			AIProvider:       resolved.Provider.AIProvider,
			BaseURL:          baseURL,
			Host:             host,
			Port:             port,
			Model:            resolved.Model,
			Credential: connectivityCredential{
				SelectionPolicy:   resolved.Provider.SelectionPolicy,
				DefaultCredential: resolved.Provider.DefaultCredential,
				SelectedLabel:     resolved.Credential.Label,
				SelectedPriority:  resolved.Credential.Priority,
				APIKeyPresent:     strings.TrimSpace(resolved.Credential.APIKey) != "",
			},
		},
		Environment: proxyEnv,
	}

	if report.Resolution.Credential.APIKeyPresent && doctorAILinkConnectivityShowSecrets {
		report.Resolution.Credential.APIKeyHint = maskKey(resolved.Credential.APIKey)
	}

	// promptPreferred comes from the resolved prompt definition (so it reflects overrides).
	providerDefault := ""
	if resolved.Provider.Models != nil {
		providerDefault = strings.TrimSpace(resolved.Provider.Models["default"])
	}
	modelSource := "unknown"
	switch {
	case promptPreferred != "":
		modelSource = "prompt_preferred_models"
	case providerDefault != "":
		modelSource = "provider.models.default"
	}
	if strings.TrimSpace(report.Resolution.Model) != "" {
		report.Resolution.ModelSource = modelSource
		report.Resolution.PromptPreferred = promptPreferred
		report.Resolution.ProviderDefault = providerDefault
	}

	checks := make([]connectivityCheck, 0, 4)

	dnsCheck := runDNSCheck(ctx, host, timeout)
	checks = append(checks, dnsCheck)
	if !dnsCheck.OK {
		report.Checks = checks
		report.Summary = classifyConnectivity(checks, report, "dns")
		return report, nil
	}

	tcpCheck, conn := runTCPCheck(ctx, host, port, timeout)
	checks = append(checks, tcpCheck)
	if !tcpCheck.OK {
		report.Checks = checks
		report.Summary = classifyConnectivity(checks, report, "tcp")
		return report, nil
	}

	tlsCheck := runTLSCheck(ctx, host, conn, timeout)
	checks = append(checks, tlsCheck)
	if !tlsCheck.OK {
		report.Checks = checks
		report.Summary = classifyConnectivity(checks, report, "tls")
		return report, nil
	}

	httpCheck := runHTTPAuthCheck(ctx, report.Resolution.AIProvider, baseURL, resolved.Credential.APIKey, timeout)
	checks = append(checks, httpCheck)
	report.Checks = checks
	report.Summary = classifyConnectivity(checks, report, "http_auth")
	return report, nil
}

func runDNSCheck(ctx context.Context, host string, timeout time.Duration) connectivityCheck {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	elapsed := time.Since(start)
	check := connectivityCheck{Name: "dns", LatencyMS: elapsed.Milliseconds()}
	if err != nil {
		check.OK = false
		check.Error = &connectivityErrInfo{Code: "DNS_ERROR", Message: err.Error()}
		return check
	}

	resolved := make([]string, 0, len(ips))
	for _, ip := range ips {
		resolved = append(resolved, ip.IP.String())
	}
	check.OK = true
	check.Details = map[string]any{"resolved_ips": resolved}
	return check
}

func runTCPCheck(ctx context.Context, host string, port int, timeout time.Duration) (connectivityCheck, net.Conn) {
	start := time.Now()
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", host, port))
	elapsed := time.Since(start)
	check := connectivityCheck{Name: "tcp", LatencyMS: elapsed.Milliseconds()}
	if err != nil {
		check.OK = false
		check.Error = &connectivityErrInfo{Code: "TCP_ERROR", Message: err.Error()}
		return check, nil
	}
	check.OK = true
	check.Details = map[string]any{"remote_addr": conn.RemoteAddr().String()}
	return check, conn
}

func runTLSCheck(ctx context.Context, host string, conn net.Conn, timeout time.Duration) connectivityCheck {
	check := connectivityCheck{Name: "tls"}
	if conn == nil {
		check.Skipped = true
		check.OK = false
		return check
	}

	start := time.Now()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	client := tls.Client(conn, &tls.Config{ServerName: host})
	err := client.HandshakeContext(ctx)
	elapsed := time.Since(start)
	check.LatencyMS = elapsed.Milliseconds()

	defer func() { _ = client.Close() }()
	if err != nil {
		check.OK = false
		check.Error = &connectivityErrInfo{Code: "TLS_ERROR", Message: err.Error()}
		return check
	}

	state := client.ConnectionState()
	check.OK = true
	if len(state.PeerCertificates) > 0 {
		leaf := state.PeerCertificates[0]
		serverMatch := certMatchesHost(leaf, host)
		suspicious := !serverMatch || looksLikeIntercept(leaf)
		check.Details = map[string]any{
			"tls_version":         tlsVersionName(state.Version),
			"cipher_suite":        tls.CipherSuiteName(state.CipherSuite),
			"cert_subject":        leaf.Subject.CommonName,
			"cert_issuer":         leaf.Issuer.CommonName,
			"cert_not_after":      leaf.NotAfter.UTC().Format(time.RFC3339),
			"intercept_suspected": suspicious,
			"server_name_match":   serverMatch,
		}

	}
	return check
}

func runHTTPAuthCheck(ctx context.Context, aiProvider string, baseURL string, apiKey string, timeout time.Duration) connectivityCheck {
	check := connectivityCheck{Name: "http_auth"}
	aiProvider = strings.ToLower(strings.TrimSpace(aiProvider))
	if aiProvider != "xai" && aiProvider != "openai" {
		check.Skipped = true
		check.OK = false
		check.Details = map[string]any{"reason": "auth probe not supported for ai_provider"}
		return check
	}
	if strings.TrimSpace(apiKey) == "" {
		check.Skipped = true
		check.OK = false
		check.Error = &connectivityErrInfo{Code: "NO_API_KEY", Message: "no API key configured"}
		return check
	}

	modelsURL := strings.TrimRight(baseURL, "/") + "/models"

	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
	if err != nil {
		check.OK = false
		check.Error = &connectivityErrInfo{Code: "HTTP_REQUEST_ERROR", Message: err.Error()}
		return check
	}
	addAuthHeader(req, apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	check.LatencyMS = elapsed.Milliseconds()
	if err != nil {
		check.OK = false
		check.Error = &connectivityErrInfo{Code: "HTTP_ERROR", Message: err.Error()}
		return check
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 32768))
	if err != nil {
		check.OK = false
		check.Error = &connectivityErrInfo{Code: "HTTP_READ_ERROR", Message: err.Error()}
		return check
	}

	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	details := map[string]any{
		"url":          modelsURL,
		"status_code":  resp.StatusCode,
		"content_type": contentType,
	}
	applyBodyDetails(details, contentType, body)
	check.Details = details

	switch resp.StatusCode {
	case 200:
		check.OK = true
	case 401, 403:
		check.OK = false
		check.Error = &connectivityErrInfo{Code: "AUTH_ERROR", Message: resp.Status}
	case 429:
		check.OK = false
		check.Error = &connectivityErrInfo{Code: "RATE_LIMITED", Message: resp.Status}
	case 502, 503, 504:
		check.OK = false
		check.Error = &connectivityErrInfo{Code: "PROVIDER_UNAVAILABLE", Message: resp.Status}
	default:
		check.OK = false
		check.Error = &connectivityErrInfo{Code: "HTTP_STATUS_ERROR", Message: resp.Status}
	}
	return check
}

func validateConnectivityReport(payload []byte) error {
	catalog, err := buildSchemaCatalog()
	if err != nil {
		// When running outside the repo, schemas may not be available.
		// Still emit the report; it is designed to be schema-conformant.
		return nil
	}

	diagnostics, err := catalog.ValidateDataByID("ailink/v0/connectivity-report", payload)
	if err != nil {
		return err
	}
	if len(diagnostics) > 0 {
		return fmt.Errorf("connectivity report schema validation failed: %s", diagnostics[0].Message)
	}
	return nil
}

func renderConnectivityReportTable(report *connectivityReport) {
	if report == nil {
		return
	}

	headline := "AILink Connectivity"
	status := "OK"
	if !report.Summary.OK {
		status = "FAIL"
	}

	lines := []string{
		fmt.Sprintf("%s (%s)", headline, status),
		"",
		fmt.Sprintf("prompt: %s", report.Input.PromptSlug),
		fmt.Sprintf("role:   %s", report.Input.Role),
		fmt.Sprintf("prov:   %s (%s)", report.Resolution.ProviderID, report.Resolution.AIProvider),
		fmt.Sprintf("url:    %s", report.Resolution.BaseURL),
		fmt.Sprintf("model:  %s", report.Resolution.Model),
		"",
	}

	for _, chk := range report.Checks {
		label := chk.Name
		if chk.Skipped {
			lines = append(lines, fmt.Sprintf("%-9s %s", label+":", "skipped"))
			continue
		}
		symbol := "✅"
		if !chk.OK {
			symbol = "❌"
		}
		suffix := ""
		if chk.LatencyMS > 0 {
			suffix = fmt.Sprintf(" (%dms)", chk.LatencyMS)
		}
		msg := "ok"
		if chk.Error != nil {
			msg = chk.Error.Code
		}
		lines = append(lines, fmt.Sprintf("%-9s %s %s%s", label+":", symbol, msg, suffix))
	}

	if len(report.Summary.Hints) > 0 {
		lines = append(lines, "", "hints:")
		for _, hint := range report.Summary.Hints {
			lines = append(lines, "- "+hint)
		}
	}

	_, _ = fmt.Fprint(os.Stdout, ascii.DrawBox(strings.Join(lines, "\n"), 0))
}

func collectProxyEnv(host string) connectivityEnv {
	httpProxy := envSet("HTTP_PROXY") || envSet("http_proxy")
	httpsProxy := envSet("HTTPS_PROXY") || envSet("https_proxy")
	noProxyVal := firstEnv("NO_PROXY", "no_proxy")
	noProxySet := strings.TrimSpace(noProxyVal) != ""

	matches := false
	if noProxySet {
		matches = noProxyMatchesHost(noProxyVal, host)
	}

	return connectivityEnv{
		HTTPProxySet:   httpProxy,
		HTTPSProxySet:  httpsProxy,
		NoProxySet:     noProxySet,
		NoProxyMatches: matches,
	}
}

func envSet(key string) bool {
	_, ok := os.LookupEnv(key)
	return ok
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			return value
		}
	}
	return ""
}

func noProxyMatchesHost(noProxy string, host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return false
	}

	for _, part := range strings.Split(noProxy, ",") {
		entry := strings.ToLower(strings.TrimSpace(part))
		if entry == "" {
			continue
		}
		if strings.EqualFold(entry, host) {
			return true
		}
		entry = strings.TrimPrefix(entry, ".")
		if entry != "" && strings.HasSuffix(host, "."+entry) {
			return true
		}
	}
	return false
}

func maskKey(apiKey string) string {
	apiKey = strings.TrimSpace(apiKey)
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:4] + "…" + apiKey[len(apiKey)-3:]
}

func addAuthHeader(req *http.Request, apiKey string) {
	if req == nil {
		return
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
}

func applyBodyDetails(details map[string]any, contentType string, body []byte) {
	if details == nil {
		return
	}
	if len(body) == 0 {
		return
	}

	details["body_bytes"] = len(body)

	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if strings.Contains(contentType, "json") {
		var parsed any
		if err := json.Unmarshal(body, &parsed); err == nil {
			if truncated, ok := truncateJSONSnippet(parsed); ok {
				details["body_json_snippet"] = truncated
				details["body_json_truncated"] = true
			} else {
				details["body_json_snippet"] = parsed
			}
			return
		}
	}

	// Fallback: store a plain text snippet.
	text := strings.TrimSpace(string(body))
	if len(text) > 200 {
		text = text[:200]
	}
	details["body_text_snippet"] = text
}

func truncateJSONSnippet(value any) (any, bool) {
	obj, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	data, ok := obj["data"].([]any)
	if !ok {
		return nil, false
	}
	if len(data) <= 3 {
		return nil, false
	}
	copyObj := make(map[string]any, len(obj)+1)
	for k, v := range obj {
		copyObj[k] = v
	}
	copyObj["data"] = data[:3]
	copyObj["data_truncated"] = true
	return copyObj, true
}

func certMatchesHost(cert *x509.Certificate, host string) bool {
	if cert == nil {
		return false
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	return cert.VerifyHostname(host) == nil
}

func looksLikeIntercept(cert *x509.Certificate) bool {
	if cert == nil {
		return false
	}

	issuer := strings.ToLower(strings.TrimSpace(cert.Issuer.CommonName))
	subject := strings.ToLower(strings.TrimSpace(cert.Subject.CommonName))

	keywords := []string{"zscaler", "netskope", "bluecoat", "fortinet", "proxy", "corporate", "inspection"}
	for _, kw := range keywords {
		if strings.Contains(issuer, kw) || strings.Contains(subject, kw) {
			return true
		}
	}
	return false
}

func tlsVersionName(version uint16) string {
	switch version {
	case tls.VersionTLS13:
		return "TLS1.3"
	case tls.VersionTLS12:
		return "TLS1.2"
	case tls.VersionTLS11:
		return "TLS1.1"
	case tls.VersionTLS10:
		return "TLS1.0"
	default:
		return fmt.Sprintf("0x%x", version)
	}
}

func classifyConnectivity(checks []connectivityCheck, report *connectivityReport, lastLayer string) connectivitySummary {
	summary := connectivitySummary{OK: true, Classification: "ok"}

	for _, chk := range checks {
		if chk.Skipped {
			continue
		}
		if chk.OK {
			continue
		}
		summary.OK = false
		summary.FailureLayer = chk.Name
		break
	}

	if summary.OK {
		if lastLayer == "http_auth" {
			return summary
		}
		// If HTTP auth was skipped because ai_provider isn't supported, treat network reachability as ok.
		return summary
	}

	// Basic classification/hints.
	summary.Classification = "unknown"
	summary.Hints = make([]string, 0, 4)
	if report != nil {
		if report.Environment.HTTPProxySet || report.Environment.HTTPSProxySet {
			summary.Hints = append(summary.Hints, "Proxy environment variables are set; a corporate proxy may be intercepting/denying requests")
			if !report.Environment.NoProxyMatches {
				summary.Hints = append(summary.Hints, "Consider adding the provider host to NO_PROXY if proxying breaks TLS")
			}
		}
	}

	switch summary.FailureLayer {
	case "dns":
		summary.Classification = "dns_failure"
		summary.Hints = append(summary.Hints, "DNS resolution failed; check VPN/DNS configuration")
	case "tcp":
		summary.Classification = "network_blocked"
		summary.Hints = append(summary.Hints, "TCP connection failed; VPN/firewall may be blocking outbound connections")
	case "tls":
		summary.Classification = "tls_failure"
		summary.Hints = append(summary.Hints, "TLS handshake failed; check proxy/VPN interception or certificates")
	case "http_auth":
		summary.Classification = "http_error"
		for _, chk := range checks {
			if chk.Name != "http_auth" || chk.Error == nil {
				continue
			}
			switch chk.Error.Code {
			case "NO_API_KEY":
				summary.Classification = "misconfigured"
				summary.Hints = append(summary.Hints, "No API key configured for selected provider credential")
			case "AUTH_ERROR":
				summary.Classification = "auth_invalid"
				summary.Hints = append(summary.Hints, "API key rejected (401/403); verify key and permissions")
			case "RATE_LIMITED":
				summary.Classification = "rate_limited"
				summary.Hints = append(summary.Hints, "Provider rate limited the request; retry later or rotate credentials")
			case "PROVIDER_UNAVAILABLE":
				summary.Classification = "provider_overloaded"
				summary.Hints = append(summary.Hints, "Provider returned 5xx; retry later")
			default:
				summary.Classification = "http_error"
			}
			break
		}
	}

	return summary
}

func init() {
	doctorAILinkCmd.AddCommand(doctorAILinkConnectivityCmd)

	doctorAILinkConnectivityCmd.Flags().StringVar(&doctorAILinkConnectivityRole, "role", "", "Role to resolve (defaults to prompt slug)")
	doctorAILinkConnectivityCmd.Flags().StringVar(&doctorAILinkConnectivityProviderID, "provider-id", "", "Force a provider instance id")
	doctorAILinkConnectivityCmd.Flags().DurationVar(&doctorAILinkConnectivityTimeout, "timeout", 10*time.Second, "Timeout per step (e.g. 10s)")
	doctorAILinkConnectivityCmd.Flags().BoolVar(&doctorAILinkConnectivityQuiet, "quiet", false, "Exit code only")
	doctorAILinkConnectivityCmd.Flags().BoolVar(&doctorAILinkConnectivityShowSecrets, "show-secrets", false, "Include masked key hints in output")
	doctorAILinkConnectivityCmd.Flags().StringVar(&doctorAILinkConnectivityOutputRaw, "output", string(output.FormatTable), "Output format: table|json")
}
