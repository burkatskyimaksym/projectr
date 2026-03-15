package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/burkatskyimaksym/projectr/internal/config"
	"github.com/burkatskyimaksym/projectr/internal/delete"
	"github.com/burkatskyimaksym/projectr/internal/project"
	"github.com/burkatskyimaksym/projectr/internal/store"
	"github.com/burkatskyimaksym/projectr/internal/upload"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "--help", "-h":
		printUsage()

	case "--config":
		path, err := config.Path()
		if err != nil {
			fatal("Error: %v", err)
		}
		fmt.Println(path)

	case "--reset":
		path, _ := config.Path()
		os.Remove(path)
		fmt.Println("✓ Config removed. You will be prompted for settings on next run.")

	case "list":
		cfg := mustLoadConfig()
		if err := store.List(cfg); err != nil {
			fatal("Error: %v", err)
		}

	case "done":
		if len(os.Args) < 3 {
			fatal("Error: provide project name")
		}
		cfg := mustLoadConfig()
		name := strings.Join(os.Args[2:], " ")
		if err := store.UpdateStatus(cfg, name, "done"); err != nil {
			fatal("Error: %v", err)
		}
		fmt.Printf("✓ Marked as done: %s\n", name)

	case "import":
		cfg := mustLoadConfig()
		if err := project.Import(cfg); err != nil {
			fatal("Error: %v", err)
		}

	case "delete":
		if len(os.Args) < 3 {
			fatal("Error: provide project name\n  projectr delete \"35 Logo redesign (maria22)\"")
		}
		cfg := mustLoadConfig()
		name := strings.Join(os.Args[2:], " ")
		if err := delete.Delete(cfg, name); err != nil {
			fatal("Error: %v", err)
		}

	case "upload":
		if len(os.Args) < 3 {
			fatal("Error: provide project name\n  projectr upload \"35 Logo redesign (maria22)\"")
		}
		cfg := mustLoadConfig()

		// First-time rclone setup if not configured yet
		if cfg.RemoteName == "" || cfg.RemotePath == "" {
			if err := config.SetupUpload(cfg); err != nil {
				fatal("Setup error: %v", err)
			}
		}

		name := strings.Join(os.Args[2:], " ")
		if err := upload.Upload(cfg, name); err != nil {
			fatal("Error: %v", err)
		}

	default:
		cmdCreate()
	}
}

// cmdCreate handles: projectr <name> [-d deadline] [-s file ...]
func cmdCreate() {
	projectName := os.Args[1]
	args := os.Args[2:]

	// Split off -s and everything after it before flag parsing
	var sources []string
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

	fs := flag.NewFlagSet("projectr", flag.ExitOnError)
	fs.Usage = printUsage
	deadline := fs.String("d", "", "deadline in dd/mm/yyyy format")
	fs.Parse(args)

	cfg := mustLoadConfig()
	if err := project.Create(cfg, projectName, *deadline, sources); err != nil {
		fatal("Error: %v", err)
	}
}

// mustLoadConfig loads config, running interactive setup on first run.
func mustLoadConfig() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		fatal("Config error: %v", err)
	}
	if cfg == nil {
		cfg, err = config.Setup()
		if err != nil {
			fatal("Setup error: %v", err)
		}
	}
	return cfg
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  projectr <project-name> [-d dd/mm/yyyy] [-s file1 file2 ...]")
	fmt.Println("  projectr list")
	fmt.Println("  projectr done <project-name>")
	fmt.Println("  projectr delete <project-name>")
	fmt.Println("  projectr upload <project-name>")
	fmt.Println("  projectr import")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  projectr \"34 Branches and borders (Alexs1)\"")
	fmt.Println("  projectr \"35 Logo redesign (maria22)\" -d 25/03/2026 -s brief.pdf")
	fmt.Println("  projectr list")
	fmt.Println("  projectr done \"35 Logo redesign (maria22)\"")
	fmt.Println("  projectr delete \"35 Logo redesign (maria22)\"")
	fmt.Println("  projectr delete 35")
	fmt.Println("  projectr upload \"35 Logo redesign (maria22)\"")
	fmt.Println("  projectr upload 35")
	fmt.Println("  projectr import")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  -d        deadline in dd/mm/yyyy format (optional)")
	fmt.Println("  -s        source files to copy into src/")
	fmt.Println("  --config  print path to config file")
	fmt.Println("  --reset   reset configuration")
}
