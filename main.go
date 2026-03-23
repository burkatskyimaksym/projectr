package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/burkatskyimaksym/projectr/internal/config"
	"github.com/burkatskyimaksym/projectr/internal/delete"
	"github.com/burkatskyimaksym/projectr/internal/open"
	"github.com/burkatskyimaksym/projectr/internal/project"
	"github.com/burkatskyimaksym/projectr/internal/store"
	"github.com/burkatskyimaksym/projectr/internal/upload"
	"github.com/burkatskyimaksym/projectr/internal/watch"
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
		if err := store.Migrate(cfg); err != nil {
			fatal("Migration error: %v", err)
		}
		cmdList(cfg)

	case "history":
		if len(os.Args) < 3 {
			fatal("Error: provide client nickname\n  projectr history Alexs1")
		}
		cfg := mustLoadConfig()
		if err := store.Migrate(cfg); err != nil {
			fatal("Migration error: %v", err)
		}
		client := strings.Join(os.Args[2:], " ")
		if err := store.History(cfg, client); err != nil {
			fatal("Error: %v", err)
		}

	case "done":
		cmdDone()

	case "open":
		if len(os.Args) < 3 {
			fatal("Error: provide project name\n  projectr open 35")
		}
		cfg := mustLoadConfig()
		name := strings.Join(os.Args[2:], " ")
		if err := open.Open(cfg, name); err != nil {
			fatal("Error: %v", err)
		}

	case "watch":
		if len(os.Args) < 3 {
			fatal("Error: provide project name\n  projectr watch 35")
		}
		cfg := mustLoadConfig()
		if cfg.RemoteName == "" || cfg.RemotePath == "" {
			if err := config.SetupUpload(cfg); err != nil {
				fatal("Setup error: %v", err)
			}
		}
		name := strings.Join(os.Args[2:], " ")
		if err := watch.Watch(cfg, name); err != nil {
			fatal("Error: %v", err)
		}

	case "import":
		cfg := mustLoadConfig()
		if err := store.Migrate(cfg); err != nil {
			fatal("Migration error: %v", err)
		}
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
			fatal("Error: provide project name\n  projectr upload 35")
		}
		cfg := mustLoadConfig()
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

// cmdDone handles: projectr done <name> [--upload]
func cmdDone() {
	if len(os.Args) < 3 {
		fatal("Error: provide project name")
	}

	// Separate --upload flag from the project name args
	fs := flag.NewFlagSet("done", flag.ExitOnError)
	doUpload := fs.Bool("upload", false, "upload final/ to Google Drive after marking done")

	// Collect non-flag args as the project name
	var nameParts []string
	var flagArgs []string
	for _, a := range os.Args[2:] {
		if strings.HasPrefix(a, "--") || strings.HasPrefix(a, "-") {
			flagArgs = append(flagArgs, a)
		} else {
			nameParts = append(nameParts, a)
		}
	}
	fs.Parse(flagArgs)
	name := strings.Join(nameParts, " ")

	if name == "" {
		fatal("Error: provide project name")
	}

	cfg := mustLoadConfig()

	// Mark as done in CSV
	if err := store.UpdateStatus(cfg, name, "done"); err != nil {
		fatal("Error: %v", err)
	}
	fmt.Printf("✓ Marked as done: %s\n", name)

	// Optionally upload
	if *doUpload {
		if cfg.RemoteName == "" || cfg.RemotePath == "" {
			if err := config.SetupUpload(cfg); err != nil {
				fatal("Setup error: %v", err)
			}
		}
		if err := upload.Upload(cfg, name); err != nil {
			fatal("Upload error: %v", err)
		}
	}
}

// cmdList parses list flags and calls store.List.
func cmdList(cfg *config.Config) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	overdue := fs.Bool("overdue", false, "show only overdue orders")
	done := fs.Bool("done", false, "show only completed orders")
	month := fs.String("month", "", "filter by creation month, e.g. 03/2026")
	client := fs.String("client", "", "filter by client nickname")
	fs.Parse(os.Args[2:])

	filters := store.ListFilters{
		OnlyOverdue: *overdue,
		OnlyDone:    *done,
		Month:       *month,
		Client:      *client,
	}
	if err := store.List(cfg, filters); err != nil {
		fatal("Error: %v", err)
	}
}

// cmdCreate handles: projectr <name> [-d deadline] [-p priority] [-s file ...]
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
	priority := fs.String("p", "", "priority: high, medium, or low")
	fs.Parse(args)

	cfg := mustLoadConfig()
	if err := store.Migrate(cfg); err != nil {
		fatal("Migration error: %v", err)
	}
	if err := project.Create(cfg, projectName, *deadline, *priority, sources); err != nil {
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
	fmt.Println("  projectr <project-name> [-d dd/mm/yyyy] [-p high|medium|low] [-s file ...]")
	fmt.Println("  projectr list [--overdue] [--done] [--month mm/yyyy] [--client nickname]")
	fmt.Println("  projectr history <client>")
	fmt.Println("  projectr done <project-name> [--upload]")
	fmt.Println("  projectr open <project-name>")
	fmt.Println("  projectr watch <project-name>")
	fmt.Println("  projectr upload <project-name>")
	fmt.Println("  projectr delete <project-name>")
	fmt.Println("  projectr import")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  projectr \"35 Logo redesign (maria22)\" -d 25/03/2026 -p high -s brief.pdf")
	fmt.Println("  projectr list --overdue")
	fmt.Println("  projectr list --client Alexs1")
	fmt.Println("  projectr history Alexs1")
	fmt.Println("  projectr done 35 --upload")
	fmt.Println("  projectr open 35")
	fmt.Println("  projectr watch 35")
	fmt.Println("  projectr upload 35")
	fmt.Println("  projectr delete 35")
	fmt.Println("")
	fmt.Println("Flags (new project):")
	fmt.Println("  -d        deadline in dd/mm/yyyy format (optional)")
	fmt.Println("  -p        priority: high, medium, or low (optional)")
	fmt.Println("  -s        source files to copy into src/")
	fmt.Println("  --config  print path to config file")
	fmt.Println("  --reset   reset configuration")
}
