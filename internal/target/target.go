package target

import "github.com/IQNeoXen/aictx/internal/config"

// Target represents a tool whose configuration is managed by aictx.
type Target interface {
	// ID returns the unique identifier for this target (e.g. "claude-code-cli").
	ID() string

	// Name returns the human-readable name of this target.
	Name() string

	// Detect returns true if this target is installed / config exists.
	Detect() bool

	// Apply writes the given target entry's config into this target.
	Apply(te config.TargetEntry) error

	// Discover reads the target's current config and returns a TargetEntry.
	// Returns nil if nothing useful is found.
	Discover() (*config.TargetEntry, error)
}
