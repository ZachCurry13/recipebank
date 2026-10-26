package store

import (
	"encoding/binary"
	"math"
)

// KeyEmbedModel is the embedding model for "Search by meaning" ("" = off).
const KeyEmbedModel = "embed_model"

func packVec(v []float32) []byte {
	b := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(b[4*i:], math.Float32bits(f))
	}
	return b
}

func unpackVec(b []byte) []float32 {
	v := make([]float32, len(b)/4)
	for i := range v {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[4*i:]))
	}
	return v
}

// SaveEmbedding keeps a recipe's embedding and what it was made from.
func (s *Store) SaveEmbedding(recipeID int64, model, hash string, vec []float32) error {
	_, err := s.DB.Exec(`INSERT INTO embeddings (recipe_id, model, hash, vec) VALUES (?, ?, ?, ?)
		ON CONFLICT (recipe_id) DO UPDATE SET model = excluded.model, hash = excluded.hash, vec = excluded.vec`,
		recipeID, model, hash, packVec(vec))
	return err
}

// EmbeddedHashes maps each recipe embedded with model to the hash it was made from.
func (s *Store) EmbeddedHashes(model string) (map[int64]string, error) {
	var rows []struct {
		ID   int64  `db:"recipe_id"`
		Hash string `db:"hash"`
	}
	if err := s.DB.Select(&rows, `SELECT recipe_id, hash FROM embeddings WHERE model = ?`, model); err != nil {
		return nil, err
	}
	out := make(map[int64]string, len(rows))
	for _, r := range rows {
		out[r.ID] = r.Hash
	}
	return out, nil
}

// Embeddings are the vectors made with model, by recipe.
func (s *Store) Embeddings(model string) (map[int64][]float32, error) {
	var rows []struct {
		ID  int64  `db:"recipe_id"`
		Vec []byte `db:"vec"`
	}
	if err := s.DB.Select(&rows, `SELECT recipe_id, vec FROM embeddings WHERE model = ?`, model); err != nil {
		return nil, err
	}
	out := make(map[int64][]float32, len(rows))
	for _, r := range rows {
		out[r.ID] = unpackVec(r.Vec)
	}
	return out, nil
}

// DropOtherEmbeddings removes vectors made with any model but model (a new
// model's vectors can't be compared with the old ones).
func (s *Store) DropOtherEmbeddings(model string) error {
	_, err := s.DB.Exec(`DELETE FROM embeddings WHERE model != ?`, model)
	return err
}
