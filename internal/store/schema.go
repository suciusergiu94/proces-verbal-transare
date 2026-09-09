package store

import (
	"database/sql"
	"fmt"
	"time"

	"proces-verbal-transare/internal/calc"
	"proces-verbal-transare/internal/model"
)

const schemaVersion = 5

const schemaSQL = `
CREATE TABLE IF NOT EXISTS settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  unitate_nume TEXT NOT NULL DEFAULT '',
  next_nr INTEGER NOT NULL DEFAULT 1,
  cota_tva REAL NOT NULL DEFAULT 11,
  gestiune TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS templates (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  nume TEXT NOT NULL,
  ordine INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS products (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  template_id INTEGER NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
  denumire TEXT NOT NULL,
  um TEXT NOT NULL DEFAULT 'Kg',
  pret_cu_tva REAL NOT NULL DEFAULT 0,
  procent_din_intrare REAL NOT NULL DEFAULT 0,
  ordine INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS documents (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  template_id INTEGER REFERENCES templates(id) ON DELETE SET NULL,
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

CREATE INDEX IF NOT EXISTS idx_products_template ON products(template_id, ordine);
CREATE INDEX IF NOT EXISTS idx_intrare_document ON document_intrare_rows(document_id, pozitie);
CREATE INDEX IF NOT EXISTS idx_iesire_document ON document_iesire_rows(document_id, pozitie);
`

// defaultTemplateNume is the template a fresh install is seeded with. It is
// copied onto the "ce intra" row of a document created from a template that
// has no documents yet, so it reads as the thing being butchered rather than
// as a category.
const defaultTemplateNume = "Carcasa Porc"

// defaultUnitate is the company printed on the paper form.
const defaultUnitate = "S.C. Largiana Carn S.R.L."

// defaultCotaTVA is the standard Romanian TVA rate for food and food
// processing, in percent.
const defaultCotaTVA = 11.0

// defaultGestiune is the gestiune a fresh install starts from, and the one the
// shipped proces verbal is booked against.
const defaultGestiune = "Magazin Bradet"

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
//
// The sheet this was copied from booked 1.2 Kg of deseu and left the remaining
// kilogram of the 162.2 Kg carcass unaccounted for. It is booked as deseu here
// instead, so the document accounts for the whole carcass and agrees with the
// ratios in seedProcente. Deseu is priced at 0, so this moves no money at all:
// the row's value, both totals in lei, Diferență and the margin are all exactly
// what the signed sheet showed.
var seedIesireCantitati = []float64{
	15, 1.5, 2, 10, 1, 8.5, 11, 8.5, 2, 3.5, 12, 8, 10.5, 20.5, 12, 7.5, 18, 8.5, 2.2,
}

// procenteDinCantitati turns what each product yielded into its share of what
// went in, in percent, to three decimals.
//
// The last entry takes whatever the others leave over instead of its own
// quotient, for two reasons. Independently rounded ratios miss 100% by a
// thousandth or two, and the column has to land on it exactly — a carcass is
// fully accounted for or the arithmetic is wrong. And a butchering rarely
// yields back everything it consumed: the kilogram that evaporates has to be
// booked somewhere. Both corrections go to the row that closes the list, which
// in this app is Deseu fara valoare — the right place for meat that turned into
// nothing in particular, and priced at 0, so absorbing them costs nothing.
//
// Everything comes back zero when nothing went in, and the residual is clamped
// at zero if the products somehow already account for more than the input; the
// column then does not reach 100% and Setări says so rather than this quietly
// inventing a negative yield.
func procenteDinCantitati(cantitati []float64, totalIntrare float64) []float64 {
	out := make([]float64, len(cantitati))
	if totalIntrare <= 0 || len(cantitati) == 0 {
		return out
	}
	rest := 100.0
	for i := 0; i < len(out)-1; i++ {
		out[i] = calc.Round3(cantitati[i] / totalIntrare * 100)
		rest -= out[i]
	}
	if rest < 0 {
		rest = 0
	}
	out[len(out)-1] = calc.Round3(rest)
	return out
}

// seedProcente is each seedProducts entry's share of the seeded carcass,
// derived from the quantities above rather than written out by hand so the two
// cannot drift apart.
func seedProcente() []float64 {
	return procenteDinCantitati(seedIesireCantitati, calc.TotalsIntrare(seedIntrare).Cantitate)
}

// seedIesirePreturiFaraTVA is each row's price without TVA as it was entered,
// rather than re-derived here: the stored document must keep the figures the
// sheet was signed with.
var seedIesirePreturiFaraTVA = []float64{
	19.73, 35.59, 21.62, 26.58, 21.62, 30.54, 27.48, 26.13, 14.86,
	22.52, 18.83, 19.73, 16.67, 10.81, 5.86, 5.86, 1.8, 1.8, 0,
}

// migrate brings the database to schemaVersion and seeds first-run data. It is
// safe to call on every startup: seeding happens only when the settings row is
// absent, so a user who deleted every product does not get them back.
func migrate(db *sql.DB) error {
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}

	// A database stamped below the current version was written by a
	// pre-release build, whose products table has no template_id and which has
	// no templates table at all — every query below would fail on it. The app
	// has never shipped, so there is nothing in such a database worth an
	// upgrade path that would then have to be carried forever; it is dropped
	// and reseeded instead.
	//
	// Version 0 is a database that does not exist yet, which needs no reset.
	//
	// This is a pre-release convenience and nothing more. The first time a
	// released database matters, this branch must be replaced by a real
	// migration rather than extended.
	if version > 0 && version < schemaVersion {
		if err := dropAllTables(db); err != nil {
			return err
		}
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		return err
	}
	// A crash between dropAllTables and here is safe: user_version lives in the
	// database header, not in a table, so it survives the drops untouched and is
	// only stamped to schemaVersion after schemaSQL has recreated everything.
	// The next Open then sees the same stale, non-zero version again, drops
	// whatever partial shape remains (every drop is IF EXISTS), and reseeds —
	// the reset is self-healing at any interleaving.
	if _, err := db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, schemaVersion)); err != nil {
		return err
	}

	var seeded int
	if err := db.QueryRow(`SELECT COUNT(*) FROM settings WHERE id = 1`).Scan(&seeded); err != nil {
		return err
	}
	if seeded > 0 {
		return nil
	}
	return seed(db)
}

// dropAllTables empties the database. The tables are dropped children-first so
// no foreign key is left dangling mid-drop, and every drop is IF EXISTS
// because the shape being dropped is an older one that may be missing some of
// them.
func dropAllTables(db *sql.DB) error {
	for _, table := range []string{
		"document_iesire_rows",
		"document_intrare_rows",
		"documents",
		"products",
		"templates",
		"settings",
	} {
		if _, err := db.Exec(`DROP TABLE IF EXISTS ` + table); err != nil {
			return fmt.Errorf("stergere tabel %s: %w", table, err)
		}
	}
	return nil
}

// seed writes the first-run contents: the settings row, one template holding
// the 19 products printed on the paper form, and the proces verbal those
// products were first filled in on. All of it in one transaction, so a fresh
// install is either fully seeded or not seeded at all.
func seed(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// NR 1 is taken by the seeded document below, so a new document starts at 2.
	if _, err := tx.Exec(
		`INSERT INTO settings (id, unitate_nume, next_nr, cota_tva, gestiune)
		 VALUES (1, ?, ?, ?, ?)`,
		defaultUnitate, seedDocumentNr+1, defaultCotaTVA, defaultGestiune,
	); err != nil {
		return err
	}

	res, err := tx.Exec(`INSERT INTO templates (nume, ordine) VALUES (?, 0)`, defaultTemplateNume)
	if err != nil {
		return err
	}
	templateID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	// The product ids are collected as they are inserted rather than assumed to
	// run 1..19: the seeded document links to them, and that link must hold
	// whatever ids SQLite hands out.
	productIDs := make([]int64, len(seedProducts))
	procente := seedProcente()
	for i, p := range seedProducts {
		res, err := tx.Exec(
			`INSERT INTO products (template_id, denumire, um, pret_cu_tva, procent_din_intrare, ordine)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			templateID, p.Denumire, p.UM, p.PretCuTVA, procente[i], i,
		)
		if err != nil {
			return err
		}
		if productIDs[i], err = res.LastInsertId(); err != nil {
			return err
		}
	}

	if err := seedFirstDocument(tx, templateID, productIDs); err != nil {
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
func seedFirstDocument(tx *sql.Tx, templateID int64, productIDs []int64) error {
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
		`INSERT INTO documents (template_id, nr, data, gestiune, document_referinta, diferenta_tip,
		         diferenta_valoare, incarca_descarca_tip, incarca_descarca_valoare,
		         gestionar, calculator, vizat_compartiment_productie, created_at, updated_at)
		 VALUES (?, ?, ?, ?, '', ?, ?, ?, ?, '', '', '', ?, ?)`,
		templateID, seedDocumentNr, now.Format("2006-01-02"), defaultGestiune, difTip, difVal,
		incTip, incVal, now.UTC().Format(time.RFC3339), now.UTC().Format(time.RFC3339),
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
