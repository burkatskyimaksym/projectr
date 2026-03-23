package open

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/burkatskyimaksym/projectr/internal/config"
	"github.com/burkatskyimaksym/projectr/internal/fs"
)

// Open opens the project folder in the system file manager.
func Open(cfg *config.Config, name string) error {
	projectPath, err := fs.FindProject(cfg.OrdersPath, name)
	if err != nil {
		return err
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", projectPath)
	case "windows":
		cmd = exec.Command("explorer", projectPath)
	default: // linux and others
		cmd = exec.Command("xdg-open", projectPath)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not open folder: %w", err)
	}

	fmt.Printf("📂 Opened: %s\n", projectPath)
	return nil
}
