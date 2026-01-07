package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/namelens/namelens/internal/core"
)

// Orchestrator coordinates checks across available checkers.
type Orchestrator struct {
	Checkers           map[core.CheckType]Checker
	RegistryCheckers   map[string]Checker
	HandleCheckers     map[string]Checker
	IncludeUnsupported bool
	Clock              func() time.Time
}

// Checker describes a name availability checker.
type Checker interface {
	Check(ctx context.Context, name string) (*core.CheckResult, error)
	Type() core.CheckType
	SupportsName(name string) bool
}

// Check runs checks based on the provided profile.
func (o *Orchestrator) Check(ctx context.Context, name string, profile core.Profile) ([]*core.CheckResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	baseName := strings.TrimSpace(name)
	if baseName == "" {
		return nil, fmt.Errorf("name is required")
	}

	results := make([]*core.CheckResult, 0)

	if len(profile.TLDs) > 0 {
		domainChecker := o.getChecker(core.CheckTypeDomain)
		for _, tld := range profile.TLDs {
			normalized := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(tld)), ".")
			if normalized == "" {
				continue
			}
			domain := fmt.Sprintf("%s.%s", baseName, normalized)
			result, err := o.runChecker(ctx, domainChecker, core.CheckTypeDomain, domain)
			if err != nil {
				return nil, err
			}
			if result != nil {
				results = append(results, result)
			}
		}
	}

	for _, registry := range profile.Registries {
		key := normalizeKey(registry)
		if key == "" {
			continue
		}
		checker := o.getNamedChecker(o.RegistryCheckers, key)
		result, err := o.runNamedChecker(ctx, checker, key, baseName)
		if err != nil {
			return nil, err
		}
		if result != nil {
			results = append(results, result)
		}
	}

	for _, handle := range profile.Handles {
		key := normalizeKey(handle)
		if key == "" {
			continue
		}
		checker := o.getNamedChecker(o.HandleCheckers, key)
		result, err := o.runNamedChecker(ctx, checker, key, baseName)
		if err != nil {
			return nil, err
		}
		if result != nil {
			results = append(results, result)
		}
	}

	return results, nil
}

func (o *Orchestrator) runChecker(ctx context.Context, c Checker, checkType core.CheckType, name string) (*core.CheckResult, error) {
	if c == nil {
		if !o.IncludeUnsupported {
			return nil, nil
		}
		return o.unsupportedResult(name, checkType, "checker not configured"), nil
	}

	if !c.SupportsName(name) {
		if !o.IncludeUnsupported {
			return nil, nil
		}
		return o.unsupportedResult(name, checkType, "checker does not support name"), nil
	}

	result, err := c.Check(ctx, name)
	if err != nil {
		if !o.IncludeUnsupported {
			return &core.CheckResult{
				Name:      name,
				CheckType: checkType,
				Available: core.AvailabilityError,
				Message:   err.Error(),
				Provenance: core.Provenance{
					RequestedAt: o.now(),
					ResolvedAt:  o.now(),
					Source:      "orchestrator",
				},
			}, nil
		}
		return nil, err
	}

	return result, nil
}

func (o *Orchestrator) getChecker(checkType core.CheckType) Checker {
	if o == nil || o.Checkers == nil {
		return nil
	}
	return o.Checkers[checkType]
}

func (o *Orchestrator) getNamedChecker(group map[string]Checker, key string) Checker {
	if o == nil || group == nil {
		return nil
	}
	return group[key]
}

func (o *Orchestrator) runNamedChecker(ctx context.Context, c Checker, key string, name string) (*core.CheckResult, error) {
	checkType, ok := checkTypeForKey(key)
	if !ok {
		return nil, nil
	}
	return o.runChecker(ctx, c, checkType, name)
}

func normalizeKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func checkTypeForKey(key string) (core.CheckType, bool) {
	switch key {
	case "npm":
		return core.CheckTypeNPM, true
	case "pypi":
		return core.CheckTypePyPI, true
	case "github":
		return core.CheckTypeGitHub, true
	default:
		return "", false
	}
}

func (o *Orchestrator) unsupportedResult(name string, checkType core.CheckType, message string) *core.CheckResult {
	now := o.now()
	return &core.CheckResult{
		Name:      name,
		CheckType: checkType,
		Available: core.AvailabilityUnsupported,
		Message:   message,
		Provenance: core.Provenance{
			RequestedAt: now,
			ResolvedAt:  now,
			Source:      "orchestrator",
		},
	}
}

func (o *Orchestrator) now() time.Time {
	if o != nil && o.Clock != nil {
		return o.Clock()
	}
	return time.Now().UTC()
}
