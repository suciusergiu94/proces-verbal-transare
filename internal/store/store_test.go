package store

import (
	"path/filepath"
	"testing"

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
	if version != 1 {
		t.Errorf("user_version = %d, want 1", version)
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
	if got.NextNr != 1 {
		t.Errorf("NextNr = %d, want 1", got.NextNr)
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
