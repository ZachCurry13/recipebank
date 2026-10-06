package store

// ReadingHint is one thing the AI misread on the family's cards.
type ReadingHint struct {
	ID    int64  `db:"id" json:"id"`
	Wrong string `db:"wrong" json:"wrong"`
	Right string `db:"right_text" json:"right"`
	Times int    `db:"times" json:"times"`
}

// ReadingHints lists the hints, most often seen (then newest) first; n <= 0
// means all of them.
func (s *Store) ReadingHints(n int) ([]ReadingHint, error) {
	if n <= 0 {
		n = -1
	}
	out := []ReadingHint{}
	err := s.DB.Select(&out, `SELECT id, wrong, right_text, times FROM reading_hints
		ORDER BY times DESC, updated_at DESC, id DESC LIMIT ?`, n)
	return out, err
}

// LearnHint records a misread, or counts it once more.
func (s *Store) LearnHint(wrong, right string) error {
	_, err := s.DB.Exec(`INSERT INTO reading_hints (wrong, right_text) VALUES (?, ?)
		ON CONFLICT(wrong, right_text) DO UPDATE SET times = times + 1, updated_at = CURRENT_TIMESTAMP`, wrong, right)
	return err
}

func (s *Store) DeleteHint(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM reading_hints WHERE id = ?`, id)
	return err
}
