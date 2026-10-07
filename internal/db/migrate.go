package db

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

// migrate brings databases made by older versions up to date. schema.sql
// creates missing tables; new columns on existing tables are added here with
// addColumn, one line per column, oldest first.
func migrate(d *sqlx.DB) error {
	for _, c := range []struct{ table, column, def string }{
		{"recipes", "read_by", "TEXT NOT NULL DEFAULT ''"},
		{"recipes", "ai_reading", "TEXT NOT NULL DEFAULT ''"},
		{"recipes", "difficulty", "TEXT NOT NULL DEFAULT ''"},
		{"meal_plan", "leftovers_of", "INTEGER REFERENCES meal_plan(id) ON DELETE CASCADE"},
		{"stock", "price", "REAL NOT NULL DEFAULT 0"},
		{"stock", "size", "TEXT NOT NULL DEFAULT ''"},
		{"users", "plan_meals", "TEXT NOT NULL DEFAULT ''"},   // meals the person's plan shows; '' = all
		{"users", "hidden_pages", "TEXT NOT NULL DEFAULT ''"}, // pages left out of the person's menu
		{"person_rules", "allow", "TEXT NOT NULL DEFAULT ''"}, // allergy: what they can have anyway, comma-separated
	} {
		if err := addColumn(d, c.table, c.column, c.def); err != nil {
			return err
		}
	}
	// The last version each person saw "What's new" for. People who used the
	// app before 0.3 get 0.3's notes instead of the first-time welcome.
	var had int
	if err := d.Get(&had, `SELECT COUNT(*) FROM pragma_table_info('users') WHERE name = 'seen_version'`); err != nil {
		return err
	}
	if had == 0 {
		if err := addColumn(d, "users", "seen_version", "TEXT NOT NULL DEFAULT ''"); err != nil {
			return err
		}
		if _, err := d.Exec(`UPDATE users SET seen_version = '0.2.0'`); err != nil {
			return err
		}
	}
	return widenSourceKinds(d)
}

// addColumn adds a column unless the table already has it.
func addColumn(d *sqlx.DB, table, column, def string) error {
	var n int
	if err := d.Get(&n, `SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, column); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	if _, err := d.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, column, def)); err != nil {
		return fmt.Errorf("add %s.%s: %w", table, column, err)
	}
	return nil
}
