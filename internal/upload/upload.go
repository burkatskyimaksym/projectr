package upload

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/burkatskyimaksym/projectr/internal/config"
	"github.com/burkatskyimaksym/projectr/internal/fs"
)

// CheckRclone verifies that rclone is installed and accessible.
func CheckRclone() error {
	if _, err := exec.LookPath("rclone"); err != nil {
		return fmt.Errorf("rclone not found — install it from https://rclone.org/install/")
	}
	return nil
}

// Upload copies the final/ folder of a project to the configured rclone remote.
func Upload(cfg *config.Config, projectName string) error {
	if err := CheckRclone(); err != nil {
		return err
	}

	projectPath, err := fs.FindProject(cfg.OrdersPath, projectName)
	if err != nil {
		return err
	}

	finalDir := filepath.Join(projectPath, "final")
	if _, err := os.Stat(finalDir); os.IsNotExist(err) {
		return fmt.Errorf("no final/ folder found in: %s", projectPath)
	}

	entries, err := os.ReadDir(finalDir)
	if err != nil {
		return fmt.Errorf("could not read final/ folder: %w", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("final/ folder is empty — nothing to upload")
	}

	folderName := filepath.Base(projectPath)
	dest := fmt.Sprintf("%s:%s/%s", cfg.RemoteName, cfg.RemotePath, folderName)

	fmt.Printf("\n☁  Uploading to %s\n", dest)
	fmt.Printf("   Source: %s\n\n", finalDir)

	cmd := exec.Command("rclone", "copy", "--progress", finalDir, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rclone failed: %w", err)
	}

	fmt.Printf("\n✅ Upload complete: %s\n", dest)
	return nil
}
