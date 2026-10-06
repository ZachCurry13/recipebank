package db

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
)

// A database from before 0.5 keeps every recipe and every link to it when
// the recipes table is rebuilt to allow "ai" and "app" recipes.
func TestWidenSourceKinds(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "old.db") + "?_pragma=foreign_keys(1)"
	old, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	old.SetMaxOpenConns(1)
	if _, err := old.Exec(strings.Replace(schema, newKinds, oldKinds, 1)); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO recipes (id, title, source_kind, ingredients) VALUES (7, 'Soup', 'photo', '[{"line":"1 onion"}]')`,
		`INSERT INTO recipes (id, title, version_of) VALUES (9, 'Our soup', 7)`,
		`INSERT INTO cooks (recipe_id, cooked_on) VALUES (7, '2026-10-01')`,
		`INSERT INTO collections (id, name) VALUES (1, 'Winter')`,
		`INSERT INTO collection_recipes (collection_id, recipe_id) VALUES (1, 7)`,
	} {
		if _, err := old.Exec(q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	if _, err := old.Exec(`INSERT INTO recipes (title, source_kind) VALUES ('Guess', 'ai')`); err == nil {
		t.Fatal("the old table should refuse 'ai'")
	}
	old.Close()

	d, err := OpenDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	var create string
	d.Get(&create, `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'recipes'`)
	if !strings.Contains(create, newKinds) || !strings.Contains(create, "REFERENCES recipes(id)") {
		t.Fatalf("table: %s", create)
	}
	var n, cooks, cols int
	var ings string
	var versionOf *int64
	d.Get(&n, `SELECT COUNT(*) FROM recipes`)
	d.Get(&ings, `SELECT ingredients FROM recipes WHERE id = 7`)
	d.Get(&versionOf, `SELECT version_of FROM recipes WHERE id = 9`)
	d.Get(&cooks, `SELECT COUNT(*) FROM cooks WHERE recipe_id = 7`)
	d.Get(&cols, `SELECT COUNT(*) FROM collection_recipes WHERE recipe_id = 7`)
	if n != 2 || ings != `[{"line":"1 onion"}]` || versionOf == nil || *versionOf != 7 || cooks != 1 || cols != 1 {
		t.Fatalf("after: %d %q %v %d %d", n, ings, versionOf, cooks, cols)
	}
	var idx int
	d.Get(&idx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'recipes_area'`)
	if idx != 1 {
		t.Fatal("the index wasn't made again")
	}
	if _, err := d.Exec(`INSERT INTO recipes (title, source_kind) VALUES ('Guess', 'ai'), ('Imported', 'app')`); err != nil {
		t.Fatalf("new kinds: %v", err)
	}
	// Links still work: deleting a recipe takes its cooks and collection entries with it.
	if _, err := d.Exec(`DELETE FROM recipes WHERE id = 7`); err != nil {
		t.Fatal(err)
	}
	d.Get(&cooks, `SELECT COUNT(*) FROM cooks`)
	d.Get(&cols, `SELECT COUNT(*) FROM collection_recipes`)
	d.Get(&versionOf, `SELECT version_of FROM recipes WHERE id = 9`)
	if cooks != 0 || cols != 0 || versionOf != nil {
		t.Fatalf("links: %d %d %v", cooks, cols, versionOf)
	}
	var fk int
	d.Get(&fk, `PRAGMA foreign_keys`)
	if fk != 1 {
		t.Fatal("foreign keys are off after the upgrade")
	}
	// Opening again changes nothing.
	if err := migrate(d); err != nil {
		t.Fatal(err)
	}
}
