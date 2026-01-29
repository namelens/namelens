package context

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ExtractMetadataFromFile extracts key metadata from package/project files.
// Returns a formatted string suitable for inclusion in context.
func ExtractMetadataFromFile(content []byte, filename string) string {
	switch {
	case strings.HasSuffix(filename, "package.json"):
		return extractPackageJSON(content)
	case strings.HasSuffix(filename, "Cargo.toml"):
		return extractCargoToml(content)
	case strings.HasSuffix(filename, "go.mod"):
		return extractGoMod(content)
	case strings.HasSuffix(filename, "pyproject.toml"):
		return extractPyProjectToml(content)
	default:
		return string(content)
	}
}

// extractPackageJSON extracts name, version, description from package.json.
func extractPackageJSON(content []byte) string {
	var pkg struct {
		Name        string   `json:"name"`
		Version     string   `json:"version"`
		Description string   `json:"description"`
		Keywords    []string `json:"keywords"`
		Author      string   `json:"author"`
		License     string   `json:"license"`
	}

	if err := json.Unmarshal(content, &pkg); err != nil {
		return "[Could not parse package.json]"
	}

	var parts []string
	if pkg.Name != "" {
		parts = append(parts, fmt.Sprintf("Name: %s", pkg.Name))
	}
	if pkg.Version != "" {
		parts = append(parts, fmt.Sprintf("Version: %s", pkg.Version))
	}
	if pkg.Description != "" {
		parts = append(parts, fmt.Sprintf("Description: %s", pkg.Description))
	}
	if len(pkg.Keywords) > 0 {
		parts = append(parts, fmt.Sprintf("Keywords: %s", strings.Join(pkg.Keywords, ", ")))
	}
	if pkg.Author != "" {
		parts = append(parts, fmt.Sprintf("Author: %s", pkg.Author))
	}
	if pkg.License != "" {
		parts = append(parts, fmt.Sprintf("License: %s", pkg.License))
	}

	if len(parts) == 0 {
		return "[Empty package.json]"
	}

	return strings.Join(parts, "\n")
}

// extractCargoToml extracts name, version, description from Cargo.toml.
func extractCargoToml(content []byte) string {
	text := string(content)
	var parts []string

	// Extract [package] section fields using regex
	// This is a simple approach - not full TOML parsing
	if name := extractTomlField(text, "name"); name != "" {
		parts = append(parts, fmt.Sprintf("Name: %s", name))
	}
	if version := extractTomlField(text, "version"); version != "" {
		parts = append(parts, fmt.Sprintf("Version: %s", version))
	}
	if desc := extractTomlField(text, "description"); desc != "" {
		parts = append(parts, fmt.Sprintf("Description: %s", desc))
	}
	if license := extractTomlField(text, "license"); license != "" {
		parts = append(parts, fmt.Sprintf("License: %s", license))
	}
	if keywords := extractTomlArray(text, "keywords"); keywords != "" {
		parts = append(parts, fmt.Sprintf("Keywords: %s", keywords))
	}

	if len(parts) == 0 {
		return "[Could not extract Cargo.toml metadata]"
	}

	return strings.Join(parts, "\n")
}

// extractGoMod extracts module name and Go version from go.mod.
func extractGoMod(content []byte) string {
	text := string(content)
	var parts []string

	// Extract module line
	moduleRe := regexp.MustCompile(`(?m)^module\s+(.+)$`)
	if m := moduleRe.FindStringSubmatch(text); len(m) > 1 {
		parts = append(parts, fmt.Sprintf("Module: %s", strings.TrimSpace(m[1])))
	}

	// Extract go version
	goRe := regexp.MustCompile(`(?m)^go\s+(\d+\.\d+)`)
	if m := goRe.FindStringSubmatch(text); len(m) > 1 {
		parts = append(parts, fmt.Sprintf("Go Version: %s", m[1]))
	}

	if len(parts) == 0 {
		return "[Could not extract go.mod metadata]"
	}

	return strings.Join(parts, "\n")
}

// extractPyProjectToml extracts name, version, description from pyproject.toml.
func extractPyProjectToml(content []byte) string {
	text := string(content)
	var parts []string

	// Try [project] section first (PEP 621)
	if name := extractTomlField(text, "name"); name != "" {
		parts = append(parts, fmt.Sprintf("Name: %s", name))
	}
	if version := extractTomlField(text, "version"); version != "" {
		parts = append(parts, fmt.Sprintf("Version: %s", version))
	}
	if desc := extractTomlField(text, "description"); desc != "" {
		parts = append(parts, fmt.Sprintf("Description: %s", desc))
	}
	if license := extractTomlField(text, "license"); license != "" {
		parts = append(parts, fmt.Sprintf("License: %s", license))
	}
	if keywords := extractTomlArray(text, "keywords"); keywords != "" {
		parts = append(parts, fmt.Sprintf("Keywords: %s", keywords))
	}

	if len(parts) == 0 {
		return "[Could not extract pyproject.toml metadata]"
	}

	return strings.Join(parts, "\n")
}

// extractTomlField extracts a simple key = "value" field from TOML.
func extractTomlField(text, key string) string {
	// Match: key = "value" or key = 'value'
	re := regexp.MustCompile(fmt.Sprintf(`(?m)^%s\s*=\s*["']([^"']+)["']`, regexp.QuoteMeta(key)))
	if m := re.FindStringSubmatch(text); len(m) > 1 {
		return m[1]
	}
	return ""
}

// extractTomlArray extracts a simple array field from TOML.
func extractTomlArray(text, key string) string {
	// Match: key = ["a", "b", "c"]
	re := regexp.MustCompile(fmt.Sprintf(`(?m)^%s\s*=\s*\[([^\]]+)\]`, regexp.QuoteMeta(key)))
	if m := re.FindStringSubmatch(text); len(m) > 1 {
		// Clean up the array content
		items := strings.Split(m[1], ",")
		var cleaned []string
		for _, item := range items {
			item = strings.TrimSpace(item)
			item = strings.Trim(item, `"'`)
			if item != "" {
				cleaned = append(cleaned, item)
			}
		}
		return strings.Join(cleaned, ", ")
	}
	return ""
}
