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
		// {"recipes", "example", "TEXT NOT NULL DEFAULT ''"},
	} {
		if err := addColumn(d, c.table, c.column, c.def); err != nil {
			return err
		}
	}
	return nil
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
