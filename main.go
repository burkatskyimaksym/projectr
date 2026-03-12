package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const configDir = ".config/projectr"
const configFile = "config"

type Config struct {
	OrdersPath string
}

func getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not find home directory: %w", err)
	}
	return filepath.Join(home, configDir, configFile), nil
}

func loadConfig() (*Config, error) {
	cfgPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // config doesn't exist — first run
		}
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	cfg := &Config{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key == "OrderPath" {
			cfg.OrdersPath = val
		}
	}

	if cfg.OrdersPath == "" {
		return nil, fmt.Errorf("OrderPath not found in config")
	}
	return cfg, nil
}

func saveConfig(cfg *Config) error {
	cfgPath, err := getConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}

	content := fmt.Sprintf("OrderPath=%s\n", cfg.OrdersPath)
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("error writing config: %w", err)
	}
	return nil
}

func setupConfig() (*Config, error) {
	reader := bufio.NewReader(os.Stdin)
	home, _ := os.UserHomeDir()
	defaultPath := filepath.Join(home, "Documents", "Orders")

	fmt.Println("┌─────────────────────────────────────────┐")
	fmt.Println("│          projectr — first run            │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Printf("\nEnter path to your orders folder\n")
	fmt.Printf("(press Enter to use: %s): ", defaultPath)

	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("error reading input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input == "" {
		input = defaultPath
	}

	// Expand ~ if present
	if strings.HasPrefix(input, "~/") {
		input = filepath.Join(home, input[2:])
	}

	// Create folder if it doesn't exist
	if _, err := os.Stat(input); os.IsNotExist(err) {
		fmt.Printf("\nFolder does not exist. Create \"%s\"? [Y/n]: ", input)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "" && answer != "y" && answer != "yes" {
			return nil, fmt.Errorf("cancelled by user")
		}
		if err := os.MkdirAll(input, 0755); err != nil {
			return nil, fmt.Errorf("could not create folder: %w", err)
		}
		fmt.Printf("✓ Folder created: %s\n", input)
	} else {
		fmt.Printf("✓ Using existing folder: %s\n", input)
	}

	cfg := &Config{OrdersPath: input}
	if err := saveConfig(cfg); err != nil {
		return nil, err
	}

	cfgPath, _ := getConfigPath()
	fmt.Printf("✓ Config saved: %s\n\n", cfgPath)
	return cfg, nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func sanitizeName(name string) string {
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
	)
	return strings.TrimSpace(replacer.Replace(name))
}

func createProject(cfg *Config, projectName string, sources []string) error {
	safeName := sanitizeName(projectName)
	projectPath := filepath.Join(cfg.OrdersPath, safeName)

	// Check if project already exists
	if _, err := os.Stat(projectPath); !os.IsNotExist(err) {
		return fmt.Errorf("project already exists: %s", projectPath)
	}

	subDirs := []string{"src", "drafts", "final", "references"}

	fmt.Printf("\n📁 Creating project: %s\n", safeName)
	fmt.Printf("   Path: %s\n\n", projectPath)

	for _, sub := range subDirs {
		path := filepath.Join(projectPath, sub)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("error creating subfolder %s: %w", sub, err)
		}
		fmt.Printf("   ✓ /%s\n", sub)
	}

	// Copy source files
	if len(sources) > 0 {
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
					fmt.Printf("   ✗ %s — directories are not supported, specify files\n", match)
					continue
				}

				dst := filepath.Join(projectPath, "src", filepath.Base(match))
				if err := copyFile(match, dst); err != nil {
					fmt.Printf("   ✗ %s — error: %v\n", match, err)
					continue
				}
				fmt.Printf("   ✓ %s\n", filepath.Base(match))
			}
		}
	}

	// Create README
	readmePath := filepath.Join(projectPath, "README.md")
	readmeContent := fmt.Sprintf("# %s\n\nCreated: %s\n\n## Structure\n- `src/` — source files from client\n- `drafts/` — work in progress\n- `final/` — deliverables for client\n- `references/` — reference materials\n",
		projectName, time.Now().Format("2006-01-02"))
	os.WriteFile(readmePath, []byte(readmeContent), 0644)
	fmt.Printf("\n   ✓ README.md\n")

	fmt.Printf("\n✅ Done! Project created:\n   %s\n", projectPath)
	return nil
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  projectr <project-name> [-s file1 file2 ...]")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  projectr \"34 Branches and borders (Alexs1)\"")
	fmt.Println("  projectr \"35 Logo redesign (maria22)\" -s brief.pdf logo_v1.ai")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  -s        source files to copy into src/")
	fmt.Println("  --config  print path to config file")
	fmt.Println("  --reset   reset configuration")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "--config":
		cfgPath, _ := getConfigPath()
		fmt.Println(cfgPath)
		return
	case "--reset":
		cfgPath, _ := getConfigPath()
		os.Remove(cfgPath)
		fmt.Println("✓ Config removed. You will be prompted for settings on next run.")
		return
	case "--help", "-h":
		printUsage()
		return
	}

	projectName := os.Args[1]
	var sources []string

	args := os.Args[2:]
	fs := flag.NewFlagSet("projectr", flag.ExitOnError)
	fs.Usage = printUsage

	sIdx := -1
	for i, a := range args {
		if a == "-s" {
			sIdx = i
			break
		}
	}

	if sIdx != -1 {
		sources = args[sIdx+1:]
		args = args[:sIdx]
	}

	fs.Parse(args)

	if projectName == "" {
		fmt.Fprintln(os.Stderr, "Error: project name is required")
		printUsage()
		os.Exit(1)
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}

	if cfg == nil {
		cfg, err = setupConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Setup error: %v\n", err)
			os.Exit(1)
		}
	}

	if err := createProject(cfg, projectName, sources); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
