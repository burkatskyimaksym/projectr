package store

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/burkatskyimaksym/projectr/internal/config"
)

const csvFile = "orders.csv"

var csvHeaders = []string{"name", "created", "deadline", "status"}

// Order represents a single freelance order entry.
type Order struct {
	Name     string
	Created  string
	Deadline string
	Status   string
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

// Append adds a new order row to orders.csv.
func Append(cfg *config.Config, o Order) error {
	if err := ensureCSV(cfg); err != nil {
		return err
	}
	f, err := os.OpenFile(csvPath(cfg), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	w.Write([]string{o.Name, o.Created, o.Deadline, o.Status})
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
		if i == 0 || len(row) < 4 {
			continue
		}
		orders = append(orders, Order{
			Name:     row[0],
			Created:  row[1],
			Deadline: row[2],
			Status:   row[3],
		})
	}
	return orders, nil
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

	f, err := os.Create(csvPath(cfg))
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	w.Write(csvHeaders)
	for _, o := range orders {
		w.Write([]string{o.Name, o.Created, o.Deadline, o.Status})
	}
	w.Flush()
	return w.Error()
}

// Delete removes the order matching name from orders.csv.
func Delete(cfg *config.Config, name string) error {
	orders, err := Load(cfg)
	if err != nil {
		return err
	}

	filtered := orders[:0]
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

	f, err := os.Create(csvPath(cfg))
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	w.Write(csvHeaders)
	for _, o := range filtered {
		w.Write([]string{o.Name, o.Created, o.Deadline, o.Status})
	}
	w.Flush()
	return w.Error()
}

// List prints a formatted table of all orders to stdout.
func List(cfg *config.Config) error {
	orders, err := Load(cfg)
	if err != nil {
		return err
	}
	if len(orders) == 0 {
		fmt.Println("No orders yet.")
		return nil
	}

	today, _ := time.Parse("02/01/2006", time.Now().Format("02/01/2006"))

	fmt.Println()
	fmt.Printf("  %s  %s  %s\n",
		col("PROJECT", 38),
		col("DEADLINE", 12),
		col("STATUS", 12),
	)
	fmt.Println("  " + strings.Repeat("─", 70))

	for _, o := range orders {
		deadline := o.Deadline
		if deadline == "" {
			deadline = "—"
		}

		warning := ""
		if deadline != "—" && o.Status != "done" {
			if d, err := time.Parse("02/01/2006", deadline); err == nil && d.Before(today) {
				warning = "⚠ overdue"
			}
		}

		fmt.Printf("  %s  %s  %s  %s\n",
			col(o.Name, 38),
			col(deadline, 12),
			col(o.Status, 12),
			warning,
		)
	}
	fmt.Println()
	return nil
}

// col pads or truncates s to exactly w characters.
func col(s string, w int) string {
	if len(s) >= w {
		return s[:w-1] + "…"
	}
	return s + strings.Repeat(" ", w-len(s))
}
