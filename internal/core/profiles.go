package core

import (
	"strings"
	"time"
)

// Profile defines what to check for a given name.
type Profile struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	TLDs        []string `json:"tlds,omitempty"`
	Registries  []string `json:"registries,omitempty"`
	Handles     []string `json:"handles,omitempty"`
}

// ProfileRecord wraps a profile with persistence metadata.
type ProfileRecord struct {
	Profile   Profile
	IsBuiltin bool
	UpdatedAt time.Time
}

// BuiltInProfiles provides default profiles bundled with NameLens.
var BuiltInProfiles = []Profile{
	{
		Name:        "startup",
		Description: "Balanced checks for common startup naming needs",
		TLDs:        []string{"com", "io", "dev", "app"},
		Registries:  []string{"npm", "pypi"},
		Handles:     []string{"github"},
	},
	{
		Name:        "minimal",
		Description: "Minimal domain-only check for quick availability scan",
		TLDs:        []string{"com"},
	},
	{
		Name:        "developer",
		Description: "Developer tool naming with package registries and code hosts",
		TLDs:        []string{"com", "io", "dev", "app", "sh", "org", "net"},
		Registries:  []string{"npm", "pypi", "cargo"},
		Handles:     []string{"github"},
	},
	{
		Name:        "website",
		Description: "Traditional website domains for general web presence",
		TLDs:        []string{"com", "org", "net"},
	},
	{
		Name:        "web3",
		Description: "Web3-friendly domains with registry + handle checks",
		TLDs:        []string{"xyz", "io", "gg"},
		Registries:  []string{"npm"},
		Handles:     []string{"github"},
	},
}

// FindBuiltInProfile looks up a built-in profile by name.
func FindBuiltInProfile(name string) (*Profile, bool) {
	needle := strings.TrimSpace(strings.ToLower(name))
	if needle == "" {
		return nil, false
	}

	for _, profile := range BuiltInProfiles {
		if strings.EqualFold(profile.Name, needle) {
			copied := profile
			return &copied, true
		}
	}

	return nil, false
}
