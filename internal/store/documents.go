package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"proces-verbal-transare/internal/model"
)

// ErrNotFound is returned when a document id does not exist.
var ErrNotFound = errors.New("documentul nu a fost gasit")

// ListDocuments returns every document, newest first.
func (s *Store) ListDocuments() ([]model.DocumentSummary, error) {
	rows, err := s.db.Query(
		`SELECT id, nr, data, gestiune FROM documents ORDER BY data DESC, id DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("citire documente: %w", err)
	}
	defer rows.Close()

	out := []model.DocumentSummary{}
	for rows.Next() {
		var d model.DocumentSummary
		if err := rows.Scan(&d.ID, &d.Nr, &d.Data, &d.Gestiune); err != nil {
			return nil, fmt.Errorf("citire document: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// GetDocument reads one document with both row tables.
func (s *Store) GetDocument(id int64) (model.Document, error) {
	var d model.Document
	var templateID sql.NullInt64
	err := s.db.QueryRow(
		`SELECT id, template_id, nr, data, gestiune, document_referinta, diferenta_tip, diferenta_valoare,
		        incarca_descarca_tip, incarca_descarca_valoare, gestionar, calculator,
		        vizat_compartiment_productie, created_at, updated_at
		 FROM documents WHERE id = ?`, id,
	).Scan(
		&d.ID, &templateID, &d.Nr, &d.Data, &d.Gestiune, &d.DocumentReferinta, &d.DiferentaTip, &d.DiferentaValoare,
		&d.IncarcaDescarcaTip, &d.IncarcaDescarcaValoare, &d.Gestionar, &d.Calculator,
		&d.VizatCompartimentProductie, &d.CreatedAt, &d.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Document{}, ErrNotFound
	}
	if err != nil {
		return model.Document{}, fmt.Errorf("citire document: %w", err)
	}
	if templateID.Valid {
		id := templateID.Int64
		d.TemplateID = &id
	}

	if d.Intrare, err = s.intrareRows(id); err != nil {
		return model.Document{}, err
	}
	if d.Iesire, err = s.iesireRows(id); err != nil {
		return model.Document{}, err
	}
	return d, nil
}

func (s *Store) intrareRows(documentID int64) ([]model.IntrareRow, error) {
	rows, err := s.db.Query(
		`SELECT id, pozitie, denumire, um, cantitate, pret_fara_tva, pret_cu_tva, cota_tva
		 FROM document_intrare_rows WHERE document_id = ? ORDER BY pozitie, id`, documentID,
	)
	if err != nil {
		return nil, fmt.Errorf("citire randuri intrare: %w", err)
	}
	defer rows.Close()

	out := []model.IntrareRow{}
	for rows.Next() {
		var r model.IntrareRow
		if err := rows.Scan(&r.ID, &r.Pozitie, &r.Denumire, &r.UM, &r.Cantitate, &r.PretFaraTVA, &r.PretCuTVA, &r.CotaTVA); err != nil {
			return nil, fmt.Errorf("citire rand intrare: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) iesireRows(documentID int64) ([]model.IesireRow, error) {
	rows, err := s.db.Query(
		`SELECT id, product_id, pozitie, denumire, um, pret_cu_tva, cantitate, pret_fara_tva, cota_tva
		 FROM document_iesire_rows WHERE document_id = ? ORDER BY pozitie, id`, documentID,
	)
	if err != nil {
		return nil, fmt.Errorf("citire randuri iesire: %w", err)
	}
	defer rows.Close()

	out := []model.IesireRow{}
	for rows.Next() {
		var (
			r         model.IesireRow
			productID sql.NullInt64
		)
		if err := rows.Scan(&r.ID, &productID, &r.Pozitie, &r.Denumire, &r.UM, &r.PretCuTVA, &r.Cantitate, &r.PretFaraTVA, &r.CotaTVA); err != nil {
			return nil, fmt.Errorf("citire rand iesire: %w", err)
		}
		if productID.Valid {
			id := productID.Int64
			r.ProductID = &id
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SaveDocument inserts d when its ID is 0 and updates it otherwise. Row tables
// are wholly owned by the document, so they are replaced rather than diffed.
// It returns the document as stored, re-read from the database.
func (s *Store) SaveDocument(d model.Document) (model.Document, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	tx, err := s.db.Begin()
	if err != nil {
		return model.Document{}, fmt.Errorf("salvare document: %w", err)
	}
	defer tx.Rollback()

	if d.ID == 0 {
		res, err := tx.Exec(
			`INSERT INTO documents (nr, data, gestiune, document_referinta, diferenta_tip, diferenta_valoare,
			         incarca_descarca_tip, incarca_descarca_valoare, gestionar, calculator,
			         vizat_compartiment_productie, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			d.Nr, d.Data, d.Gestiune, d.DocumentReferinta, d.DiferentaTip, d.DiferentaValoare,
			d.IncarcaDescarcaTip, d.IncarcaDescarcaValoare, d.Gestionar, d.Calculator,
			d.VizatCompartimentProductie, now, now,
		)
		if err != nil {
			return model.Document{}, fmt.Errorf("adaugare document: %w", err)
		}
		if d.ID, err = res.LastInsertId(); err != nil {
			return model.Document{}, fmt.Errorf("adaugare document: %w", err)
		}

		// Bump next_nr in the same transaction as the insert so the counter can
		// never fall out of sync with a committed document: either both the
		// document and the counter advance together, or (on any failure) the
		// whole transaction rolls back and next_nr is untouched.
		var nextNr int
		if err := tx.QueryRow(`SELECT next_nr FROM settings WHERE id = 1`).Scan(&nextNr); err != nil {
			return model.Document{}, fmt.Errorf("citire contor nr: %w", err)
		}
		if d.Nr >= nextNr {
			if _, err := tx.Exec(`UPDATE settings SET next_nr = ? WHERE id = 1`, d.Nr+1); err != nil {
				return model.Document{}, fmt.Errorf("actualizare contor nr: %w", err)
			}
		}
	} else {
		res, err := tx.Exec(
			`UPDATE documents SET nr = ?, data = ?, gestiune = ?, document_referinta = ?,
			        diferenta_tip = ?, diferenta_valoare = ?, incarca_descarca_tip = ?,
			        incarca_descarca_valoare = ?, gestionar = ?, calculator = ?,
			        vizat_compartiment_productie = ?, updated_at = ?
			 WHERE id = ?`,
			d.Nr, d.Data, d.Gestiune, d.DocumentReferinta, d.DiferentaTip, d.DiferentaValoare,
			d.IncarcaDescarcaTip, d.IncarcaDescarcaValoare, d.Gestionar, d.Calculator,
			d.VizatCompartimentProductie, now, d.ID,
		)
		if err != nil {
			return model.Document{}, fmt.Errorf("actualizare document: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return model.Document{}, fmt.Errorf("actualizare document: %w", err)
		}
		if n == 0 {
			return model.Document{}, ErrNotFound
		}
		if _, err := tx.Exec(`DELETE FROM document_intrare_rows WHERE document_id = ?`, d.ID); err != nil {
			return model.Document{}, fmt.Errorf("actualizare randuri intrare: %w", err)
		}
		if _, err := tx.Exec(`DELETE FROM document_iesire_rows WHERE document_id = ?`, d.ID); err != nil {
			return model.Document{}, fmt.Errorf("actualizare randuri iesire: %w", err)
		}
	}

	for i, r := range d.Intrare {
		if _, err := tx.Exec(
			`INSERT INTO document_intrare_rows (document_id, pozitie, denumire, um, cantitate, pret_fara_tva, pret_cu_tva, cota_tva)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			d.ID, i, r.Denumire, r.UM, r.Cantitate, r.PretFaraTVA, r.PretCuTVA, r.CotaTVA,
		); err != nil {
			return model.Document{}, fmt.Errorf("salvare rand intrare: %w", err)
		}
	}
	for i, r := range d.Iesire {
		var productID any
		if r.ProductID != nil {
			productID = *r.ProductID
		}
		if _, err := tx.Exec(
			`INSERT INTO document_iesire_rows (document_id, product_id, pozitie, denumire, um, pret_cu_tva, cantitate, pret_fara_tva, cota_tva)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			d.ID, productID, i, r.Denumire, r.UM, r.PretCuTVA, r.Cantitate, r.PretFaraTVA, r.CotaTVA,
		); err != nil {
			return model.Document{}, fmt.Errorf("salvare rand iesire: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return model.Document{}, fmt.Errorf("salvare document: %w", err)
	}
	return s.GetDocument(d.ID)
}

// DeleteDocument removes a document; its rows go with it via ON DELETE CASCADE.
func (s *Store) DeleteDocument(id int64) error {
	res, err := s.db.Exec(`DELETE FROM documents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("stergere document: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("stergere document: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// LastDocument returns the most recently saved document, if any. It is the
// source of the prefilled "ce intra" rows on a new document.
func (s *Store) LastDocument() (model.Document, bool, error) {
	var id int64
	err := s.db.QueryRow(`SELECT id FROM documents ORDER BY id DESC LIMIT 1`).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Document{}, false, nil
	}
	if err != nil {
		return model.Document{}, false, fmt.Errorf("citire ultimul document: %w", err)
	}
	d, err := s.GetDocument(id)
	if err != nil {
		return model.Document{}, false, err
	}
	return d, true, nil
}
