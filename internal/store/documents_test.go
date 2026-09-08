package store

import (
	"errors"
	"testing"

	"proces-verbal-transare/internal/model"
)

func sampleDocument() model.Document {
	return model.Document{
		Nr:                         193,
		Data:                       "2026-09-03",
		Gestiune:                   "Magazin Bradet",
		DocumentReferinta:          "fact. 6721411/03.09.26",
		DiferentaTip:               "plus",
		DiferentaValoare:           2812.30,
		IncarcaDescarcaTip:         "incarca",
		IncarcaDescarcaValoare:     2812.30,
		Gestionar:                  "Ion Popescu",
		Calculator:                 "Maria Ionescu",
		VizatCompartimentProductie: "Sef productie",
		Intrare: []model.IntrareRow{
			{Pozitie: 0, Denumire: "Carcasa porc f cap", UM: "Kg", Cantitate: 162.20, PretFaraTVA: 12.50, PretCuTVA: 14.03},
		},
		Iesire: []model.IesireRow{
			{Pozitie: 0, Denumire: "Pulpa fara os", UM: "Kg", PretCuTVA: 21.9, Cantitate: 15, PretFaraTVA: 16.5},
			{Pozitie: 1, Denumire: "Muschiulet", UM: "Kg", PretCuTVA: 39.5, Cantitate: 1.5},
		},
	}
}

func TestSaveAndGetDocument(t *testing.T) {
	s := newTestStore(t)

	saved, err := s.SaveDocument(sampleDocument())
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	if saved.ID == 0 {
		t.Fatal("saved.ID = 0, want a generated id")
	}
	if saved.CreatedAt == "" || saved.UpdatedAt == "" {
		t.Errorf("timestamps not set: created=%q updated=%q", saved.CreatedAt, saved.UpdatedAt)
	}

	got, err := s.GetDocument(saved.ID)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if got.Nr != 193 || got.Data != "2026-09-03" || got.Gestiune != "Magazin Bradet" {
		t.Errorf("header round-trip failed: %+v", got)
	}
	if got.DocumentReferinta != "fact. 6721411/03.09.26" {
		t.Errorf("DocumentReferinta = %q", got.DocumentReferinta)
	}
	if got.DiferentaTip != "plus" || got.DiferentaValoare != 2812.30 {
		t.Errorf("diferenta round-trip failed: %q %v", got.DiferentaTip, got.DiferentaValoare)
	}
	if got.IncarcaDescarcaTip != "incarca" || got.IncarcaDescarcaValoare != 2812.30 {
		t.Errorf("incarca/descarca round-trip failed: %q %v", got.IncarcaDescarcaTip, got.IncarcaDescarcaValoare)
	}
	if got.Gestionar != "Ion Popescu" || got.Calculator != "Maria Ionescu" || got.VizatCompartimentProductie != "Sef productie" {
		t.Errorf("footer names round-trip failed: %+v", got)
	}
	if len(got.Intrare) != 1 || got.Intrare[0].Denumire != "Carcasa porc f cap" || got.Intrare[0].Cantitate != 162.20 {
		t.Errorf("intrare round-trip failed: %+v", got.Intrare)
	}
	if len(got.Iesire) != 2 {
		t.Fatalf("len(Iesire) = %d, want 2", len(got.Iesire))
	}
	if got.Iesire[0].Denumire != "Pulpa fara os" || got.Iesire[0].Cantitate != 15 || got.Iesire[0].PretCuTVA != 21.9 {
		t.Errorf("iesire[0] round-trip failed: %+v", got.Iesire[0])
	}
	if got.Iesire[1].Pozitie != 1 {
		t.Errorf("iesire[1].Pozitie = %d, want 1", got.Iesire[1].Pozitie)
	}
}

func TestSaveDocumentUpdatesInPlace(t *testing.T) {
	s := newEmptyTestStore(t)

	saved, err := s.SaveDocument(sampleDocument())
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	saved.Gestiune = "Magazin Central"
	saved.Iesire = saved.Iesire[:1]
	saved.Iesire[0].Cantitate = 20
	updated, err := s.SaveDocument(saved)
	if err != nil {
		t.Fatalf("SaveDocument (update): %v", err)
	}
	if updated.ID != saved.ID {
		t.Errorf("ID changed on update: %d -> %d", saved.ID, updated.ID)
	}

	got, err := s.GetDocument(saved.ID)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if got.Gestiune != "Magazin Central" {
		t.Errorf("Gestiune = %q, want Magazin Central", got.Gestiune)
	}
	if len(got.Iesire) != 1 || got.Iesire[0].Cantitate != 20 {
		t.Errorf("iesire after update = %+v, want a single row with cantitate 20", got.Iesire)
	}

	docs, err := s.ListDocuments()
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("len(docs) = %d after an update, want 1", len(docs))
	}
}

func TestListDocumentsNewestFirst(t *testing.T) {
	s := newEmptyTestStore(t)

	older := sampleDocument()
	older.Nr = 192
	older.Data = "2026-09-01"
	if _, err := s.SaveDocument(older); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	if _, err := s.SaveDocument(sampleDocument()); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	docs, err := s.ListDocuments()
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("len(docs) = %d, want 2", len(docs))
	}
	if docs[0].Nr != 193 || docs[1].Nr != 192 {
		t.Errorf("order = %d, %d; want 193, 192 (newest first)", docs[0].Nr, docs[1].Nr)
	}
	if docs[0].Gestiune != "Magazin Bradet" || docs[0].Data != "2026-09-03" {
		t.Errorf("summary = %+v", docs[0])
	}
}

func TestDeleteDocumentCascadesRows(t *testing.T) {
	s := newEmptyTestStore(t)

	saved, err := s.SaveDocument(sampleDocument())
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	if err := s.DeleteDocument(saved.ID); err != nil {
		t.Fatalf("DeleteDocument: %v", err)
	}

	if _, err := s.GetDocument(saved.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetDocument after delete: err = %v, want ErrNotFound", err)
	}

	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM document_iesire_rows`).Scan(&n); err != nil {
		t.Fatalf("count iesire rows: %v", err)
	}
	if n != 0 {
		t.Errorf("iesire rows left after delete = %d, want 0 (cascade must fire)", n)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM document_intrare_rows`).Scan(&n); err != nil {
		t.Fatalf("count intrare rows: %v", err)
	}
	if n != 0 {
		t.Errorf("intrare rows left after delete = %d, want 0", n)
	}
}

func TestGetDocumentMissing(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.GetDocument(999); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestLastDocument(t *testing.T) {
	s := newEmptyTestStore(t)

	if _, ok, err := s.LastDocument(); err != nil || ok {
		t.Errorf("LastDocument on empty store = ok %v, err %v; want false, nil", ok, err)
	}

	if _, err := s.SaveDocument(sampleDocument()); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	got, ok, err := s.LastDocument()
	if err != nil {
		t.Fatalf("LastDocument: %v", err)
	}
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if len(got.Intrare) != 1 || got.Intrare[0].Denumire != "Carcasa porc f cap" {
		t.Errorf("LastDocument intrare = %+v, want the saved row", got.Intrare)
	}
}

func TestIesireRowKeepsSnapshotAfterProductDeleted(t *testing.T) {
	s := newTestStore(t)

	templates, err := s.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	products := templates[0].Products
	pid := products[0].ID

	doc := sampleDocument()
	doc.Iesire[0].ProductID = &pid
	saved, err := s.SaveDocument(doc)
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	if _, err := s.db.Exec(`DELETE FROM products WHERE id = ?`, pid); err != nil {
		t.Fatalf("delete product: %v", err)
	}

	got, err := s.GetDocument(saved.ID)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if got.Iesire[0].Denumire != "Pulpa fara os" || got.Iesire[0].PretCuTVA != 21.9 {
		t.Errorf("snapshot lost after product delete: %+v", got.Iesire[0])
	}
	if got.Iesire[0].ProductID != nil {
		t.Errorf("ProductID = %v, want nil after the product was deleted", *got.Iesire[0].ProductID)
	}
}

func TestSaveDocumentRoundTripsRowTvaRates(t *testing.T) {
	s := newTestStore(t)

	d := sampleDocument()
	d.Intrare[0].CotaTVA = 11
	d.Iesire[0].CotaTVA = 11
	d.Iesire[1].CotaTVA = 21 // a row deliberately taxed at a different rate

	saved, err := s.SaveDocument(d)
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	got, err := s.GetDocument(saved.ID)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if got.Intrare[0].CotaTVA != 11 {
		t.Errorf("Intrare[0].CotaTVA = %v, want 11", got.Intrare[0].CotaTVA)
	}
	if got.Iesire[0].CotaTVA != 11 {
		t.Errorf("Iesire[0].CotaTVA = %v, want 11", got.Iesire[0].CotaTVA)
	}
	if got.Iesire[1].CotaTVA != 21 {
		t.Errorf("Iesire[1].CotaTVA = %v, want 21 (a per-row rate must survive)", got.Iesire[1].CotaTVA)
	}
}
