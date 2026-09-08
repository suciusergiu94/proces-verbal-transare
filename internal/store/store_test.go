package store

import (
	"database/sql"
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

func TestOpenSeedsOneTemplateWithEveryProduct(t *testing.T) {
	s := newTestStore(t)

	templates, err := s.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("len(templates) = %d, want 1", len(templates))
	}
	if templates[0].Nume != "Carcasa Porc" {
		t.Errorf("nume = %q, want %q", templates[0].Nume, "Carcasa Porc")
	}
	if len(templates[0].Products) != len(seedProducts) {
		t.Fatalf("len(products) = %d, want %d", len(templates[0].Products), len(seedProducts))
	}
	for i, p := range templates[0].Products {
		if p.Denumire != seedProducts[i].Denumire {
			t.Errorf("product %d denumire = %q, want %q", i, p.Denumire, seedProducts[i].Denumire)
		}
		if p.TemplateID != templates[0].ID {
			t.Errorf("product %d templateId = %d, want %d", i, p.TemplateID, templates[0].ID)
		}
		if p.Ordine != i {
			t.Errorf("product %d ordine = %d, want %d", i, p.Ordine, i)
		}
	}
}

func TestOpenSeedsRatiosSummingToExactlyOneHundred(t *testing.T) {
	s := newTestStore(t)

	templates, err := s.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	var total float64
	for _, p := range templates[0].Products {
		total += p.ProcentDinIntrare
	}
	if calc.Round3(total) != 100 {
		t.Errorf("total procente = %v, want exactly 100", calc.Round3(total))
	}
}

func TestOpenSeedsFirstDocumentStampedWithTheTemplate(t *testing.T) {
	s := newTestStore(t)

	templates, err := s.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	docs, err := s.ListDocuments()
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("len(docs) = %d, want 1", len(docs))
	}
	doc, err := s.GetDocument(docs[0].ID)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if doc.TemplateID == nil {
		t.Fatal("doc.TemplateID = nil, want the seeded template")
	}
	if *doc.TemplateID != templates[0].ID {
		t.Errorf("doc.TemplateID = %d, want %d", *doc.TemplateID, templates[0].ID)
	}
}

// A database written by a pre-release build has no templates table and a
// products table with no template_id, so every query would fail on it. It is
// dropped and reseeded rather than migrated: the app has never shipped, and
// carrying an upgrade path for it would mean carrying it forever.
func TestOpenResetsAPreReleaseDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE settings (id INTEGER PRIMARY KEY CHECK (id = 1), unitate_nume TEXT NOT NULL DEFAULT '',
		  next_nr INTEGER NOT NULL DEFAULT 1, cota_tva REAL NOT NULL DEFAULT 11, gestiune TEXT NOT NULL DEFAULT '');
		CREATE TABLE products (id INTEGER PRIMARY KEY AUTOINCREMENT, denumire TEXT NOT NULL,
		  um TEXT NOT NULL DEFAULT 'Kg', pret_cu_tva REAL NOT NULL DEFAULT 0,
		  procent_din_intrare REAL NOT NULL DEFAULT 0, ordine INTEGER NOT NULL);
		INSERT INTO settings (id, unitate_nume, next_nr) VALUES (1, 'Vechi SRL', 42);
		INSERT INTO products (denumire, um, pret_cu_tva, procent_din_intrare, ordine)
		  VALUES ('Ceva vechi', 'Kg', 1, 100, 0);
		PRAGMA user_version = 4;
	`); err != nil {
		t.Fatalf("build v4 database: %v", err)
	}
	db.Close()

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	templates, err := s.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(templates) != 1 || len(templates[0].Products) != len(seedProducts) {
		t.Fatalf("got %d templates with %d products, want 1 with %d",
			len(templates), len(templates[0].Products), len(seedProducts))
	}
	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if settings.UnitateNume != defaultUnitate {
		t.Errorf("unitate = %q, want the seeded %q — the old row survived the reset",
			settings.UnitateNume, defaultUnitate)
	}
}

// A database already at the current version is left exactly as the user left
// it. Without this guard the reset above would wipe real data on every start.
func TestOpenDoesNotResetACurrentDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "current.db")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	settings.UnitateNume = "Altceva SRL"
	if err := s.SaveSettings(settings); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	s.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s2.Close()
	got, err := s2.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got.UnitateNume != "Altceva SRL" {
		t.Errorf("unitate = %q, want %q — the database was reset when it should not have been",
			got.UnitateNume, "Altceva SRL")
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	s1, err := Open(path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	if _, err := s1.db.Exec(`DELETE FROM products`); err != nil {
		t.Fatalf("delete products: %v", err)
	}
	s1.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer s2.Close()

	templates, err := s2.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("len(templates) = %d, want 1", len(templates))
	}
	if len(templates[0].Products) != 0 {
		t.Errorf("len(products) = %d after reopening an emptied DB, want 0 (re-seeding must not happen)", len(templates[0].Products))
	}
}

func TestOpenSetsUserVersion(t *testing.T) {
	s := newTestStore(t)

	var version int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatalf("PRAGMA user_version: %v", err)
	}
	if version != 5 {
		t.Errorf("user_version = %d, want 5", version)
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
	if got.Gestiune != "Magazin Bradet" {
		t.Errorf("Gestiune = %q, want the seeded default", got.Gestiune)
	}

	if err := s.SaveSettings(model.Settings{
		UnitateNume: "Alt SRL",
		NextNr:      194,
		Gestiune:    "Magazin Ocolis",
	}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	got, err = s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings after save: %v", err)
	}
	if got.UnitateNume != "Alt SRL" || got.NextNr != 194 || got.Gestiune != "Magazin Ocolis" {
		t.Errorf("settings = %+v, want {Alt SRL 194 Magazin Ocolis}", got)
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
	// The shipped document is booked against the same default a fresh install
	// starts from, so the example agrees with the settings beside it.
	if docs[0].Gestiune != "Magazin Bradet" {
		t.Errorf("docs[0].Gestiune = %q, want the seeded default", docs[0].Gestiune)
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
	if doc.Iesire[18].Denumire != "Deseu fara valoare" || doc.Iesire[18].Cantitate != 2.2 {
		t.Errorf("Iesire[18] = %+v, want Deseu fara valoare / 2.2", doc.Iesire[18])
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

	templates, err := s.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	products := templates[0].Products
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

func TestOpenSeedsProductPercentages(t *testing.T) {
	s := newTestStore(t)

	templates, err := s.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	products := templates[0].Products

	// Each ratio is the product's share of the 162.2 Kg carcass the shipped
	// proces verbal was cut from: 15 / 162.2 = 9.248%, 1.5 / 162.2 = 0.925%.
	want := map[int]float64{0: 9.248, 1: 0.925, 3: 6.165, 13: 12.639}
	for i, w := range want {
		if products[i].ProcentDinIntrare != w {
			t.Errorf("products[%d] (%s).ProcentDinIntrare = %v, want %v",
				i, products[i].Denumire, products[i].ProcentDinIntrare, w)
		}
	}

	// Deseu fara valoare closes the list: it carries whatever the named cuts
	// leave over, so the column reaches exactly 100%.
	if products[18].ProcentDinIntrare != 1.358 {
		t.Errorf("products[18] (%s).ProcentDinIntrare = %v, want 1.358 (the residual)",
			products[18].Denumire, products[18].ProcentDinIntrare)
	}

	var total float64
	for _, p := range products {
		total += p.ProcentDinIntrare
	}
	if calc.Round3(total) != 100 {
		t.Errorf("sum of ProcentDinIntrare = %v, want exactly 100", calc.Round3(total))
	}
}
