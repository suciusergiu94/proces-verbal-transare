package main

import (
	"path/filepath"
	"testing"

	"proces-verbal-transare/internal/model"
	"proces-verbal-transare/internal/store"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return &App{store: s}
}

// newEmptyTestApp is newTestApp with the shipped proces verbal removed, for
// the tests that are about a store with no documents in it at all.
func newEmptyTestApp(t *testing.T) *App {
	t.Helper()
	a := newTestApp(t)
	docs, err := a.ListDocuments()
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	for _, d := range docs {
		if err := a.DeleteDocument(d.ID); err != nil {
			t.Fatalf("DeleteDocument: %v", err)
		}
	}
	return a
}

// A brand-new install already has the shipped proces verbal behind it, so the
// first document the user starts is numbered after it and carries its intrare
// row forward — the same prefill any other saved document would give.
func TestNewDocumentDraftOnFreshInstall(t *testing.T) {
	a := newTestApp(t)

	draft, err := a.NewDocumentDraft()
	if err != nil {
		t.Fatalf("NewDocumentDraft: %v", err)
	}
	if draft.Nr != 2 {
		t.Errorf("draft.Nr = %d, want 2 (the seeded document took NR 1)", draft.Nr)
	}
	if len(draft.Intrare) != 1 || draft.Intrare[0].Denumire != "Carcasa" {
		t.Errorf("draft.Intrare = %+v, want the seeded document's intrare row", draft.Intrare)
	}
	if draft.Intrare[0].ID != 0 {
		t.Errorf("draft.Intrare[0].ID = %d, want 0 (prefilled rows must be unsaved)", draft.Intrare[0].ID)
	}
}

func TestNewDocumentDraftWithoutHistory(t *testing.T) {
	a := newEmptyTestApp(t)

	draft, err := a.NewDocumentDraft()
	if err != nil {
		t.Fatalf("NewDocumentDraft: %v", err)
	}
	if draft.ID != 0 {
		t.Errorf("draft.ID = %d, want 0", draft.ID)
	}
	if draft.Nr != 2 {
		t.Errorf("draft.Nr = %d, want 2 (settings.NextNr)", draft.Nr)
	}
	if draft.Data == "" {
		t.Error("draft.Data is empty, want today's ISO date")
	}
	if len(draft.Intrare) != 1 || draft.Intrare[0].Denumire != "" {
		t.Errorf("draft.Intrare = %+v, want one blank row", draft.Intrare)
	}
	if len(draft.Iesire) != 19 {
		t.Fatalf("len(draft.Iesire) = %d, want 19 (one per seeded product)", len(draft.Iesire))
	}
	if draft.Iesire[0].Denumire != "Pulpa fara os" || draft.Iesire[0].PretCuTVA != 21.9 {
		t.Errorf("draft.Iesire[0] = %+v, want the first product prefilled", draft.Iesire[0])
	}
	if draft.Iesire[0].ProductID == nil {
		t.Error("draft.Iesire[0].ProductID is nil, want the product id")
	}
	if draft.Iesire[0].Cantitate != 0 {
		t.Errorf("draft.Iesire[0].Cantitate = %v, want 0", draft.Iesire[0].Cantitate)
	}
}

func TestNewDocumentDraftPrefillsFromLastDocument(t *testing.T) {
	a := newTestApp(t)

	first, err := a.NewDocumentDraft()
	if err != nil {
		t.Fatalf("NewDocumentDraft: %v", err)
	}
	first.Nr = 193
	first.Gestiune = "Magazin Bradet"
	first.Intrare = []model.IntrareRow{
		{Denumire: "Carcasa porc f cap", UM: "Kg", Cantitate: 162.20, PretFaraTVA: 12.50, PretCuTVA: 14.03},
	}
	if _, err := a.SaveDocument(first); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	draft, err := a.NewDocumentDraft()
	if err != nil {
		t.Fatalf("NewDocumentDraft: %v", err)
	}
	if draft.Nr != 194 {
		t.Errorf("draft.Nr = %d, want 194 (next_nr bumped after saving 193)", draft.Nr)
	}
	if draft.Gestiune != "Magazin Bradet" {
		t.Errorf("draft.Gestiune = %q, want the previous gestiune", draft.Gestiune)
	}
	if len(draft.Intrare) != 1 || draft.Intrare[0].Denumire != "Carcasa porc f cap" {
		t.Fatalf("draft.Intrare = %+v, want the previous intrare rows", draft.Intrare)
	}
	if draft.Intrare[0].ID != 0 {
		t.Errorf("draft.Intrare[0].ID = %d, want 0 (prefilled rows must be unsaved)", draft.Intrare[0].ID)
	}
	if draft.Intrare[0].PretCuTVA != 14.03 {
		t.Errorf("draft.Intrare[0].PretCuTVA = %v, want the previous price", draft.Intrare[0].PretCuTVA)
	}
	// The prices carry over, the quantity does not: every carcass is weighed
	// afresh, so a quantity left over from the previous document would be
	// wrong on every new one and silently drive the whole "ce iese" split.
	if draft.Intrare[0].Cantitate != 0 {
		t.Errorf("draft.Intrare[0].Cantitate = %v, want 0", draft.Intrare[0].Cantitate)
	}
}

func TestSaveDocumentDoesNotLowerNextNr(t *testing.T) {
	a := newTestApp(t)

	if err := a.SaveSettings(model.Settings{UnitateNume: "X", NextNr: 200}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	draft, err := a.NewDocumentDraft()
	if err != nil {
		t.Fatalf("NewDocumentDraft: %v", err)
	}
	draft.Nr = 5 // user typed an older number
	if _, err := a.SaveDocument(draft); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	settings, err := a.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if settings.NextNr != 200 {
		t.Errorf("NextNr = %d, want 200 (an older Nr must not rewind the counter)", settings.NextNr)
	}
}

func TestSaveDocumentUpdateDoesNotBumpNextNr(t *testing.T) {
	a := newTestApp(t)

	draft, err := a.NewDocumentDraft()
	if err != nil {
		t.Fatalf("NewDocumentDraft: %v", err)
	}
	draft.Nr = 10
	saved, err := a.SaveDocument(draft)
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	after, err := a.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if after.NextNr != 11 {
		t.Fatalf("NextNr = %d after inserting Nr 10, want 11", after.NextNr)
	}

	saved.Gestiune = "editat"
	if _, err := a.SaveDocument(saved); err != nil {
		t.Fatalf("SaveDocument (update): %v", err)
	}
	after, err = a.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if after.NextNr != 11 {
		t.Errorf("NextNr = %d after re-saving an existing document, want 11", after.NextNr)
	}
}

func TestNewDocumentDraftAppliesTheDefaultTvaRate(t *testing.T) {
	a := newTestApp(t)

	draft, err := a.NewDocumentDraft()
	if err != nil {
		t.Fatalf("NewDocumentDraft: %v", err)
	}
	if draft.Intrare[0].CotaTVA != 11 {
		t.Errorf("draft.Intrare[0].CotaTVA = %v, want 11 (the settings default)", draft.Intrare[0].CotaTVA)
	}
	if draft.Iesire[0].CotaTVA != 11 {
		t.Errorf("draft.Iesire[0].CotaTVA = %v, want 11 (the settings default)", draft.Iesire[0].CotaTVA)
	}
	// Pulpa fara os is seeded at 21.9 with TVA; at 11% that is 19.73 without.
	if draft.Iesire[0].PretFaraTVA != 19.73 {
		t.Errorf("draft.Iesire[0].PretFaraTVA = %v, want 19.73 derived from 21.9 at 11%%", draft.Iesire[0].PretFaraTVA)
	}
}

func TestNewDocumentDraftUsesTheConfiguredTvaRate(t *testing.T) {
	a := newTestApp(t)

	if err := a.SaveSettings(model.Settings{UnitateNume: "X", NextNr: 1, CotaTVA: 0}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	draft, err := a.NewDocumentDraft()
	if err != nil {
		t.Fatalf("NewDocumentDraft: %v", err)
	}
	if draft.Iesire[0].CotaTVA != 0 {
		t.Errorf("draft.Iesire[0].CotaTVA = %v, want 0 (the configured rate)", draft.Iesire[0].CotaTVA)
	}
	if draft.Iesire[0].PretFaraTVA != 21.9 {
		t.Errorf("draft.Iesire[0].PretFaraTVA = %v, want 21.9 (a zero rate leaves the price alone)", draft.Iesire[0].PretFaraTVA)
	}
}

func TestNewDocumentDraftKeepsThePreviousRowsOwnRate(t *testing.T) {
	a := newTestApp(t)

	first, err := a.NewDocumentDraft()
	if err != nil {
		t.Fatalf("NewDocumentDraft: %v", err)
	}
	first.Intrare = []model.IntrareRow{
		{Denumire: "Carcasa porc f cap", UM: "Kg", Cantitate: 100, PretFaraTVA: 12.50, PretCuTVA: 15.13, CotaTVA: 21},
	}
	if _, err := a.SaveDocument(first); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	draft, err := a.NewDocumentDraft()
	if err != nil {
		t.Fatalf("NewDocumentDraft: %v", err)
	}
	if draft.Intrare[0].CotaTVA != 21 {
		t.Errorf("draft.Intrare[0].CotaTVA = %v, want 21 (carried over with the row, not reset to the default)", draft.Intrare[0].CotaTVA)
	}
}
