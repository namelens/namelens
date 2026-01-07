package ailink

import (
	"fmt"
	"strings"
	"sync"

	"github.com/namelens/namelens/internal/ailink/driver"
	"github.com/namelens/namelens/internal/ailink/driver/xai"
	"github.com/namelens/namelens/internal/ailink/prompt"
)

type Registry struct {
	cfg Config

	mu      sync.Mutex
	drivers map[string]driver.Driver
	rr      map[string]int
}

type ResolvedProvider struct {
	ProviderID string
	Provider   ProviderInstanceConfig
	Credential CredentialConfig
	Driver     driver.Driver
	Model      string
	BaseURL    string
}

func NewRegistry(cfg Config) *Registry {
	return &Registry{cfg: cfg}
}

func (r *Registry) Resolve(role string, promptDef *prompt.Prompt, modelOverride string) (*ResolvedProvider, error) {
	providerID, providerCfg, err := r.resolveProvider(role)
	if err != nil {
		return nil, err
	}

	cred, credKey, err := selectCredential(providerCfg, func(groupKey string, n int) int {
		return r.rrIndex(providerID+":"+groupKey, n)
	})
	if err != nil {
		return nil, err
	}

	drv, err := r.driverFor(providerID, providerCfg, cred, credKey)
	if err != nil {
		return nil, err
	}

	model, err := resolveModel(providerCfg, promptDef, modelOverride)
	if err != nil {
		return nil, err
	}

	baseURL := strings.TrimSpace(providerCfg.BaseURL)
	if client, ok := drv.(*xai.Client); ok {
		baseURL = strings.TrimSpace(client.BaseURL)
	}

	return &ResolvedProvider{
		ProviderID: providerID,
		Provider:   providerCfg,
		Credential: cred,
		Driver:     drv,
		Model:      model,
		BaseURL:    baseURL,
	}, nil
}

func (r *Registry) resolveProvider(role string) (string, ProviderInstanceConfig, error) {
	if r == nil {
		return "", ProviderInstanceConfig{}, fmt.Errorf("ailink registry not configured")
	}

	role = strings.TrimSpace(role)
	if role != "" {
		if providerID, ok := r.cfg.Routing[role]; ok {
			providerID = strings.TrimSpace(providerID)
			if providerID != "" {
				providerCfg, ok := r.cfg.Providers[providerID]
				if !ok {
					return "", ProviderInstanceConfig{}, fmt.Errorf("unknown provider %q for role %q", providerID, role)
				}
				if !providerCfg.Enabled {
					return "", ProviderInstanceConfig{}, fmt.Errorf("provider %q is disabled", providerID)
				}
				return providerID, providerCfg, nil
			}
		}

		for providerID, providerCfg := range r.cfg.Providers {
			if !providerCfg.Enabled {
				continue
			}
			if contains(providerCfg.Roles, role) {
				return providerID, providerCfg, nil
			}
		}
	}

	if id := strings.TrimSpace(r.cfg.DefaultProvider); id != "" {
		providerCfg, ok := r.cfg.Providers[id]
		if !ok {
			return "", ProviderInstanceConfig{}, fmt.Errorf("default provider %q not configured", id)
		}
		if !providerCfg.Enabled {
			return "", ProviderInstanceConfig{}, fmt.Errorf("default provider %q is disabled", id)
		}
		return id, providerCfg, nil
	}

	var onlyID string
	var onlyCfg ProviderInstanceConfig
	for providerID, providerCfg := range r.cfg.Providers {
		if !providerCfg.Enabled {
			continue
		}
		if onlyID != "" {
			return "", ProviderInstanceConfig{}, fmt.Errorf("no provider routing configured")
		}
		onlyID = providerID
		onlyCfg = providerCfg
	}
	if onlyID == "" {
		return "", ProviderInstanceConfig{}, fmt.Errorf("no enabled providers configured")
	}
	return onlyID, onlyCfg, nil
}

func selectCredential(cfg ProviderInstanceConfig, rrNext func(groupKey string, n int) int) (CredentialConfig, string, error) {
	if len(cfg.Credentials) == 0 {
		return CredentialConfig{}, "", fmt.Errorf("no credentials configured")
	}

	enabled := make([]CredentialConfig, 0, len(cfg.Credentials))
	for _, cred := range cfg.Credentials {
		if !cred.Enabled && strings.TrimSpace(cred.Label) != "" {
			continue
		}
		if strings.TrimSpace(cred.APIKey) == "" {
			continue
		}
		enabled = append(enabled, cred)
	}
	if len(enabled) == 0 {
		// Credentials exist but are not usable; return first so caller can report missing key.
		cred := cfg.Credentials[0]
		key := strings.TrimSpace(cred.Label)
		if key == "" {
			key = "0"
		}
		return cred, key, nil
	}

	if label := strings.TrimSpace(cfg.DefaultCredential); label != "" {
		for _, cred := range enabled {
			if strings.EqualFold(strings.TrimSpace(cred.Label), label) {
				return cred, strings.TrimSpace(cred.Label), nil
			}
		}
	}

	policy := strings.ToLower(strings.TrimSpace(cfg.SelectionPolicy))
	if policy == "" {
		policy = "priority"
	}

	// Compute highest priority set.
	highest := enabled[0].Priority
	for _, cred := range enabled[1:] {
		if cred.Priority > highest {
			highest = cred.Priority
		}
	}
	group := make([]CredentialConfig, 0, len(enabled))
	for _, cred := range enabled {
		if cred.Priority == highest {
			group = append(group, cred)
		}
	}

	switch policy {
	case "round_robin":
		idx := 0
		if rrNext != nil {
			idx = rrNext(fmt.Sprintf("%d", highest), len(group))
		}
		cred := group[idx]
		key := strings.TrimSpace(cred.Label)
		if key == "" {
			key = fmt.Sprintf("p%d", highest)
		}
		return cred, key, nil
	case "priority":
		fallthrough
	default:
		cred := group[0]
		key := strings.TrimSpace(cred.Label)
		if key == "" {
			key = fmt.Sprintf("p%d", highest)
		}
		return cred, key, nil
	}
}

func (r *Registry) driverFor(providerID string, providerCfg ProviderInstanceConfig, cred CredentialConfig, credKey string) (driver.Driver, error) {
	if strings.TrimSpace(providerID) == "" {
		return nil, fmt.Errorf("provider id is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.drivers == nil {
		r.drivers = map[string]driver.Driver{}
	}
	if r.rr == nil {
		r.rr = map[string]int{}
	}
	driverKey := providerID
	if strings.TrimSpace(credKey) != "" {
		driverKey += ":" + credKey
	}
	if drv, ok := r.drivers[driverKey]; ok {
		return drv, nil
	}

	providerType := strings.ToLower(strings.TrimSpace(providerCfg.AIProvider))
	switch providerType {
	case "xai":
		client := xai.NewClient(providerCfg.BaseURL, cred.APIKey)
		client.Timeout = r.cfg.DefaultTimeout
		r.drivers[driverKey] = client
		return client, nil
	default:
		if providerType == "" {
			providerType = "(unset)"
		}
		return nil, fmt.Errorf("unsupported ai_provider %q for provider %q", providerType, providerID)
	}
}

func resolveModel(providerCfg ProviderInstanceConfig, promptDef *prompt.Prompt, override string) (string, error) {
	model := strings.TrimSpace(override)
	if model != "" {
		return model, nil
	}

	if promptDef != nil {
		if models := preferredModels(promptDef); len(models) > 0 {
			model = strings.TrimSpace(models[0])
			if model != "" {
				return model, nil
			}
		}
	}

	if providerCfg.Models != nil {
		model = strings.TrimSpace(providerCfg.Models["default"])
		if model != "" {
			return model, nil
		}
	}

	return "", fmt.Errorf("model not configured")
}

func preferredModels(promptDef *prompt.Prompt) []string {
	if promptDef == nil {
		return nil
	}

	value, ok := promptDef.Config.ProviderHints["preferred_models"]
	if !ok || value == nil {
		return nil
	}

	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		models := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				models = append(models, s)
			}
		}
		return models
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{typed}
	default:
		return nil
	}
}

func (r *Registry) rrIndex(key string, n int) int {
	if n <= 1 {
		return 0
	}
	if r == nil {
		return 0
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.rr == nil {
		r.rr = map[string]int{}
	}
	idx := r.rr[key] % n
	r.rr[key] = r.rr[key] + 1
	return idx
}

func contains(values []string, needle string) bool {
	needle = strings.TrimSpace(needle)
	if needle == "" {
		return false
	}
	for _, v := range values {
		if strings.EqualFold(strings.TrimSpace(v), needle) {
			return true
		}
	}
	return false
}
