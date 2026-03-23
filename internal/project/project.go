package project

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/burkatskyimaksym/projectr/internal/config"
	"github.com/burkatskyimaksym/projectr/internal/store"
)

var subDirs = []string{"src", "drafts", "final", "references"}

// Create scaffolds a new project folder and registers it in orders.csv.
func Create(cfg *config.Config, name, deadline, priority string, sources []string) error {
	if deadline != "" {
		if _, err := time.Parse("02/01/2006", deadline); err != nil {
			return fmt.Errorf("invalid deadline format, use dd/mm/yyyy (e.g. 25/03/2026)")
		}
	}
	if priority != "" && priority != "high" && priority != "medium" && priority != "low" {
		return fmt.Errorf("invalid priority %q — use: high, medium, or low", priority)
	}

	safeName := sanitize(name)
	projectPath := filepath.Join(cfg.OrdersPath, safeName)

	if _, err := os.Stat(projectPath); !os.IsNotExist(err) {
		return fmt.Errorf("project already exists: %s", projectPath)
	}

	fmt.Printf("\n📁 Creating project: %s\n", safeName)
	fmt.Printf("   Path: %s\n\n", projectPath)

	for _, sub := range subDirs {
		if err := os.MkdirAll(filepath.Join(projectPath, sub), 0755); err != nil {
			return fmt.Errorf("error creating subfolder %s: %w", sub, err)
		}
		fmt.Printf("   ✓ /%s\n", sub)
	}

	if err := copySourceFiles(projectPath, sources); err != nil {
		return err
	}

	if err := writeReadme(projectPath, name, deadline); err != nil {
		fmt.Printf("   ⚠ Could not write README.md: %v\n", err)
	} else {
		fmt.Printf("\n   ✓ README.md\n")
	}

	if err := store.Append(cfg, store.Order{
		Name:     name,
		Created:  time.Now().Format("02/01/2006"),
		Deadline: deadline,
		Priority: priority,
		Status:   "in progress",
	}); err != nil {
		fmt.Printf("   ⚠ Could not update orders.csv: %v\n", err)
	} else {
		fmt.Printf("   ✓ orders.csv\n")
	}

	fmt.Printf("\n✅ Done! Project created:\n   %s\n", projectPath)
	return nil
}

// Import scans the orders folder and adds any untracked directories to orders.csv.
func Import(cfg *config.Config) error {
	entries, err := os.ReadDir(cfg.OrdersPath)
	if err != nil {
		return fmt.Errorf("could not read orders folder: %w", err)
	}

	existing, err := store.Load(cfg)
	if err != nil {
		return err
	}
	known := make(map[string]bool, len(existing))
	for _, o := range existing {
		known[o.Name] = true
	}

	fmt.Println()
	imported, skipped := 0, 0

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if known[name] {
			fmt.Printf("  ~ %-40s already tracked\n", name)
			skipped++
			continue
		}

		created := time.Now().Format("02/01/2006")
		if info, err := e.Info(); err == nil {
			created = info.ModTime().Format("02/01/2006")
		}

		if err := store.Append(cfg, store.Order{
			Name:    name,
			Created: created,
			Status:  "in progress",
		}); err != nil {
			fmt.Printf("  ✗ %-40s error: %v\n", name, err)
			continue
		}
		fmt.Printf("  ✓ %-40s imported\n", name)
		imported++
	}

	fmt.Printf("\nDone: %d imported, %d already tracked.\n\n", imported, skipped)
	return nil
}

// copySourceFiles copies a list of files into projectPath/src/.
func copySourceFiles(projectPath string, sources []string) error {
	if len(sources) == 0 {
		return nil
	}
	fmt.Printf("\n📋 Copying files to /src:\n")
	for _, src := range sources {
		matches, err := filepath.Glob(src)
		if err != nil || len(matches) == 0 {
			matches = []string{src}
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				fmt.Printf("   ✗ %s — not found\n", match)
				continue
			}
			if info.IsDir() {
				fmt.Printf("   ✗ %s — directories are not supported\n", match)
				continue
			}
			dst := filepath.Join(projectPath, "src", filepath.Base(match))
			if err := copyFile(match, dst); err != nil {
				fmt.Printf("   ✗ %s — error: %v\n", filepath.Base(match), err)
				continue
			}
			fmt.Printf("   ✓ %s\n", filepath.Base(match))
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func writeReadme(projectPath, name, deadline string) error {
	dl := "not set"
	if deadline != "" {
		dl = deadline
	}
	content := fmt.Sprintf(
		"# %s\n\nCreated: %s\nDeadline: %s\n\n## Structure\n- `src/` — source files from client\n- `drafts/` — work in progress\n- `final/` — deliverables for client\n- `references/` — reference materials\n",
		name, time.Now().Format("2006-01-02"), dl,
	)
	return os.WriteFile(filepath.Join(projectPath, "README.md"), []byte(content), 0644)
}

// sanitize removes characters that are invalid in folder names.
func sanitize(name string) string {
	r := strings.NewReplacer(
		"/", "-", "\\", "-", ":", "-",
		"*", "-", "?", "-", "\"", "-",
		"<", "-", ">", "-", "|", "-",
	)
	return strings.TrimSpace(r.Replace(name))
}
