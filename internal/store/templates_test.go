package store

import (
	"errors"
	"testing"

	"proces-verbal-transare/internal/model"
)

// seededTemplate is the one template a fresh store starts with.
func seededTemplate(t *testing.T, s *Store) model.Template {
	t.Helper()
	templates, err := s.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(templates) == 0 {
		t.Fatal("len(templates) = 0, want the seeded templates")
	}
	// The pig template, which the seeded document belongs to. The seed ships a
	// beef template beside it; tests that care how many there are assert that
	// for themselves.
	return templates[0]
}

func TestGetTemplateReturnsItsProducts(t *testing.T) {
	s := newTestStore(t)
	seeded := seededTemplate(t, s)

	got, err := s.GetTemplate(seeded.ID)
	if err != nil {
		t.Fatalf("GetTemplate: %v", err)
	}
	if got.Nume != seeded.Nume {
		t.Errorf("nume = %q, want %q", got.Nume, seeded.Nume)
	}
	if len(got.Products) != len(seeded.Products) {
		t.Errorf("len(products) = %d, want %d", len(got.Products), len(seeded.Products))
	}
}

func TestGetTemplateOnAnUnknownID(t *testing.T) {
	s := newTestStore(t)

	_, err := s.GetTemplate(9999)
	if !errors.Is(err, ErrTemplateNotFound) {
		t.Errorf("err = %v, want ErrTemplateNotFound", err)
	}
}

func TestSaveTemplatesInsertsANewTemplateWithItsProducts(t *testing.T) {
	s := newTestStore(t)
	templates, err := s.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}

	templates = append(templates, model.Template{
		Nume: "Carcasa Vitel",
		Products: []model.Product{
			{Denumire: "Antricot", UM: "Kg", PretCuTVA: 45, ProcentDinIntrare: 40},
			{Denumire: "Deseu", UM: "Kg", PretCuTVA: 0, ProcentDinIntrare: 60},
		},
	})
	if err := s.SaveTemplates(templates); err != nil {
		t.Fatalf("SaveTemplates: %v", err)
	}

	stored, err := s.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(stored) != len(templates) {
		t.Fatalf("len(templates) = %d, want %d", len(stored), len(templates))
	}
	last := len(stored) - 1
	vitel := stored[last]
	if vitel.Nume != "Carcasa Vitel" {
		t.Errorf("nume = %q, want %q", vitel.Nume, "Carcasa Vitel")
	}
	if vitel.Ordine != last {
		t.Errorf("ordine = %d, want %d (its position in the saved slice)", vitel.Ordine, last)
	}
	if len(vitel.Products) != 2 {
		t.Fatalf("len(products) = %d, want 2", len(vitel.Products))
	}
	if vitel.Products[0].Denumire != "Antricot" {
		t.Errorf("products[0] = %q, want %q", vitel.Products[0].Denumire, "Antricot")
	}
	if vitel.Products[0].TemplateID != vitel.ID {
		t.Errorf("products[0].TemplateID = %d, want %d", vitel.Products[0].TemplateID, vitel.ID)
	}
}

func TestSaveTemplatesUpdatesRenamesAndReorders(t *testing.T) {
	s := newTestStore(t)
	templates, err := s.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	templates = append(templates, model.Template{
		Nume:     "Carcasa Vitel",
		Products: []model.Product{{Denumire: "Antricot", UM: "Kg", ProcentDinIntrare: 100}},
	})
	if err := s.SaveTemplates(templates); err != nil {
		t.Fatalf("SaveTemplates: %v", err)
	}

	stored, err := s.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	// Swap the first and last, and rename the first. Everything between keeps
	// its place, so this is a reorder rather than a rewrite of the whole list.
	stored[0].Nume = "Carcasa Porc mare"
	last := len(stored) - 1
	swapped := make([]model.Template, len(stored))
	copy(swapped, stored)
	swapped[0], swapped[last] = swapped[last], swapped[0]
	if err := s.SaveTemplates(swapped); err != nil {
		t.Fatalf("SaveTemplates: %v", err)
	}

	after, err := s.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(after) != len(stored) {
		t.Fatalf("len(after) = %d, want %d — a reorder must not add or drop a template",
			len(after), len(stored))
	}
	if after[0].Nume != "Carcasa Vitel" {
		t.Errorf("after[0] = %q, want %q", after[0].Nume, "Carcasa Vitel")
	}
	if after[last].Nume != "Carcasa Porc mare" {
		t.Errorf("after[%d] = %q, want %q", last, after[last].Nume, "Carcasa Porc mare")
	}
	if after[last].ID != stored[0].ID {
		t.Errorf("the renamed template got a new id (%d, was %d) — it was replaced, not updated",
			after[last].ID, stored[0].ID)
	}
}

func TestSaveTemplatesInsertsUpdatesDeletesAndReordersProducts(t *testing.T) {
	s := newTestStore(t)
	seeded := seededTemplate(t, s)

	// Keep the first two, swap them, edit one, drop the rest, add one.
	first, second := seeded.Products[0], seeded.Products[1]
	second.PretCuTVA = 99.5
	seeded.Products = []model.Product{
		second,
		first,
		{Denumire: "Adaugat", UM: "Kg", PretCuTVA: 1, ProcentDinIntrare: 0},
	}
	if err := s.SaveTemplates([]model.Template{seeded}); err != nil {
		t.Fatalf("SaveTemplates: %v", err)
	}

	after := seededTemplate(t, s)
	if len(after.Products) != 3 {
		t.Fatalf("len(products) = %d, want 3", len(after.Products))
	}
	if after.Products[0].ID != second.ID {
		t.Errorf("products[0].ID = %d, want %d — reorder replaced instead of updating",
			after.Products[0].ID, second.ID)
	}
	if after.Products[0].PretCuTVA != 99.5 {
		t.Errorf("products[0].PretCuTVA = %v, want 99.5", after.Products[0].PretCuTVA)
	}
	if after.Products[2].Denumire != "Adaugat" || after.Products[2].ID == 0 {
		t.Errorf("products[2] = %+v, want a stored row named Adaugat", after.Products[2])
	}
	for i, p := range after.Products {
		if p.Ordine != i {
			t.Errorf("products[%d].Ordine = %d, want %d", i, p.Ordine, i)
		}
	}
}

// Deleting a template takes its products with it and leaves its documents
// standing: the rows of a stored document carry their own denumire, UM and
// price, so the document is still complete and printable. What it loses is
// only its link back to a template, which is what the ratio write-back needs.
func TestSaveTemplatesDeleteRemovesProductsButKeepsDocuments(t *testing.T) {
	s := newTestStore(t)
	templates, err := s.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	porc := templates[0]

	templates = append(templates, model.Template{
		Nume:     "Carcasa Vitel",
		Products: []model.Product{{Denumire: "Antricot", UM: "Kg", ProcentDinIntrare: 100}},
	})
	if err := s.SaveTemplates(templates); err != nil {
		t.Fatalf("SaveTemplates: %v", err)
	}
	stored, err := s.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}

	// Drop the seeded pig template, which the seeded document points at.
	if err := s.SaveTemplates([]model.Template{stored[1]}); err != nil {
		t.Fatalf("SaveTemplates: %v", err)
	}

	var orphans int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM products WHERE template_id = ?`, porc.ID,
	).Scan(&orphans); err != nil {
		t.Fatalf("count products: %v", err)
	}
	if orphans != 0 {
		t.Errorf("%d products survived their template", orphans)
	}

	docs, err := s.ListDocuments()
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("len(docs) = %d, want the seeded document to survive", len(docs))
	}
	doc, err := s.GetDocument(docs[0].ID)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if doc.TemplateID != nil {
		t.Errorf("doc.TemplateID = %d, want nil after its template was deleted", *doc.TemplateID)
	}
	if len(doc.Iesire) == 0 {
		t.Error("the document lost its iesire rows")
	}
}

// TestSaveTemplatesRoundTripsProcentDinIntrare guards the SELECT/Scan column
// order for products: a fractional ProcentDinIntrare must come back exactly
// as written, through both the insert and the update path, so a swap with
// pret_cu_tva (also a REAL column) would show up here.
func TestSaveTemplatesRoundTripsProcentDinIntrare(t *testing.T) {
	s := newTestStore(t)

	// One inserted row and one updated row, so both SQL paths are covered.
	tmpl := model.Template{
		Nume: "Carcasa Vitel",
		Products: []model.Product{
			{Denumire: "Pulpa fara os", UM: "Kg", PretCuTVA: 21.9, ProcentDinIntrare: 60.5},
			{Denumire: "Slanina", UM: "Kg", PretCuTVA: 12, ProcentDinIntrare: 39.5},
		},
	}
	if err := s.SaveTemplates([]model.Template{tmpl}); err != nil {
		t.Fatalf("SaveTemplates: %v", err)
	}

	stored, err := s.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	inserted := stored[0]
	if len(inserted.Products) != 2 {
		t.Fatalf("len(products) = %d, want 2", len(inserted.Products))
	}
	if inserted.Products[0].ProcentDinIntrare != 60.5 || inserted.Products[1].ProcentDinIntrare != 39.5 {
		t.Fatalf("after insert = %v / %v, want 60.5 / 39.5",
			inserted.Products[0].ProcentDinIntrare, inserted.Products[1].ProcentDinIntrare)
	}

	inserted.Products[0].ProcentDinIntrare = 12.345
	if err := s.SaveTemplates([]model.Template{inserted}); err != nil {
		t.Fatalf("SaveTemplates (update): %v", err)
	}

	updated, err := s.GetTemplate(inserted.ID)
	if err != nil {
		t.Fatalf("GetTemplate: %v", err)
	}
	if updated.Products[0].ProcentDinIntrare != 12.345 {
		t.Errorf("after update = %v, want 12.345", updated.Products[0].ProcentDinIntrare)
	}
}

func TestSaveTemplatesRefusesAnEmptyList(t *testing.T) {
	s := newTestStore(t)

	if err := s.SaveTemplates([]model.Template{}); err == nil {
		t.Fatal("SaveTemplates([]) = nil, want an error — there must always be a template")
	}
	if len(seededTemplate(t, s).Products) == 0 {
		t.Error("the refused save still emptied the database")
	}
}

func TestSaveTemplatesRefusesABlankName(t *testing.T) {
	s := newTestStore(t)
	seeded := seededTemplate(t, s)
	seeded.Nume = "   "

	if err := s.SaveTemplates([]model.Template{seeded}); err == nil {
		t.Fatal("SaveTemplates with a blank name = nil, want an error")
	}
	if seededTemplate(t, s).Nume != "Carcasa Porc" {
		t.Error("the refused save renamed the template anyway")
	}
}
