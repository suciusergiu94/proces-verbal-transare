package store

import (
	"database/sql"
	"time"

	"proces-verbal-transare/internal/calc"
	"proces-verbal-transare/internal/model"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  unitate_nume TEXT NOT NULL DEFAULT '',
  next_nr INTEGER NOT NULL DEFAULT 1,
  cota_tva REAL NOT NULL DEFAULT 11
);

CREATE TABLE IF NOT EXISTS products (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  denumire TEXT NOT NULL,
  um TEXT NOT NULL DEFAULT 'Kg',
  pret_cu_tva REAL NOT NULL DEFAULT 0,
  ordine INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS documents (
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

CREATE TABLE IF NOT EXISTS document_intrare_rows (
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

CREATE TABLE IF NOT EXISTS document_iesire_rows (
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

CREATE INDEX IF NOT EXISTS idx_intrare_document ON document_intrare_rows(document_id, pozitie);
CREATE INDEX IF NOT EXISTS idx_iesire_document ON document_iesire_rows(document_id, pozitie);
`

// defaultUnitate is the company printed on the paper form.
const defaultUnitate = "S.C. Largiana Carn S.R.L."

// defaultCotaTVA is the standard Romanian TVA rate for food and food
// processing, in percent.
const defaultCotaTVA = 11.0

// seedProducts are the 19 items printed on the paper form, in printed order.
// Row 4 is printed as "Cotiet fara os", an evident typo for "Cotlet fara os".
var seedProducts = []model.Product{
	{Denumire: "Pulpa fara os", UM: "Kg", PretCuTVA: 21.9},
	{Denumire: "Muschiulet", UM: "Kg", PretCuTVA: 39.5},
	{Denumire: "Cotlet cu os", UM: "Kg", PretCuTVA: 24.0},
	{Denumire: "Cotlet fara os", UM: "Kg", PretCuTVA: 29.5},
	{Denumire: "Ceafa cu os", UM: "Kg", PretCuTVA: 24.0},
	{Denumire: "Ceafa fara os", UM: "Kg", PretCuTVA: 33.9},
	{Denumire: "Fleica", UM: "Kg", PretCuTVA: 30.5},
	{Denumire: "Costita", UM: "Kg", PretCuTVA: 29.0},
	{Denumire: "Ciolan", UM: "Kg", PretCuTVA: 16.5},
	{Denumire: "Ciolan dezosat", UM: "Kg", PretCuTVA: 25.0},
	{Denumire: "CPL", UM: "Kg", PretCuTVA: 20.9},
	{Denumire: "Spata porc", UM: "Kg", PretCuTVA: 21.9},
	{Denumire: "Slanina tablii", UM: "Kg", PretCuTVA: 18.5},
	{Denumire: "Slanina lucru", UM: "Kg", PretCuTVA: 12.0},
	{Denumire: "Oase garf", UM: "Kg", PretCuTVA: 6.5},
	{Denumire: "Picioare", UM: "Kg", PretCuTVA: 6.5},
	{Denumire: "Oase mici", UM: "Kg", PretCuTVA: 2.0},
	{Denumire: "Sorici", UM: "Kg", PretCuTVA: 2.0},
	{Denumire: "Deseu fara valoare", UM: "Kg", PretCuTVA: 0},
}

// seedIntrare and seedIesire are the proces verbal NR 1 that ships with the
// app, copied verbatim from the sheet it was first filled in on. A new install
// opens on a complete, real example rather than an empty list, which also
// gives the margin buttons something to work against on the first run.
//
// The iesire rows are one per seedProducts entry, in the same order, so the
// two lists are linked positionally when they are inserted (see migrate).
// Prices are stored as they were entered, three decimals and all: rounding
// 13.653 to 13.65 here would shift the totals and leave the footer disagreeing
// with the rows above it.
var seedIntrare = []model.IntrareRow{
	{Denumire: "Carcasa", UM: "Kg", Cantitate: 162.2, PretFaraTVA: 12.3, PretCuTVA: 13.653, CotaTVA: defaultCotaTVA},
}

// seedIesireCantitati is the quantity for each seedProducts entry, in order.
// The rest of every row — denumire, UM, price — is the product itself.
var seedIesireCantitati = []float64{
	15, 1.5, 2, 10, 1, 8.5, 11, 8.5, 2, 3.5, 12, 8, 10.5, 20.5, 12, 7.5, 18, 8.5, 1.2,
}

// seedIesirePreturiFaraTVA is each row's price without TVA as it was entered,
// rather than re-derived here: the stored document must keep the figures the
// sheet was signed with.
var seedIesirePreturiFaraTVA = []float64{
	19.73, 35.59, 21.62, 26.58, 21.62, 30.54, 27.48, 26.13, 14.86,
	22.52, 18.83, 19.73, 16.67, 10.81, 5.86, 5.86, 1.8, 1.8, 0,
}

// hasColumn reports whether a table already carries a column.
func hasColumn(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(`SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

// migrate creates the schema and seeds first-run data. It is safe to call on
// every startup: seeding happens only when the settings row is absent, so a
// user who deleted every product does not get them back on the next launch.
func migrate(db *sql.DB) error {
	if _, err := db.Exec(schemaSQL); err != nil {
		return err
	}

	// v2 added the TVA rate: one default in settings and one per document row.
	// CREATE TABLE IF NOT EXISTS above is a no-op on a database that already
	// has the tables, so a v1 database needs the columns added by hand. The
	// columns are added on absence rather than on the version marker, which
	// makes the upgrade safe to run against a database in any state — including
	// one interrupted halfway through a previous attempt.
	for _, c := range []struct{ table, column, ddl string }{
		{"settings", "cota_tva", `ALTER TABLE settings ADD COLUMN cota_tva REAL NOT NULL DEFAULT 11`},
		{"document_intrare_rows", "cota_tva", `ALTER TABLE document_intrare_rows ADD COLUMN cota_tva REAL NOT NULL DEFAULT 11`},
		{"document_iesire_rows", "cota_tva", `ALTER TABLE document_iesire_rows ADD COLUMN cota_tva REAL NOT NULL DEFAULT 11`},
	} {
		has, err := hasColumn(db, c.table, c.column)
		if err != nil {
			return err
		}
		if has {
			continue
		}
		if _, err := db.Exec(c.ddl); err != nil {
			return err
		}
	}

	// Stamp the schema version so a future migration can tell this shape apart
	// from whatever comes after it. Future migrations should switch on the
	// current value of PRAGMA user_version.
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version < 2 {
		if _, err := db.Exec(`PRAGMA user_version = 2`); err != nil {
			return err
		}
	}

	var seeded int
	if err := db.QueryRow(`SELECT COUNT(*) FROM settings WHERE id = 1`).Scan(&seeded); err != nil {
		return err
	}
	if seeded > 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// NR 1 is taken by the seeded document below, so a new document starts at 2.
	if _, err := tx.Exec(
		`INSERT INTO settings (id, unitate_nume, next_nr, cota_tva) VALUES (1, ?, ?, ?)`,
		defaultUnitate, seedDocumentNr+1, defaultCotaTVA,
	); err != nil {
		return err
	}

	// The product ids are collected as they are inserted rather than assumed to
	// run 1..19: the seeded document links to them, and that link must hold
	// whatever ids SQLite hands out.
	productIDs := make([]int64, len(seedProducts))
	for i, p := range seedProducts {
		res, err := tx.Exec(
			`INSERT INTO products (denumire, um, pret_cu_tva, ordine) VALUES (?, ?, ?, ?)`,
			p.Denumire, p.UM, p.PretCuTVA, i,
		)
		if err != nil {
			return err
		}
		if productIDs[i], err = res.LastInsertId(); err != nil {
			return err
		}
	}

	if err := seedFirstDocument(tx, productIDs); err != nil {
		return err
	}
	return tx.Commit()
}

// seedDocumentNr is the number the shipped proces verbal carries.
const seedDocumentNr = 1

// seedFirstDocument inserts the shipped proces verbal and its two row tables.
//
// The document is dated the day it is seeded rather than the day the original
// sheet was written, so a fresh install does not open on a document that looks
// months stale. Its footer is derived from its own rows by the same calc the
// form and the PDF use, so the stored figures agree with the rows exactly as
// they would had the user typed the document in and pressed Salvează.
func seedFirstDocument(tx *sql.Tx, productIDs []int64) error {
	now := time.Now()
	iesire := make([]model.IesireRow, len(seedProducts))
	for i, p := range seedProducts {
		iesire[i] = model.IesireRow{
			ProductID:   &productIDs[i],
			Pozitie:     i,
			Denumire:    p.Denumire,
			UM:          p.UM,
			PretCuTVA:   p.PretCuTVA,
			Cantitate:   seedIesireCantitati[i],
			PretFaraTVA: seedIesirePreturiFaraTVA[i],
			CotaTVA:     defaultCotaTVA,
		}
	}

	totalIntrare := calc.TotalsIntrare(seedIntrare).ValoareCuTVA
	totalIesire := calc.TotalsIesire(iesire).ValoareCuTVA
	difTip, difVal := calc.Diferenta(totalIesire, totalIntrare)
	incTip, incVal := calc.IncarcaDescarca(totalIesire, totalIntrare)

	res, err := tx.Exec(
		`INSERT INTO documents (nr, data, gestiune, document_referinta, diferenta_tip,
		         diferenta_valoare, incarca_descarca_tip, incarca_descarca_valoare,
		         gestionar, calculator, vizat_compartiment_productie, created_at, updated_at)
		 VALUES (?, ?, '', '', ?, ?, ?, ?, '', '', '', ?, ?)`,
		seedDocumentNr, now.Format("2006-01-02"), difTip, difVal, incTip, incVal,
		now.UTC().Format(time.RFC3339), now.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return err
	}
	documentID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	for i, r := range seedIntrare {
		if _, err := tx.Exec(
			`INSERT INTO document_intrare_rows (document_id, pozitie, denumire, um,
			         cantitate, pret_fara_tva, pret_cu_tva, cota_tva)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			documentID, i, r.Denumire, r.UM, r.Cantitate, r.PretFaraTVA, r.PretCuTVA, r.CotaTVA,
		); err != nil {
			return err
		}
	}
	for _, r := range iesire {
		if _, err := tx.Exec(
			`INSERT INTO document_iesire_rows (document_id, product_id, pozitie, denumire,
			         um, pret_cu_tva, cantitate, pret_fara_tva, cota_tva)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			documentID, *r.ProductID, r.Pozitie, r.Denumire, r.UM, r.PretCuTVA,
			r.Cantitate, r.PretFaraTVA, r.CotaTVA,
		); err != nil {
			return err
		}
	}
	return nil
}
