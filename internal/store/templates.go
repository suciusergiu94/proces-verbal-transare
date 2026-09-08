package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"proces-verbal-transare/internal/model"
)

// ListTemplates returns every template in display order, each carrying its own
// "ce iese" products in theirs.
func (s *Store) ListTemplates() ([]model.Template, error) {
	rows, err := s.db.Query(`SELECT id, nume, ordine FROM templates ORDER BY ordine, id`)
	if err != nil {
		return nil, fmt.Errorf("citire sabloane: %w", err)
	}
	defer rows.Close()

	out := []model.Template{}
	for rows.Next() {
		var t model.Template
		if err := rows.Scan(&t.ID, &t.Nume, &t.Ordine); err != nil {
			return nil, fmt.Errorf("citire sablon: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("citire sabloane: %w", err)
	}

	for i := range out {
		if out[i].Products, err = s.listProducts(out[i].ID); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// listProducts returns one template's products in display order.
func (s *Store) listProducts(templateID int64) ([]model.Product, error) {
	rows, err := s.db.Query(
		`SELECT id, template_id, denumire, um, pret_cu_tva, procent_din_intrare, ordine
		 FROM products WHERE template_id = ? ORDER BY ordine, id`, templateID,
	)
	if err != nil {
		return nil, fmt.Errorf("citire produse: %w", err)
	}
	defer rows.Close()

	out := []model.Product{}
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.TemplateID, &p.Denumire, &p.UM,
			&p.PretCuTVA, &p.ProcentDinIntrare, &p.Ordine); err != nil {
			return nil, fmt.Errorf("citire produs: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ErrTemplateNotFound is returned when a template id does not exist.
var ErrTemplateNotFound = errors.New("sablonul nu a fost gasit")

// GetTemplate reads one template with its products.
func (s *Store) GetTemplate(id int64) (model.Template, error) {
	var t model.Template
	err := s.db.QueryRow(`SELECT id, nume, ordine FROM templates WHERE id = ?`, id).
		Scan(&t.ID, &t.Nume, &t.Ordine)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Template{}, ErrTemplateNotFound
	}
	if err != nil {
		return model.Template{}, fmt.Errorf("citire sablon: %w", err)
	}
	if t.Products, err = s.listProducts(t.ID); err != nil {
		return model.Template{}, err
	}
	return t, nil
}

// SaveTemplates makes the stored templates match in, products included: rows
// with ID 0 are inserted, known IDs are updated, absent IDs are deleted, and
// ordine is reassigned from the slice order at both levels.
//
// Deleting a template takes its products with it through the foreign key, and
// leaves its documents standing with a null template_id: a stored document
// carries its own denumire, UM and price on every row, so it remains complete
// and printable without the template it came from.
func (s *Store) SaveTemplates(in []model.Template) error {
	// There is nowhere to create a document from without at least one template,
	// and no screen from which to make one — Setari edits templates, it does
	// not bootstrap them. The frontend hides the delete button on the last
	// remaining template; this is the backstop behind it.
	if len(in) == 0 {
		return errors.New("trebuie sa existe cel putin un sablon")
	}
	for _, t := range in {
		if strings.TrimSpace(t.Nume) == "" {
			return errors.New("fiecare sablon trebuie sa aiba un nume")
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("salvare sabloane: %w", err)
	}
	defer tx.Rollback()

	keep := make(map[int64]bool, len(in))
	for i, t := range in {
		id := t.ID
		if id == 0 {
			res, err := tx.Exec(`INSERT INTO templates (nume, ordine) VALUES (?, ?)`, t.Nume, i)
			if err != nil {
				return fmt.Errorf("adaugare sablon %q: %w", t.Nume, err)
			}
			if id, err = res.LastInsertId(); err != nil {
				return fmt.Errorf("adaugare sablon %q: %w", t.Nume, err)
			}
		} else if _, err := tx.Exec(
			`UPDATE templates SET nume = ?, ordine = ? WHERE id = ?`, t.Nume, i, id,
		); err != nil {
			return fmt.Errorf("actualizare sablon %q: %w", t.Nume, err)
		}
		keep[id] = true

		if err := saveTemplateProducts(tx, id, t.Products); err != nil {
			return err
		}
	}

	stale, err := staleIDs(tx, `SELECT id FROM templates`, keep)
	if err != nil {
		return fmt.Errorf("salvare sabloane: %w", err)
	}
	for _, id := range stale {
		if _, err := tx.Exec(`DELETE FROM templates WHERE id = ?`, id); err != nil {
			return fmt.Errorf("stergere sablon: %w", err)
		}
	}
	return tx.Commit()
}

// saveTemplateProducts diffs one template's product list the way SaveTemplates
// diffs the templates themselves. template_id is taken from the owning
// template rather than from the incoming row, so a product cannot be moved
// into a template it was not sent under.
func saveTemplateProducts(tx *sql.Tx, templateID int64, products []model.Product) error {
	keep := make(map[int64]bool, len(products))
	for i, p := range products {
		if p.ID == 0 {
			res, err := tx.Exec(
				`INSERT INTO products (template_id, denumire, um, pret_cu_tva, procent_din_intrare, ordine)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				templateID, p.Denumire, p.UM, p.PretCuTVA, p.ProcentDinIntrare, i,
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
			`UPDATE products SET template_id = ?, denumire = ?, um = ?, pret_cu_tva = ?,
			        procent_din_intrare = ?, ordine = ? WHERE id = ?`,
			templateID, p.Denumire, p.UM, p.PretCuTVA, p.ProcentDinIntrare, i, p.ID,
		); err != nil {
			return fmt.Errorf("actualizare produs %q: %w", p.Denumire, err)
		}
		keep[p.ID] = true
	}

	stale, err := staleIDs(tx, `SELECT id FROM products WHERE template_id = `+fmt.Sprint(templateID), keep)
	if err != nil {
		return fmt.Errorf("salvare produse: %w", err)
	}
	for _, id := range stale {
		if _, err := tx.Exec(`DELETE FROM products WHERE id = ?`, id); err != nil {
			return fmt.Errorf("stergere produs: %w", err)
		}
	}
	return nil
}

// staleIDs runs a query returning one id column and reports those absent from
// keep — the rows a save has dropped.
func staleIDs(tx *sql.Tx, query string, keep map[int64]bool) ([]int64, error) {
	rows, err := tx.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stale []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if !keep[id] {
			stale = append(stale, id)
		}
	}
	return stale, rows.Err()
}
