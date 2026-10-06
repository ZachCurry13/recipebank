package db

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jmoiron/sqlx"
)

// Databases made before 0.5 only allow four kinds of recipe source, so a
// recipe drafted from a photo of a dish ("ai") or imported from another app
// ("app") couldn't be saved. SQLite can't change a CHECK in place, so the
// recipes table is rebuilt once, the way SQLite's documentation describes:
// foreign keys paused, every row copied with its id, indexes made again,
// and the links from other tables checked before it's committed.
const (
	oldKinds = `source_kind IN ('web', 'photo', 'text', 'manual')`
	newKinds = `source_kind IN ('web', 'photo', 'text', 'manual', 'ai', 'app')`
)

var createRecipes = regexp.MustCompile(`^CREATE TABLE\s+"?recipes"?\s*\(`)

func widenSourceKinds(d *sqlx.DB) error {
	var create string
	if err := d.Get(&create, `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'recipes'`); err != nil {
		return err
	}
	if !strings.Contains(create, oldKinds) {
		return nil
	}
	if !createRecipes.MatchString(create) {
		return fmt.Errorf("upgrade recipes: unexpected table definition")
	}
	newCreate := createRecipes.ReplaceAllString(strings.Replace(create, oldKinds, newKinds, 1), "CREATE TABLE recipes_new (")
	var indexes []string
	if err := d.Select(&indexes, `SELECT sql FROM sqlite_master WHERE type = 'index' AND tbl_name = 'recipes' AND sql IS NOT NULL`); err != nil {
		return err
	}
	var cols []string
	if err := d.Select(&cols, `SELECT name FROM pragma_table_info('recipes')`); err != nil {
		return err
	}
	list := `"` + strings.Join(cols, `", "`) + `"`

	ctx := context.Background()
	conn, err := d.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		return err
	}
	defer conn.ExecContext(ctx, `PRAGMA foreign_keys = ON`)
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	steps := []string{newCreate,
		`INSERT INTO recipes_new (` + list + `) SELECT ` + list + ` FROM recipes`,
		`DROP TABLE recipes`,
		`ALTER TABLE recipes_new RENAME TO recipes`}
	for _, q := range append(steps, indexes...) {
		if _, err := tx.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("upgrade recipes: %w", err)
		}
	}
	rows, err := tx.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return err
	}
	broken := rows.Next()
	rows.Close()
	if broken {
		return fmt.Errorf("upgrade recipes: links between tables don't match; nothing was changed")
	}
	return tx.Commit()
}
