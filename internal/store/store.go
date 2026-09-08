// Package store is the SQLite persistence layer.
package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"

	"proces-verbal-transare/internal/model"
)

// Store owns the database handle.
type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the SQLite file at path, applies the schema
// and seeds first-run data.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("deschidere bazei de date: %w", err)
	}
	// SQLite tolerates a single writer; one connection keeps writes serialized
	// and keeps an in-memory DB from splitting across connections.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("conectare la baza de date: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrare baza de date: %w", err)
	}
	return &Store{db: db}, nil
}

// Close releases the database handle.
func (s *Store) Close() error {
	return s.db.Close()
}

// GetSettings reads the single settings row.
func (s *Store) GetSettings() (model.Settings, error) {
	var out model.Settings
	err := s.db.QueryRow(
		`SELECT unitate_nume, next_nr, cota_tva FROM settings WHERE id = 1`,
	).Scan(&out.UnitateNume, &out.NextNr, &out.CotaTVA)
	if err != nil {
		return model.Settings{}, fmt.Errorf("citire setari: %w", err)
	}
	return out, nil
}

// SaveSettings overwrites the settings row.
func (s *Store) SaveSettings(in model.Settings) error {
	_, err := s.db.Exec(
		`INSERT INTO settings (id, unitate_nume, next_nr, cota_tva) VALUES (1, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET unitate_nume = excluded.unitate_nume,
		        next_nr = excluded.next_nr, cota_tva = excluded.cota_tva`,
		in.UnitateNume, in.NextNr, in.CotaTVA,
	)
	if err != nil {
		return fmt.Errorf("salvare setari: %w", err)
	}
	return nil
}

// ListProducts returns every product ordered by ordine.
func (s *Store) ListProducts() ([]model.Product, error) {
	rows, err := s.db.Query(
		`SELECT id, denumire, um, pret_cu_tva, procent_din_intrare, ordine
		 FROM products ORDER BY ordine, id`,
	)
	if err != nil {
		return nil, fmt.Errorf("citire produse: %w", err)
	}
	defer rows.Close()

	out := []model.Product{}
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.Denumire, &p.UM, &p.PretCuTVA, &p.ProcentDinIntrare, &p.Ordine); err != nil {
			return nil, fmt.Errorf("citire produs: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SaveProducts makes the stored product list match in: rows with ID 0 are
// inserted, known IDs are updated, absent IDs are deleted, and ordine is
// reassigned from the slice order.
func (s *Store) SaveProducts(in []model.Product) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("salvare produse: %w", err)
	}
	defer tx.Rollback()

	keep := make(map[int64]bool, len(in))
	for i, p := range in {
		if p.ID == 0 {
			res, err := tx.Exec(
				`INSERT INTO products (denumire, um, pret_cu_tva, procent_din_intrare, ordine)
				 VALUES (?, ?, ?, ?, ?)`,
				p.Denumire, p.UM, p.PretCuTVA, p.ProcentDinIntrare, i,
			)
			if err != nil {
				return fmt.Errorf("adaugare produs %q: %w", p.Denumire, err)
			}
			id, err := res.LastInsertId()
			if err != nil {
				return fmt.Errorf("adaugare produs %q: %w", p.Denumire, err)
			}
			keep[id] = true
			continue
		}
		if _, err := tx.Exec(
			`UPDATE products SET denumire = ?, um = ?, pret_cu_tva = ?,
			        procent_din_intrare = ?, ordine = ? WHERE id = ?`,
			p.Denumire, p.UM, p.PretCuTVA, p.ProcentDinIntrare, i, p.ID,
		); err != nil {
			return fmt.Errorf("actualizare produs %q: %w", p.Denumire, err)
		}
		keep[p.ID] = true
	}

	existing, err := tx.Query(`SELECT id FROM products`)
	if err != nil {
		return fmt.Errorf("salvare produse: %w", err)
	}
	var stale []int64
	for existing.Next() {
		var id int64
		if err := existing.Scan(&id); err != nil {
			existing.Close()
			return fmt.Errorf("salvare produse: %w", err)
		}
		if !keep[id] {
			stale = append(stale, id)
		}
	}
	existing.Close()
	if err := existing.Err(); err != nil {
		return fmt.Errorf("salvare produse: %w", err)
	}

	for _, id := range stale {
		if _, err := tx.Exec(`DELETE FROM products WHERE id = ?`, id); err != nil {
			return fmt.Errorf("stergere produs: %w", err)
		}
	}
	return tx.Commit()
}
