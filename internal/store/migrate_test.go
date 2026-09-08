package store

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// v1SchemaSQL is the schema as it shipped before the TVA rate existed. The
// migration test builds a database from it so the upgrade path is exercised
// against the real thing rather than against the current schema.
const v1SchemaSQL = `
CREATE TABLE settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  unitate_nume TEXT NOT NULL DEFAULT '',
  next_nr INTEGER NOT NULL DEFAULT 1
);
CREATE TABLE products (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  denumire TEXT NOT NULL,
  um TEXT NOT NULL DEFAULT 'Kg',
  pret_cu_tva REAL NOT NULL DEFAULT 0,
  ordine INTEGER NOT NULL
);
CREATE TABLE documents (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  nr INTEGER NOT NULL,
  data TEXT NOT NULL,
  gestiune TEXT NOT NULL,
  document_referinta TEXT NOT NULL DEFAULT '',
  diferenta_tip TEXT NOT NULL DEFAULT '',
  diferenta_valoare REAL NOT NULL DEFAULT 0,
  incarca_descarca_tip TEXT NOT NULL DEFAULT '',
  incarca_descarca_valoare REAL NOT NULL DEFAULT 0,
  gestionar TEXT NOT NULL DEFAULT '',
  calculator TEXT NOT NULL DEFAULT '',
  vizat_compartiment_productie TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE document_intrare_rows (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  document_id INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  pozitie INTEGER NOT NULL,
  denumire TEXT NOT NULL DEFAULT '',
  um TEXT NOT NULL DEFAULT '',
  cantitate REAL NOT NULL DEFAULT 0,
  pret_fara_tva REAL NOT NULL DEFAULT 0,
  pret_cu_tva REAL NOT NULL DEFAULT 0
);
CREATE TABLE document_iesire_rows (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  document_id INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  product_id INTEGER REFERENCES products(id) ON DELETE SET NULL,
  pozitie INTEGER NOT NULL,
  denumire TEXT NOT NULL,
  um TEXT NOT NULL,
  pret_cu_tva REAL NOT NULL,
  cantitate REAL NOT NULL DEFAULT 0,
  pret_fara_tva REAL NOT NULL DEFAULT 0
);
PRAGMA user_version = 1;
INSERT INTO settings (id, unitate_nume, next_nr) VALUES (1, 'S.C. Vechi S.R.L.', 42);
INSERT INTO documents (nr, data, gestiune, created_at, updated_at)
  VALUES (7, '2026-01-15', 'Magazin Bradet', '2026-01-15T10:00:00Z', '2026-01-15T10:00:00Z');
INSERT INTO document_intrare_rows (document_id, pozitie, denumire, um, cantitate, pret_fara_tva, pret_cu_tva)
  VALUES (1, 0, 'Carcasa porc f cap', 'Kg', 100, 12.5, 14.03);
INSERT INTO document_iesire_rows (document_id, pozitie, denumire, um, pret_cu_tva, cantitate, pret_fara_tva)
  VALUES (1, 0, 'Pulpa fara os', 'Kg', 21.9, 15, 16.5);
`

// writeV1Database creates a pre-TVA-rate database and returns its path.
func writeV1Database(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "v1.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(v1SchemaSQL); err != nil {
		t.Fatalf("build v1 schema: %v", err)
	}
	return path
}

func TestOpenUpgradesV1DatabaseWithDefaultTvaRate(t *testing.T) {
	path := writeV1Database(t)

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open on a v1 database: %v", err)
	}
	defer s.Close()

	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if settings.CotaTVA != 11 {
		t.Errorf("CotaTVA = %v, want 11", settings.CotaTVA)
	}
	if settings.UnitateNume != "S.C. Vechi S.R.L." || settings.NextNr != 42 {
		t.Errorf("settings = %+v, want the existing values preserved", settings)
	}

	doc, err := s.GetDocument(1)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if len(doc.Intrare) != 1 || doc.Intrare[0].CotaTVA != 11 {
		t.Errorf("doc.Intrare = %+v, want one row at the default rate", doc.Intrare)
	}
	if len(doc.Iesire) != 1 || doc.Iesire[0].CotaTVA != 11 {
		t.Errorf("doc.Iesire = %+v, want one row at the default rate", doc.Iesire)
	}
	if doc.Intrare[0].PretCuTVA != 14.03 || doc.Iesire[0].PretFaraTVA != 16.5 {
		t.Error("existing prices must be left exactly as saved, not recomputed from the rate")
	}
}

func TestOpenIsIdempotentOnAnUpgradedDatabase(t *testing.T) {
	path := writeV1Database(t)

	first, err := Open(path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	first.Close()

	second, err := Open(path)
	if err != nil {
		t.Fatalf("second Open on an already-upgraded database: %v", err)
	}
	defer second.Close()

	settings, err := second.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if settings.UnitateNume != "S.C. Vechi S.R.L." {
		t.Errorf("UnitateNume = %q, want the existing value untouched by re-migration", settings.UnitateNume)
	}
}

// v2SchemaSQL is the schema as it shipped with the TVA rate but before the
// carcass ratios. Its data stands in for an install that has been in use for a
// while: a product list and a document linking to it.
const v2SchemaSQL = `
CREATE TABLE settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  unitate_nume TEXT NOT NULL DEFAULT '',
  next_nr INTEGER NOT NULL DEFAULT 1,
  cota_tva REAL NOT NULL DEFAULT 11
);
CREATE TABLE products (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  denumire TEXT NOT NULL,
  um TEXT NOT NULL DEFAULT 'Kg',
  pret_cu_tva REAL NOT NULL DEFAULT 0,
  ordine INTEGER NOT NULL
);
CREATE TABLE documents (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  nr INTEGER NOT NULL,
  data TEXT NOT NULL,
  gestiune TEXT NOT NULL,
  document_referinta TEXT NOT NULL DEFAULT '',
  diferenta_tip TEXT NOT NULL DEFAULT '',
  diferenta_valoare REAL NOT NULL DEFAULT 0,
  incarca_descarca_tip TEXT NOT NULL DEFAULT '',
  incarca_descarca_valoare REAL NOT NULL DEFAULT 0,
  gestionar TEXT NOT NULL DEFAULT '',
  calculator TEXT NOT NULL DEFAULT '',
  vizat_compartiment_productie TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE document_intrare_rows (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  document_id INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  pozitie INTEGER NOT NULL,
  denumire TEXT NOT NULL DEFAULT '',
  um TEXT NOT NULL DEFAULT '',
  cantitate REAL NOT NULL DEFAULT 0,
  pret_fara_tva REAL NOT NULL DEFAULT 0,
  pret_cu_tva REAL NOT NULL DEFAULT 0,
  cota_tva REAL NOT NULL DEFAULT 11
);
CREATE TABLE document_iesire_rows (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  document_id INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  product_id INTEGER REFERENCES products(id) ON DELETE SET NULL,
  pozitie INTEGER NOT NULL,
  denumire TEXT NOT NULL,
  um TEXT NOT NULL,
  pret_cu_tva REAL NOT NULL,
  cantitate REAL NOT NULL DEFAULT 0,
  pret_fara_tva REAL NOT NULL DEFAULT 0,
  cota_tva REAL NOT NULL DEFAULT 11
);
PRAGMA user_version = 2;
INSERT INTO settings (id, unitate_nume, next_nr, cota_tva) VALUES (1, 'S.C. Vechi S.R.L.', 42, 11);
INSERT INTO products (id, denumire, um, pret_cu_tva, ordine) VALUES
  (1, 'Pulpa fara os', 'Kg', 21.9, 0),
  (2, 'Slanina', 'Kg', 12.0, 1),
  (3, 'Deseu fara valoare', 'Kg', 0, 2);
-- Two documents: the ratios must come from the older one, not the newer.
INSERT INTO documents (id, nr, data, gestiune, created_at, updated_at) VALUES
  (1, 1, '2026-01-15', '', '2026-01-15T10:00:00Z', '2026-01-15T10:00:00Z'),
  (2, 2, '2026-02-15', '', '2026-02-15T10:00:00Z', '2026-02-15T10:00:00Z');
INSERT INTO document_intrare_rows (document_id, pozitie, denumire, um, cantitate, pret_fara_tva, pret_cu_tva) VALUES
  (1, 0, 'Carcasa', 'Kg', 80, 12.5, 13.88),
  (1, 1, 'Carcasa', 'Kg', 20, 12.5, 13.88),
  (2, 0, 'Carcasa', 'Kg', 100, 12.5, 13.88);
INSERT INTO document_iesire_rows (document_id, product_id, pozitie, denumire, um, pret_cu_tva, cantitate, pret_fara_tva) VALUES
  (1, 1, 0, 'Pulpa fara os', 'Kg', 21.9, 60, 19.73),
  (1, 2, 1, 'Slanina', 'Kg', 12.0, 30, 10.81),
  (1, 3, 2, 'Deseu fara valoare', 'Kg', 0, 5, 0),
  (2, 1, 0, 'Pulpa fara os', 'Kg', 21.9, 10, 19.73);
`

// writeV2Database creates a pre-ratio database in use and returns its path.
func writeV2Database(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "v2.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(v2SchemaSQL); err != nil {
		t.Fatalf("build v2 schema: %v", err)
	}
	return path
}

func TestOpenBackfillsProcentDinIntrareFromTheOldestDocument(t *testing.T) {
	path := writeV2Database(t)

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open on a v2 database: %v", err)
	}
	defer s.Close()

	products, err := s.ListProducts()
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}

	// Document 1 put 100 Kg in (80 + 20) and got 60 Kg of pulpa and 30 Kg of
	// slanina out. The last product closes the list at whatever is left, so the
	// 5 Kg of deseu it actually recorded becomes 10% — the 5 Kg the butchering
	// lost is waste too, and the column has to reach 100%.
	want := []float64{60, 30, 10}
	for i, w := range want {
		if products[i].ProcentDinIntrare != w {
			t.Errorf("products[%d] (%s).ProcentDinIntrare = %v, want %v",
				i, products[i].Denumire, products[i].ProcentDinIntrare, w)
		}
	}
}

func TestOpenDoesNotOverwriteProcentDinIntrareOnLaterStarts(t *testing.T) {
	path := writeV2Database(t)

	first, err := Open(path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	products, err := first.ListProducts()
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	products[0].ProcentDinIntrare = 55
	products[1].ProcentDinIntrare = 35
	products[2].ProcentDinIntrare = 10
	if err := first.SaveProducts(products); err != nil {
		t.Fatalf("SaveProducts: %v", err)
	}
	first.Close()

	second, err := Open(path)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer second.Close()

	reopened, err := second.ListProducts()
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if reopened[0].ProcentDinIntrare != 55 {
		t.Errorf("products[0].ProcentDinIntrare = %v after reopening, want 55 (the backfill must run once)",
			reopened[0].ProcentDinIntrare)
	}
}
