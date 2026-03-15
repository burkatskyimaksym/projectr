package upload

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/burkatskyimaksym/projectr/internal/config"
)

// CheckRclone verifies that rclone is installed and accessible.
func CheckRclone() error {
	if _, err := exec.LookPath("rclone"); err != nil {
		return fmt.Errorf("rclone not found — install it from https://rclone.org/install/")
	}
	return nil
}

// Upload copies the final/ folder of a project to the configured rclone remote.
// If rclone settings are missing, it triggers interactive setup first.
func Upload(cfg *config.Config, projectName string) error {
	if err := CheckRclone(); err != nil {
		return err
	}

	// Find the project folder — exact match or prefix match
	projectPath, err := findProject(cfg.OrdersPath, projectName)
	if err != nil {
		return err
	}

	finalDir := filepath.Join(projectPath, "final")
	if _, err := os.Stat(finalDir); os.IsNotExist(err) {
		return fmt.Errorf("no final/ folder found in: %s", projectPath)
	}

	// Check final/ is not empty
	entries, err := os.ReadDir(finalDir)
	if err != nil {
		return fmt.Errorf("could not read final/ folder: %w", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("final/ folder is empty — nothing to upload")
	}

	// Build rclone destination: remote:RemotePath/projectName/
	folderName := filepath.Base(projectPath)
	dest := fmt.Sprintf("%s:%s/%s", cfg.RemoteName, cfg.RemotePath, folderName)

	fmt.Printf("\n☁  Uploading to %s\n", dest)
	fmt.Printf("   Source: %s\n\n", finalDir)

	// Run: rclone copy --progress finalDir dest
	cmd := exec.Command("rclone", "copy", "--progress", finalDir, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rclone failed: %w", err)
	}

	fmt.Printf("\n✅ Upload complete: %s\n", dest)
	return nil
}

// findProject returns the full path to a project folder by exact or prefix match.
func findProject(ordersPath, name string) (string, error) {
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
		fmt.Printf("   Project: %s\n", matches[0])
		return path, nil
	default:
		fmt.Println("Multiple projects match, be more specific:")
		for _, m := range matches {
			fmt.Printf("  - %s\n", m)
		}
		return "", fmt.Errorf("ambiguous project name: %q", name)
	}
}
