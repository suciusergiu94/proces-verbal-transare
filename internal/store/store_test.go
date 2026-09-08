package store

import (
	"path/filepath"
	"testing"
	"time"

	"proces-verbal-transare/internal/calc"
	"proces-verbal-transare/internal/model"
)

// newTestStore opens a throwaway on-disk DB in the test's temp dir.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// newEmptyTestStore opens a throwaway store with the shipped proces verbal
// removed. Tests about saving, listing and deleting documents state their own
// starting point this way rather than counting around the seeded one.
func newEmptyTestStore(t *testing.T) *Store {
	t.Helper()
	s := newTestStore(t)
	docs, err := s.ListDocuments()
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	for _, d := range docs {
		if err := s.DeleteDocument(d.ID); err != nil {
			t.Fatalf("DeleteDocument: %v", err)
		}
	}
	return s
}

func TestOpenSeedsProducts(t *testing.T) {
	s := newTestStore(t)

	products, err := s.ListProducts()
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if len(products) != 19 {
		t.Fatalf("len(products) = %d, want 19", len(products))
	}
	if products[0].Denumire != "Pulpa fara os" || products[0].PretCuTVA != 21.9 {
		t.Errorf("products[0] = %+v, want Pulpa fara os / 21.9", products[0])
	}
	if products[3].Denumire != "Cotlet fara os" || products[3].PretCuTVA != 29.5 {
		t.Errorf("products[3] = %+v, want Cotlet fara os / 29.5", products[3])
	}
	if products[18].Denumire != "Deseu fara valoare" || products[18].PretCuTVA != 0 {
		t.Errorf("products[18] = %+v, want Deseu fara valoare / 0", products[18])
	}
	for i, p := range products {
		if p.UM != "Kg" {
			t.Errorf("products[%d].UM = %q, want Kg", i, p.UM)
		}
		if p.Ordine != i {
			t.Errorf("products[%d].Ordine = %d, want %d", i, p.Ordine, i)
		}
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	s1, err := Open(path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	if err := s1.SaveProducts(nil); err != nil {
		t.Fatalf("SaveProducts: %v", err)
	}
	s1.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer s2.Close()

	products, err := s2.ListProducts()
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if len(products) != 0 {
		t.Errorf("len(products) = %d after reopening an emptied DB, want 0 (re-seeding must not happen)", len(products))
	}
}

func TestOpenSetsUserVersion(t *testing.T) {
	s := newTestStore(t)

	var version int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatalf("PRAGMA user_version: %v", err)
	}
	if version != 2 {
		t.Errorf("user_version = %d, want 2 (bumped when the TVA rate columns were added)", version)
	}
}

func TestSettingsDefaultsAndRoundTrip(t *testing.T) {
	s := newTestStore(t)

	got, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got.UnitateNume != "S.C. Largiana Carn S.R.L." {
		t.Errorf("UnitateNume = %q, want the seeded company name", got.UnitateNume)
	}
	// 1 is taken by the seeded proces verbal, so a new document starts at 2.
	if got.NextNr != 2 {
		t.Errorf("NextNr = %d, want 2", got.NextNr)
	}

	if err := s.SaveSettings(model.Settings{UnitateNume: "Alt SRL", NextNr: 194}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	got, err = s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings after save: %v", err)
	}
	if got.UnitateNume != "Alt SRL" || got.NextNr != 194 {
		t.Errorf("settings = %+v, want {Alt SRL 194}", got)
	}
}

func TestSaveProductsInsertsUpdatesDeletesAndReorders(t *testing.T) {
	s := newTestStore(t)

	products, err := s.ListProducts()
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}

	// Keep the first two, swap their order, rename the first, and append a new one.
	products[0].Denumire = "Pulpa fara os (redenumit)"
	next := []model.Product{
		products[1],
		products[0],
		{Denumire: "Produs nou", UM: "buc", PretCuTVA: 5.5},
	}
	if err := s.SaveProducts(next); err != nil {
		t.Fatalf("SaveProducts: %v", err)
	}

	got, err := s.ListProducts()
	if err != nil {
		t.Fatalf("ListProducts after save: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len(products) = %d, want 3", len(got))
	}
	if got[0].Denumire != "Muschiulet" || got[0].Ordine != 0 {
		t.Errorf("got[0] = %+v, want Muschiulet at ordine 0", got[0])
	}
	if got[1].Denumire != "Pulpa fara os (redenumit)" || got[1].Ordine != 1 {
		t.Errorf("got[1] = %+v, want the renamed product at ordine 1", got[1])
	}
	if got[2].Denumire != "Produs nou" || got[2].UM != "buc" || got[2].PretCuTVA != 5.5 {
		t.Errorf("got[2] = %+v, want the newly inserted product", got[2])
	}
	if got[2].ID == 0 {
		t.Error("newly inserted product has ID 0, want a generated id")
	}
}

func TestSettingsSeedTvaRate(t *testing.T) {
	s := newTestStore(t)

	got, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got.CotaTVA != 11 {
		t.Errorf("CotaTVA = %v, want 11 (standard Romanian food rate)", got.CotaTVA)
	}
}

func TestSaveSettingsRoundTripsTvaRate(t *testing.T) {
	s := newTestStore(t)

	if err := s.SaveSettings(model.Settings{UnitateNume: "X", NextNr: 4, CotaTVA: 21}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	got, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got.CotaTVA != 21 {
		t.Errorf("CotaTVA = %v, want 21", got.CotaTVA)
	}
}

// TestOpenSeedsFirstDocument covers the proces verbal that ships with the app:
// a new install opens on a real, complete example rather than an empty list.
func TestOpenSeedsFirstDocument(t *testing.T) {
	s := newTestStore(t)

	docs, err := s.ListDocuments()
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("len(docs) = %d, want 1 seeded document", len(docs))
	}
	if docs[0].Nr != 1 {
		t.Errorf("docs[0].Nr = %d, want 1", docs[0].Nr)
	}

	doc, err := s.GetDocument(docs[0].ID)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if len(doc.Intrare) != 1 {
		t.Fatalf("len(Intrare) = %d, want 1", len(doc.Intrare))
	}
	if doc.Intrare[0].Denumire != "Carcasa" || doc.Intrare[0].Cantitate != 162.2 {
		t.Errorf("Intrare[0] = %+v, want Carcasa / 162.2", doc.Intrare[0])
	}
	if len(doc.Iesire) != 19 {
		t.Fatalf("len(Iesire) = %d, want one row per seeded product", len(doc.Iesire))
	}
	if doc.Iesire[0].Denumire != "Pulpa fara os" || doc.Iesire[0].Cantitate != 15 {
		t.Errorf("Iesire[0] = %+v, want Pulpa fara os / 15", doc.Iesire[0])
	}
	if doc.Iesire[18].Denumire != "Deseu fara valoare" || doc.Iesire[18].Cantitate != 1.2 {
		t.Errorf("Iesire[18] = %+v, want Deseu fara valoare / 1.2", doc.Iesire[18])
	}
}

// The seeded document is dated the day it was seeded, not the day the sheet it
// was copied from was written: a fresh install should not open on a document
// that looks months stale.
func TestOpenSeedsFirstDocumentDatedToday(t *testing.T) {
	s := newTestStore(t)

	doc := firstSeededDocument(t, s)
	if want := time.Now().Format("2006-01-02"); doc.Data != want {
		t.Errorf("Data = %q, want %q (the install date)", doc.Data, want)
	}
}

// Every "ce iese" row must point at the product it was seeded from, so editing
// a product still reaches the document that uses it.
func TestOpenSeedsFirstDocumentLinkedToProducts(t *testing.T) {
	s := newTestStore(t)

	products, err := s.ListProducts()
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	doc := firstSeededDocument(t, s)
	for i, row := range doc.Iesire {
		if row.ProductID == nil {
			t.Fatalf("Iesire[%d].ProductID is nil, want the seeded product's id", i)
		}
		if *row.ProductID != products[i].ID {
			t.Errorf("Iesire[%d].ProductID = %d, want %d", i, *row.ProductID, products[i].ID)
		}
		if row.Denumire != products[i].Denumire {
			t.Errorf("Iesire[%d].Denumire = %q, want %q", i, row.Denumire, products[i].Denumire)
		}
	}
}

// The stored footer must agree with the rows it was stored with, exactly as it
// would if the user had typed the document in and pressed Salvează.
func TestOpenSeedsFirstDocumentWithConsistentFooter(t *testing.T) {
	s := newTestStore(t)

	doc := firstSeededDocument(t, s)
	intrare := calc.TotalsIntrare(doc.Intrare)
	iesire := calc.TotalsIesire(doc.Iesire)

	tip, val := calc.Diferenta(iesire.ValoareCuTVA, intrare.ValoareCuTVA)
	if doc.DiferentaTip != tip || doc.DiferentaValoare != val {
		t.Errorf("Diferenta = %q/%v, want %q/%v", doc.DiferentaTip, doc.DiferentaValoare, tip, val)
	}
	tip, val = calc.IncarcaDescarca(iesire.ValoareCuTVA, intrare.ValoareCuTVA)
	if doc.IncarcaDescarcaTip != tip || doc.IncarcaDescarcaValoare != val {
		t.Errorf("IncarcaDescarca = %q/%v, want %q/%v",
			doc.IncarcaDescarcaTip, doc.IncarcaDescarcaValoare, tip, val)
	}
	// The figures off the paper sheet, as a guard against a mistyped quantity.
	if intrare.ValoareCuTVA != 2214.52 {
		t.Errorf("total intrare cu TVA = %v, want 2214.52", intrare.ValoareCuTVA)
	}
	if iesire.ValoareCuTVA != 2791.4 {
		t.Errorf("total iesire cu TVA = %v, want 2791.40", iesire.ValoareCuTVA)
	}
	if val != 576.88 {
		t.Errorf("diferenta = %v, want 576.88", val)
	}
}

// A new document must not collide with the seeded one.
func TestOpenSeedsNextNrAfterTheSeededDocument(t *testing.T) {
	s := newTestStore(t)

	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if settings.NextNr != 2 {
		t.Errorf("NextNr = %d, want 2 (the seeded document already took NR 1)", settings.NextNr)
	}
}

// Seeding happens once. A user who deletes the example must not find it back
// on the next launch, exactly as with the product list.
func TestOpenDoesNotReseedDeletedDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	docs, err := s.ListDocuments()
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if err := s.DeleteDocument(docs[0].ID); err != nil {
		t.Fatalf("DeleteDocument: %v", err)
	}
	s.Close()

	s, err = Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s.Close()
	docs, err = s.ListDocuments()
	if err != nil {
		t.Fatalf("ListDocuments after reopen: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("len(docs) = %d after reopening, want 0 (re-seeding must not happen)", len(docs))
	}
}

// firstSeededDocument reads the document a fresh database was seeded with.
func firstSeededDocument(t *testing.T, s *Store) model.Document {
	t.Helper()
	docs, err := s.ListDocuments()
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if len(docs) == 0 {
		t.Fatal("no seeded document")
	}
	doc, err := s.GetDocument(docs[0].ID)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	return doc
}
