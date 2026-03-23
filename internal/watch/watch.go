package watch

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/burkatskyimaksym/projectr/internal/config"
	"github.com/burkatskyimaksym/projectr/internal/fs"
	"github.com/burkatskyimaksym/projectr/internal/upload"
	"github.com/fsnotify/fsnotify"
)

// Watch monitors the final/ folder of a project and uploads new or changed
// files to Google Drive automatically. Runs until interrupted with Ctrl+C.
func Watch(cfg *config.Config, name string) error {
	if err := upload.CheckRclone(); err != nil {
		return err
	}
	if cfg.RemoteName == "" || cfg.RemotePath == "" {
		return fmt.Errorf("rclone is not configured — run: projectr upload <n> first")
	}

	projectPath, err := fs.FindProject(cfg.OrdersPath, name)
	if err != nil {
		return err
	}

	finalDir := filepath.Join(projectPath, "final")
	if err := os.MkdirAll(finalDir, 0755); err != nil {
		return fmt.Errorf("could not access final/ folder: %w", err)
	}

	folderName := filepath.Base(projectPath)
	dest := fmt.Sprintf("%s:%s/%s", cfg.RemoteName, cfg.RemotePath, folderName)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("could not create watcher: %w", err)
	}
	defer watcher.Close()

	if err := watcher.Add(finalDir); err != nil {
		return fmt.Errorf("could not watch folder: %w", err)
	}

	fmt.Printf("\n👁  Watching: %s\n", finalDir)
	fmt.Printf("   Remote  : %s\n", dest)
	fmt.Println("   Press Ctrl+C to stop.\n")

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			// Only react to writes and new files, ignore temp/hidden files
			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
				continue
			}
			base := filepath.Base(event.Name)
			if len(base) > 0 && base[0] == '.' {
				continue // skip hidden/temp files e.g. .DS_Store
			}

			fmt.Printf("[%s] Changed: %s\n",
				time.Now().Format("15:04:05"),
				base,
			)

			if err := upload.Upload(cfg, folderName); err != nil {
				fmt.Printf("⚠  Upload failed: %v\n", err)
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Printf("⚠  Watcher error: %v\n", err)
		}
	}
}
