package checker

import (
	"context"

	"github.com/namelens/namelens/internal/core"
)

// Checker is the interface all availability checkers implement.
type Checker interface {
	// Check performs availability check for the given name.
	Check(ctx context.Context, name string) (*core.CheckResult, error)

	// Type returns the checker type.
	Type() core.CheckType

	// SupportsName returns true if this checker can handle the name.
	SupportsName(name string) bool
}
