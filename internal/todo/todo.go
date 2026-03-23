package todo

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/burkatskyimaksym/projectr/internal/config"
	"github.com/burkatskyimaksym/projectr/internal/fs"
)

const todoFile = "todos.txt"

// Item represents a single todo entry.
type Item struct {
	ID   int
	Done bool
	Text string
}

func (it Item) String() string {
	check := " "
	if it.Done {
		check = "x"
	}
	return fmt.Sprintf("%d [%s] %s", it.ID, check, it.Text)
}

// ── File helpers ──────────────────────────────────────────────────────────────

func todoPath(projectPath string) string {
	return filepath.Join(projectPath, todoFile)
}

func load(projectPath string) ([]Item, error) {
	path := todoPath(projectPath)
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil // empty list is fine
	}
	if err != nil {
		return nil, fmt.Errorf("could not open todos.txt: %w", err)
	}
	defer f.Close()

	var items []Item
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		item, err := parseLine(line)
		if err != nil {
			continue // skip malformed lines
		}
		items = append(items, item)
	}
	return items, scanner.Err()
}

func save(projectPath string, items []Item) error {
	path := todoPath(projectPath)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("could not write todos.txt: %w", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, it := range items {
		fmt.Fprintln(w, it.String())
	}
	return w.Flush()
}

// parseLine parses "3 [x] Some task text".
func parseLine(line string) (Item, error) {
	// Minimum: "1 [ ] x" = 7 chars
	if len(line) < 7 {
		return Item{}, fmt.Errorf("line too short")
	}

	// Find first space separating ID from the rest
	spaceIdx := strings.Index(line, " ")
	if spaceIdx < 1 {
		return Item{}, fmt.Errorf("no space after id")
	}

	id, err := strconv.Atoi(line[:spaceIdx])
	if err != nil {
		return Item{}, fmt.Errorf("invalid id")
	}

	rest := line[spaceIdx+1:] // "[x] Some text"
	if len(rest) < 4 {
		return Item{}, fmt.Errorf("missing checkbox")
	}

	done := rest[1] == 'x'
	text := strings.TrimSpace(rest[4:]) // skip "[x] "

	return Item{ID: id, Done: done, Text: text}, nil
}

// nextID returns max existing ID + 1.
func nextID(items []Item) int {
	max := 0
	for _, it := range items {
		if it.ID > max {
			max = it.ID
		}
	}
	return max + 1
}

// resolveProject finds the project path and returns it.
func resolveProject(cfg *config.Config, name string) (string, error) {
	return fs.FindProject(cfg.OrdersPath, name)
}

// ── Public commands ───────────────────────────────────────────────────────────

// Add appends a new todo item to the project's todos.txt.
func Add(cfg *config.Config, projectName, text string) error {
	projectPath, err := resolveProject(cfg, projectName)
	if err != nil {
		return err
	}

	items, err := load(projectPath)
	if err != nil {
		return err
	}

	item := Item{ID: nextID(items), Done: false, Text: text}
	items = append(items, item)

	if err := save(projectPath, items); err != nil {
		return err
	}

	fmt.Printf("✓ Added [%d]: %s\n", item.ID, item.Text)
	return nil
}

// List prints all todos for a project.
func List(cfg *config.Config, projectName string) error {
	projectPath, err := resolveProject(cfg, projectName)
	if err != nil {
		return err
	}

	items, err := load(projectPath)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		fmt.Println("No todos yet. Add one with: projectr todo <project> add \"task\"")
		return nil
	}

	done, total := 0, len(items)
	for _, it := range items {
		if it.Done {
			done++
		}
	}

	fmt.Printf("\n  %s — todos (%d/%d done)\n\n", filepath.Base(projectPath), done, total)

	for _, it := range items {
		check := "[ ]"
		if it.Done {
			check = "[x]"
		}
		fmt.Printf("  %3d %s %s\n", it.ID, check, it.Text)
	}
	fmt.Println()
	return nil
}

// Done marks a todo item as completed by ID.
func Done(cfg *config.Config, projectName string, id int) error {
	projectPath, err := resolveProject(cfg, projectName)
	if err != nil {
		return err
	}

	items, err := load(projectPath)
	if err != nil {
		return err
	}

	found := false
	for i, it := range items {
		if it.ID == id {
			if it.Done {
				fmt.Printf("  Already done: [%d] %s\n", it.ID, it.Text)
				return nil
			}
			items[i].Done = true
			found = true
			fmt.Printf("✓ Done [%d]: %s\n", it.ID, it.Text)
			break
		}
	}
	if !found {
		return fmt.Errorf("todo #%d not found", id)
	}

	return save(projectPath, items)
}

// Undone marks a completed todo as not done by ID.
func Undone(cfg *config.Config, projectName string, id int) error {
	projectPath, err := resolveProject(cfg, projectName)
	if err != nil {
		return err
	}

	items, err := load(projectPath)
	if err != nil {
		return err
	}

	found := false
	for i, it := range items {
		if it.ID == id {
			items[i].Done = false
			found = true
			fmt.Printf("✓ Reopened [%d]: %s\n", it.ID, it.Text)
			break
		}
	}
	if !found {
		return fmt.Errorf("todo #%d not found", id)
	}

	return save(projectPath, items)
}

// Remove deletes a todo item by ID.
func Remove(cfg *config.Config, projectName string, id int) error {
	projectPath, err := resolveProject(cfg, projectName)
	if err != nil {
		return err
	}

	items, err := load(projectPath)
	if err != nil {
		return err
	}

	filtered := make([]Item, 0, len(items))
	found := false
	for _, it := range items {
		if it.ID == id {
			found = true
			fmt.Printf("✓ Removed [%d]: %s\n", it.ID, it.Text)
			continue
		}
		filtered = append(filtered, it)
	}
	if !found {
		return fmt.Errorf("todo #%d not found", id)
	}

	return save(projectPath, filtered)
}

// Clear removes all completed todos from the list.
func Clear(cfg *config.Config, projectName string) error {
	projectPath, err := resolveProject(cfg, projectName)
	if err != nil {
		return err
	}

	items, err := load(projectPath)
	if err != nil {
		return err
	}

	filtered := make([]Item, 0, len(items))
	removed := 0
	for _, it := range items {
		if it.Done {
			removed++
			continue
		}
		filtered = append(filtered, it)
	}

	if removed == 0 {
		fmt.Println("No completed todos to clear.")
		return nil
	}

	if err := save(projectPath, filtered); err != nil {
		return err
	}

	fmt.Printf("✓ Cleared %d completed todo(s).\n", removed)
	return nil
}
