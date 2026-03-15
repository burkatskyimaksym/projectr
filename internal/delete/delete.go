package delete

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/burkatskyimaksym/projectr/internal/config"
	"github.com/burkatskyimaksym/projectr/internal/fs"
	"github.com/burkatskyimaksym/projectr/internal/store"
)

// Delete removes a project locally, from CSV, and optionally from Google Drive.
func Delete(cfg *config.Config, projectName string) error {
	// Resolve full project path (supports prefix match)
	projectPath, err := fs.FindProject(cfg.OrdersPath, projectName)
	if err != nil {
		return err
	}

	folderName := filepath.Base(projectPath)

	// Show what will be deleted and ask for confirmation
	fmt.Printf("\n🗑  About to delete: %s\n", folderName)
	fmt.Printf("   Local path : %s\n", projectPath)

	remoteConfigured := cfg.RemoteName != "" && cfg.RemotePath != ""
	remoteDest := ""
	deleteRemote := false

	if remoteConfigured {
		remoteDest = fmt.Sprintf("%s:%s/%s", cfg.RemoteName, cfg.RemotePath, folderName)
		if remoteExists(remoteDest) {
			fmt.Printf("   Google Drive: %s\n", remoteDest)
			deleteRemote = true
		} else {
			fmt.Printf("   Google Drive: not found, skipping\n")
		}
	}

	fmt.Printf("\nThis cannot be undone. Type the project number to confirm: ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}
	input = strings.TrimSpace(input)

	// Extract project number from folder name (first word)
	parts := strings.SplitN(folderName, " ", 2)
	expectedNumber := parts[0]

	if input != expectedNumber {
		return fmt.Errorf("cancelled — confirmation did not match")
	}

	fmt.Println()

	// 1. Delete local folder
	if err := os.RemoveAll(projectPath); err != nil {
		return fmt.Errorf("could not delete local folder: %w", err)
	}
	fmt.Printf("   ✓ Local folder deleted\n")

	// 2. Remove from CSV
	if err := store.Delete(cfg, folderName); err != nil {
		fmt.Printf("   ⚠ Could not remove from orders.csv: %v\n", err)
	} else {
		fmt.Printf("   ✓ Removed from orders.csv\n")
	}

	// 3. Delete from Google Drive if present
	if deleteRemote {
		if err := deleteFromDrive(remoteDest); err != nil {
			fmt.Printf("   ⚠ Could not delete from Google Drive: %v\n", err)
		} else {
			fmt.Printf("   ✓ Deleted from Google Drive\n")
		}
	}

	fmt.Printf("\n✅ Project deleted: %s\n", folderName)
	return nil
}

// remoteExists checks if the folder exists on the rclone remote.
func remoteExists(dest string) bool {
	cmd := exec.Command("rclone", "lsf", dest)
	err := cmd.Run()
	return err == nil
}

// deleteFromDrive removes a folder from the rclone remote.
func deleteFromDrive(dest string) error {
	cmd := exec.Command("rclone", "purge", dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rclone purge failed: %w", err)
	}
	return nil
}
