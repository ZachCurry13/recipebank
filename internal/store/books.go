package store

import (
	"database/sql"
	"errors"
	"strings"
)

// Book is a cookbook on the family's shelf.
type Book struct {
	ID      int64  `db:"id" json:"id"`
	Title   string `db:"title" json:"title"`
	Author  string `db:"author" json:"author"`
	ISBN    string `db:"isbn" json:"isbn"`
	Cover   string `db:"cover" json:"cover"`     // a photo name, or ""
	Entries int    `db:"entries" json:"entries"` // recipes listed from it
	Saved   int    `db:"saved" json:"saved"`     // of them, saved in RecipeBank too
}

// BookEntry is one recipe in a cookbook, by page.
type BookEntry struct {
	ID          int64  `db:"id" json:"id"`
	BookID      int64  `db:"book_id" json:"book_id"`
	Title       string `db:"title" json:"title"`
	Page        string `db:"page" json:"page"`
	RecipeID    *int64 `db:"recipe_id" json:"recipe_id"`
	RecipeTitle string `db:"recipe_title" json:"recipe_title,omitempty"`
	BookTitle   string `db:"book_title" json:"book_title,omitempty"`
}

const bookSelect = `SELECT b.id, b.title, b.author, b.isbn, b.cover,
	(SELECT COUNT(*) FROM book_entries e WHERE e.book_id = b.id) AS entries,
	(SELECT COUNT(*) FROM book_entries e WHERE e.book_id = b.id AND e.recipe_id IS NOT NULL) AS saved
	FROM books b`

const entrySelect = `SELECT e.id, e.book_id, e.title, e.page, e.recipe_id, COALESCE(r.title, '') AS recipe_title,
	b.title AS book_title FROM book_entries e JOIN books b ON b.id = e.book_id LEFT JOIN recipes r ON r.id = e.recipe_id`

// ListBooks is the shelf, by title.
func (s *Store) ListBooks() ([]Book, error) {
	out := []Book{}
	err := s.DB.Select(&out, bookSelect+` ORDER BY b.title COLLATE NOCASE`)
	return out, err
}

// GetBook is one cookbook.
func (s *Store) GetBook(id int64) (*Book, error) {
	var b Book
	err := s.DB.Get(&b, bookSelect+` WHERE b.id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &b, err
}

// BookByISBN finds a cookbook already on the shelf.
func (s *Store) BookByISBN(isbn string) (*Book, error) {
	var id int64
	if err := s.DB.Get(&id, `SELECT id FROM books WHERE isbn = ? AND isbn != ''`, isbn); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.GetBook(id)
}

// SaveBook adds (ID 0) or changes a cookbook.
func (s *Store) SaveBook(b *Book) (int64, error) {
	if b.ID == 0 {
		res, err := s.DB.Exec(`INSERT INTO books (title, author, isbn, cover) VALUES (?, ?, ?, ?)`, b.Title, b.Author, b.ISBN, b.Cover)
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
	res, err := s.DB.Exec(`UPDATE books SET title = ?, author = ?, isbn = ?, cover = ? WHERE id = ?`, b.Title, b.Author, b.ISBN, b.Cover, b.ID)
	if err != nil {
		return 0, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return 0, ErrNotFound
	}
	return b.ID, nil
}

// DeleteBook removes a cookbook and its list of recipes (saved recipes stay).
func (s *Store) DeleteBook(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM books WHERE id = ?`, id)
	return err
}

// BookEntries is a cookbook's recipes, by page.
func (s *Store) BookEntries(bookID int64) ([]BookEntry, error) {
	out := []BookEntry{}
	err := s.DB.Select(&out, entrySelect+` WHERE e.book_id = ? ORDER BY CAST(e.page AS INTEGER), e.page, e.title COLLATE NOCASE`, bookID)
	return out, err
}

// AddBookEntries adds recipes to a cookbook's list, skipping ones it has.
func (s *Store) AddBookEntries(bookID int64, entries []BookEntry) (int, error) {
	have, err := s.BookEntries(bookID)
	if err != nil {
		return 0, err
	}
	seen := map[string]bool{}
	for _, e := range have {
		seen[strings.ToLower(e.Title)+"|"+e.Page] = true
	}
	tx, err := s.DB.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	n := 0
	for _, e := range entries {
		key := strings.ToLower(e.Title) + "|" + e.Page
		if seen[key] {
			continue
		}
		seen[key] = true
		if _, err := tx.Exec(`INSERT INTO book_entries (book_id, title, page) VALUES (?, ?, ?)`, bookID, e.Title, e.Page); err != nil {
			return 0, err
		}
		n++
	}
	return n, tx.Commit()
}

// DeleteBookEntry removes one recipe from a cookbook's list.
func (s *Store) DeleteBookEntry(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM book_entries WHERE id = ?`, id)
	return err
}

// EntryForRecipe is where a saved recipe came from, if from a cookbook.
func (s *Store) EntryForRecipe(recipeID int64) (*BookEntry, error) {
	var out []BookEntry
	if err := s.DB.Select(&out, entrySelect+` WHERE e.recipe_id = ? ORDER BY e.id LIMIT 1`, recipeID); err != nil || len(out) == 0 {
		return nil, err
	}
	return &out[0], nil
}

// LinkRecipe says a saved recipe is the one on a cookbook's page: the entry
// with that page (and title, when the page has several) gets the recipe, or a
// new entry is made. bookID 0 just unlinks it.
func (s *Store) LinkRecipe(recipeID, bookID int64, page, title string) error {
	var entries []BookEntry
	if bookID > 0 {
		var err error
		if entries, err = s.BookEntries(bookID); err != nil {
			return err
		}
	}
	tx, err := s.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE book_entries SET recipe_id = NULL WHERE recipe_id = ?`, recipeID); err != nil {
		return err
	}
	if bookID > 0 {
		var onPage []BookEntry
		for _, e := range entries {
			if e.Page == page {
				onPage = append(onPage, e)
			}
		}
		id := int64(0)
		for _, e := range onPage {
			if strings.EqualFold(e.Title, title) || len(onPage) == 1 && e.RecipeID == nil {
				id = e.ID
				break
			}
		}
		if id > 0 {
			_, err = tx.Exec(`UPDATE book_entries SET recipe_id = ? WHERE id = ?`, recipeID, id)
		} else {
			_, err = tx.Exec(`INSERT INTO book_entries (book_id, title, page, recipe_id) VALUES (?, ?, ?, ?)`, bookID, title, page, recipeID)
		}
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SearchBookEntries finds cookbook recipes whose name has every word asked for.
func (s *Store) SearchBookEntries(q string, limit int) ([]BookEntry, error) {
	where, args := []string{}, []any{}
	for _, w := range strings.Fields(strings.ToLower(q)) {
		where = append(where, `LOWER(e.title) LIKE ? ESCAPE '\'`)
		args = append(args, "%"+strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(w)+"%")
	}
	out := []BookEntry{}
	if len(where) == 0 {
		return out, nil
	}
	args = append(args, limit)
	err := s.DB.Select(&out, entrySelect+` WHERE `+strings.Join(where, " AND ")+` ORDER BY e.title COLLATE NOCASE LIMIT ?`, args...)
	return out, err
}
