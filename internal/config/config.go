package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const configDir = ".config/projectr"
const configFile = "config"

// Config holds application settings.
type Config struct {
	OrdersPath string
	RemoteName string // rclone remote name, e.g. "gdrive"
	RemotePath string // base path on remote, e.g. "Orders"
}

// Path returns the absolute path to the config file.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not find home directory: %w", err)
	}
	return filepath.Join(home, configDir, configFile), nil
}

// Load reads the config file. Returns nil, nil if file does not exist yet.
func Load() (*Config, error) {
	cfgPath, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
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
		switch key {
		case "OrderPath":
			cfg.OrdersPath = val
		case "RemoteName":
			cfg.RemoteName = val
		case "RemotePath":
			cfg.RemotePath = val
		}
	}

	if cfg.OrdersPath == "" {
		return nil, fmt.Errorf("OrderPath not found in config")
	}
	return cfg, nil
}

// Save writes the config to disk, creating directories as needed.
func Save(cfg *Config) error {
	cfgPath, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("OrderPath=%s\n", cfg.OrdersPath))
	if cfg.RemoteName != "" {
		sb.WriteString(fmt.Sprintf("RemoteName=%s\n", cfg.RemoteName))
	}
	if cfg.RemotePath != "" {
		sb.WriteString(fmt.Sprintf("RemotePath=%s\n", cfg.RemotePath))
	}

	if err := os.WriteFile(cfgPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("error writing config: %w", err)
	}
	return nil
}

// Setup runs an interactive first-run prompt and saves the result.
func Setup() (*Config, error) {
	reader := bufio.NewReader(os.Stdin)
	home, _ := os.UserHomeDir()
	defaultPath := filepath.Join(home, "Documents", "Orders")

	fmt.Println("┌─────────────────────────────────────────┐")
	fmt.Println("│          projectr — first run           │")
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
	if strings.HasPrefix(input, "~/") {
		input = filepath.Join(home, input[2:])
	}

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
	if err := Save(cfg); err != nil {
		return nil, err
	}
	cfgPath, _ := Path()
	fmt.Printf("✓ Config saved: %s\n\n", cfgPath)
	return cfg, nil
}

// SetupUpload prompts for rclone remote settings and saves them.
// Called automatically the first time "upload" is used without config.
func SetupUpload(cfg *Config) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("┌─────────────────────────────────────────┐")
	fmt.Println("│        projectr — rclone setup          │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Println("\nRun `rclone listremotes` to see your configured remotes.")

	fmt.Printf("\nEnter rclone remote name (e.g. gdrive): ")
	remoteName, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}
	remoteName = strings.TrimSpace(remoteName)
	if remoteName == "" {
		return fmt.Errorf("remote name cannot be empty")
	}

	fmt.Printf("Enter remote folder path (e.g. Orders): ")
	remotePath, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}
	remotePath = strings.TrimSpace(remotePath)
	if remotePath == "" {
		return fmt.Errorf("remote path cannot be empty")
	}

	cfg.RemoteName = remoteName
	cfg.RemotePath = remotePath

	if err := Save(cfg); err != nil {
		return err
	}

	fmt.Printf("\n✓ Upload config saved.\n")
	fmt.Printf("  Remote: %s:%s\n\n", cfg.RemoteName, cfg.RemotePath)
	return nil
}
