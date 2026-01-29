package context

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractPackageJSON(t *testing.T) {
	content := []byte(`{
		"name": "my-project",
		"version": "1.0.0",
		"description": "A test project",
		"keywords": ["test", "example"],
		"author": "Test Author",
		"license": "MIT",
		"dependencies": {
			"lodash": "^4.0.0"
		}
	}`)

	result := ExtractMetadataFromFile(content, "package.json")

	require.Contains(t, result, "Name: my-project")
	require.Contains(t, result, "Version: 1.0.0")
	require.Contains(t, result, "Description: A test project")
	require.Contains(t, result, "Keywords: test, example")
	require.Contains(t, result, "Author: Test Author")
	require.Contains(t, result, "License: MIT")
	// Should NOT contain dependencies
	require.NotContains(t, result, "lodash")
}

func TestExtractCargoToml(t *testing.T) {
	content := []byte(`[package]
name = "my-crate"
version = "0.1.0"
description = "A Rust library"
license = "MIT"
keywords = ["rust", "library"]

[dependencies]
serde = "1.0"
`)

	result := ExtractMetadataFromFile(content, "Cargo.toml")

	require.Contains(t, result, "Name: my-crate")
	require.Contains(t, result, "Version: 0.1.0")
	require.Contains(t, result, "Description: A Rust library")
	require.Contains(t, result, "License: MIT")
	require.Contains(t, result, "Keywords: rust, library")
	// Should NOT contain dependencies
	require.NotContains(t, result, "serde")
}

func TestExtractGoMod(t *testing.T) {
	content := []byte(`module github.com/example/myproject

go 1.21

require (
	github.com/some/dep v1.0.0
)
`)

	result := ExtractMetadataFromFile(content, "go.mod")

	require.Contains(t, result, "Module: github.com/example/myproject")
	require.Contains(t, result, "Go Version: 1.21")
	// Should NOT contain dependencies
	require.NotContains(t, result, "github.com/some/dep")
}

func TestExtractPyProjectToml(t *testing.T) {
	content := []byte(`[project]
name = "my-package"
version = "0.1.0"
description = "A Python package"
license = "MIT"
keywords = ["python", "package"]

[project.dependencies]
requests = "^2.0"
`)

	result := ExtractMetadataFromFile(content, "pyproject.toml")

	require.Contains(t, result, "Name: my-package")
	require.Contains(t, result, "Version: 0.1.0")
	require.Contains(t, result, "Description: A Python package")
	require.Contains(t, result, "License: MIT")
	// Should NOT contain dependencies
	require.NotContains(t, result, "requests")
}

func TestExtractMetadataUnknownFile(t *testing.T) {
	content := []byte("some random content")
	result := ExtractMetadataFromFile(content, "unknown.txt")

	// Should return content as-is for unknown files
	require.Equal(t, "some random content", result)
}

func TestExtractInvalidPackageJSON(t *testing.T) {
	content := []byte("not valid json")
	result := ExtractMetadataFromFile(content, "package.json")

	require.True(t, strings.Contains(result, "Could not parse"))
}
