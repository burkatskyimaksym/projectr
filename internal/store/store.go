package store

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/burkatskyimaksym/projectr/internal/config"
)

const csvFile = "orders.csv"

var csvHeaders = []string{"name", "client", "created", "deadline", "priority", "status"}

// clientRe extracts the username from "(nickname)" at the end of a project name.
var clientRe = regexp.MustCompile(`\(([^)]+)\)\s*$`)

// Order represents a single freelance order entry.
type Order struct {
	Name     string
	Client   string
	Created  string
	Deadline string
	Priority string // high | medium | low | ""
	Status   string
}

// ListFilters controls which orders are shown in List.
type ListFilters struct {
	OnlyOverdue bool
	OnlyDone    bool
	Month       string // "03/2026" format, empty = all
	Client      string // filter by client nickname
}

// ExtractClient parses the client nickname from a project folder name.
// "35 Logo redesign (maria22)" → "maria22"
func ExtractClient(name string) string {
	m := clientRe.FindStringSubmatch(name)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func csvPath(cfg *config.Config) string {
	return filepath.Join(cfg.OrdersPath, csvFile)
}

// ensureCSV creates orders.csv with a header row if it doesn't exist.
func ensureCSV(cfg *config.Config) error {
	path := csvPath(cfg)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()
		w := csv.NewWriter(f)
		w.Write(csvHeaders)
		w.Flush()
	}
	return nil
}

// Migrate upgrades an old CSV (4 columns) to the new format (6 columns).
// Safe to call on already-migrated files — checks header before acting.
func Migrate(cfg *config.Config) error {
	path := csvPath(cfg)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // nothing to migrate
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	rows, err := csv.NewReader(f).ReadAll()
	f.Close()
	if err != nil || len(rows) == 0 {
		return err
	}

	// Already migrated — header has 6 columns
	if len(rows[0]) >= 6 {
		return nil
	}

	fmt.Println("⚙  Migrating orders.csv to new format (adding client, priority columns)...")

	// Rewrite with new columns: name, client, created, deadline, priority, status
	// Old format:                 name, created, deadline, status
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	w := csv.NewWriter(out)
	w.Write(csvHeaders)
	for i, row := range rows {
		if i == 0 {
			continue // skip old header
		}
		if len(row) < 4 {
			continue
		}
		name := row[0]
		created := row[1]
		deadline := row[2]
		status := row[3]
		client := ExtractClient(name)
		w.Write([]string{name, client, created, deadline, "", status})
	}
	w.Flush()
	fmt.Println("✓  Migration complete.\n")
	return w.Error()
}

// orderToRow serialises an Order to a CSV row.
func orderToRow(o Order) []string {
	return []string{o.Name, o.Client, o.Created, o.Deadline, o.Priority, o.Status}
}

// Append adds a new order row to orders.csv.
func Append(cfg *config.Config, o Order) error {
	if err := ensureCSV(cfg); err != nil {
		return err
	}
	// Auto-fill client if not set
	if o.Client == "" {
		o.Client = ExtractClient(o.Name)
	}
	f, err := os.OpenFile(csvPath(cfg), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	w.Write(orderToRow(o))
	w.Flush()
	return w.Error()
}

// Load reads all orders from orders.csv.
func Load(cfg *config.Config) ([]Order, error) {
	if err := ensureCSV(cfg); err != nil {
		return nil, err
	}
	f, err := os.Open(csvPath(cfg))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}

	var orders []Order
	for i, row := range rows {
		if i == 0 {
			continue // skip header
		}
		o := rowToOrder(row)
		if o == nil {
			continue
		}
		orders = append(orders, *o)
	}
	return orders, nil
}

// rowToOrder handles both old (4-col) and new (6-col) rows gracefully.
func rowToOrder(row []string) *Order {
	switch len(row) {
	case 4: // legacy
		return &Order{
			Name:     row[0],
			Client:   ExtractClient(row[0]),
			Created:  row[1],
			Deadline: row[2],
			Status:   row[3],
		}
	case 6:
		return &Order{
			Name:     row[0],
			Client:   row[1],
			Created:  row[2],
			Deadline: row[3],
			Priority: row[4],
			Status:   row[5],
		}
	default:
		return nil
	}
}

// writeAll rewrites the entire CSV with current headers and given orders.
func writeAll(cfg *config.Config, orders []Order) error {
	f, err := os.Create(csvPath(cfg))
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	w.Write(csvHeaders)
	for _, o := range orders {
		w.Write(orderToRow(o))
	}
	w.Flush()
	return w.Error()
}

// UpdateStatus sets a new status for the order matching name.
func UpdateStatus(cfg *config.Config, name, status string) error {
	orders, err := Load(cfg)
	if err != nil {
		return err
	}
	found := false
	for i, o := range orders {
		if o.Name == name {
			orders[i].Status = status
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("order not found: %s", name)
	}
	return writeAll(cfg, orders)
}

// Delete removes the order matching name from orders.csv.
func Delete(cfg *config.Config, name string) error {
	orders, err := Load(cfg)
	if err != nil {
		return err
	}
	filtered := make([]Order, 0, len(orders))
	found := false
	for _, o := range orders {
		if o.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, o)
	}
	if !found {
		return fmt.Errorf("order not found: %s", name)
	}
	return writeAll(cfg, filtered)
}

// List prints a filtered, formatted table of orders to stdout.
func List(cfg *config.Config, f ListFilters) error {
	orders, err := Load(cfg)
	if err != nil {
		return err
	}

	today, _ := time.Parse("02/01/2006", time.Now().Format("02/01/2006"))
	filtered := applyFilters(orders, f, today)

	if len(filtered) == 0 {
		fmt.Println("No orders match the given filters.")
		return nil
	}

	fmt.Println()
	fmt.Printf("  %s  %s  %s  %s  %s\n",
		col("PROJECT", 34),
		col("CLIENT", 12),
		col("DEADLINE", 12),
		col("PRIORITY", 8),
		col("STATUS", 12),
	)
	fmt.Println("  " + strings.Repeat("─", 84))

	for _, o := range filtered {
		deadline := o.Deadline
		if deadline == "" {
			deadline = "—"
		}
		priority := o.Priority
		if priority == "" {
			priority = "—"
		}
		client := o.Client
		if client == "" {
			client = "—"
		}

		warning := ""
		if deadline != "—" && o.Status != "done" {
			if d, err := time.Parse("02/01/2006", deadline); err == nil && d.Before(today) {
				warning = "⚠ overdue"
			}
		}

		fmt.Printf("  %s  %s  %s  %s  %s  %s\n",
			col(o.Name, 34),
			col(client, 12),
			col(deadline, 12),
			col(priority, 8),
			col(o.Status, 12),
			warning,
		)
	}
	fmt.Println()
	return nil
}

// History prints all orders for a specific client.
func History(cfg *config.Config, client string) error {
	orders, err := Load(cfg)
	if err != nil {
		return err
	}

	clientLower := strings.ToLower(client)
	var matches []Order
	for _, o := range orders {
		if strings.ToLower(o.Client) == clientLower {
			matches = append(matches, o)
		}
	}

	if len(matches) == 0 {
		fmt.Printf("No orders found for client: %s\n", client)
		return nil
	}

	today, _ := time.Parse("02/01/2006", time.Now().Format("02/01/2006"))

	fmt.Printf("\n  Orders for client: %s (%d total)\n\n", client, len(matches))
	fmt.Printf("  %s  %s  %s  %s\n",
		col("PROJECT", 38),
		col("DEADLINE", 12),
		col("PRIORITY", 8),
		col("STATUS", 12),
	)
	fmt.Println("  " + strings.Repeat("─", 76))

	done := 0
	for _, o := range matches {
		deadline := o.Deadline
		if deadline == "" {
			deadline = "—"
		}
		priority := o.Priority
		if priority == "" {
			priority = "—"
		}

		warning := ""
		if deadline != "—" && o.Status != "done" {
			if d, err := time.Parse("02/01/2006", deadline); err == nil && d.Before(today) {
				warning = "⚠ overdue"
			}
		}
		if o.Status == "done" {
			done++
		}

		fmt.Printf("  %s  %s  %s  %s  %s\n",
			col(o.Name, 38),
			col(deadline, 12),
			col(priority, 8),
			col(o.Status, 12),
			warning,
		)
	}
	fmt.Printf("\n  Done: %d / %d\n\n", done, len(matches))
	return nil
}

// applyFilters returns a subset of orders matching the given filters.
func applyFilters(orders []Order, f ListFilters, today time.Time) []Order {
	var out []Order
	for _, o := range orders {
		if f.OnlyDone && o.Status != "done" {
			continue
		}
		if f.Client != "" && !strings.EqualFold(o.Client, f.Client) {
			continue
		}
		if f.OnlyOverdue {
			if o.Status == "done" || o.Deadline == "" {
				continue
			}
			d, err := time.Parse("02/01/2006", o.Deadline)
			if err != nil || !d.Before(today) {
				continue
			}
		}
		if f.Month != "" {
			// f.Month is "03/2026" — match against Created "dd/mm/yyyy"
			if len(o.Created) < 7 || o.Created[3:] != f.Month {
				continue
			}
		}
		out = append(out, o)
	}
	return out
}

// col pads or truncates s to exactly w characters.
func col(s string, w int) string {
	if len(s) >= w {
		return s[:w-1] + "…"
	}
	return s + strings.Repeat(" ", w-len(s))
}
