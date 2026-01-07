package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/namelens/namelens/internal/core"
)

type stubChecker struct {
	seen []string
}

func (s *stubChecker) Check(ctx context.Context, name string) (*core.CheckResult, error) {
	s.seen = append(s.seen, name)
	return &core.CheckResult{
		Name:      name,
		CheckType: core.CheckTypeDomain,
		TLD:       name[strings.LastIndex(name, ".")+1:],
		Available: core.AvailabilityUnknown,
	}, nil
}

func (s *stubChecker) Type() core.CheckType {
	return core.CheckTypeDomain
}

func (s *stubChecker) SupportsName(name string) bool {
	return name != ""
}

func TestOrchestratorDomains(t *testing.T) {
	checker := &stubChecker{}
	orchestrator := &Orchestrator{
		Checkers: map[core.CheckType]Checker{
			core.CheckTypeDomain: checker,
		},
	}

	profile := core.Profile{
		Name: "test",
		TLDs: []string{"com", ".io", " "},
	}

	results, err := orchestrator.Check(context.Background(), "example", profile)
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, []string{"example.com", "example.io"}, checker.seen)
}
