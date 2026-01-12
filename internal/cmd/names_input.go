package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func resolveNames(positional []string, namesFile string) ([]string, error) {
	trimmed := strings.TrimSpace(namesFile)
	if trimmed != "" {
		if len(positional) > 0 {
			return nil, fmt.Errorf("cannot combine positional names with --names-file")
		}
		return readNamesFile(trimmed)
	}

	names := make([]string, 0, len(positional))
	for _, raw := range positional {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" {
			continue
		}
		if err := validateName(name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("at least one name is required")
	}
	return names, nil
}

func readNamesFile(path string) ([]string, error) {
	var reader io.Reader
	if path == "-" {
		reader = os.Stdin
	} else {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer file.Close() // nolint:errcheck
		reader = file
	}

	names := make([]string, 0)
	scanner := bufio.NewScanner(reader)
	line := 0
	for scanner.Scan() {
		line++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		name := strings.ToLower(raw)
		if err := validateName(name); err != nil {
			return nil, fmt.Errorf("invalid name on line %d: %w", line, err)
		}
		names = append(names, name)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(names) == 0 {
		return nil, fmt.Errorf("no names found")
	}
	return names, nil
}
