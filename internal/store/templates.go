package store

import (
	"fmt"

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
