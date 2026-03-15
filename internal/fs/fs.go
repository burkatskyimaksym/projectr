package fs

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindProject returns the full path to a project folder by exact or prefix match.
func FindProject(ordersPath, name string) (string, error) {
	// Try exact match first
	exact := filepath.Join(ordersPath, name)
	if _, err := os.Stat(exact); err == nil {
		return exact, nil
	}

	// Fall back to prefix match (e.g. "35" matches "35 Logo redesign (maria22)")
	entries, err := os.ReadDir(ordersPath)
	if err != nil {
		return "", fmt.Errorf("could not read orders folder: %w", err)
	}

	var matches []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if len(e.Name()) >= len(name) && e.Name()[:len(name)] == name {
			matches = append(matches, e.Name())
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("project not found: %q", name)
	case 1:
		path := filepath.Join(ordersPath, matches[0])
		fmt.Printf("   Project: %s\n", filepath.Base(path))
		return path, nil
	default:
		fmt.Println("Multiple projects match, be more specific:")
		for _, m := range matches {
			fmt.Printf("  - %s\n", m)
		}
		return "", fmt.Errorf("ambiguous project name: %q", name)
	}
}
