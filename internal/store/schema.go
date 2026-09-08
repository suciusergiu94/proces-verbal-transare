package store

import (
	"database/sql"

	"proces-verbal-transare/internal/model"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  unitate_nume TEXT NOT NULL DEFAULT '',
  next_nr INTEGER NOT NULL DEFAULT 1
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
  pret_cu_tva REAL NOT NULL DEFAULT 0
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
  pret_fara_tva REAL NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_intrare_document ON document_intrare_rows(document_id, pozitie);
CREATE INDEX IF NOT EXISTS idx_iesire_document ON document_iesire_rows(document_id, pozitie);
`

// defaultUnitate is the company printed on the paper form.
const defaultUnitate = "S.C. Largiana Carn S.R.L."

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

// migrate creates the schema and seeds first-run data. It is safe to call on
// every startup: seeding happens only when the settings row is absent, so a
// user who deleted every product does not get them back on the next launch.
func migrate(db *sql.DB) error {
	if _, err := db.Exec(schemaSQL); err != nil {
		return err
	}

	// Stamp a schema version marker so a future migration can tell a v1
	// database (this one) apart from whatever comes after it. A fresh or
	// already-stamped database reports a non-zero user_version, so this is a
	// one-time move: future migrations should switch on the current value of
	// PRAGMA user_version rather than re-checking for zero.
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version == 0 {
		if _, err := db.Exec(`PRAGMA user_version = 1`); err != nil {
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

	if _, err := tx.Exec(
		`INSERT INTO settings (id, unitate_nume, next_nr) VALUES (1, ?, 1)`,
		defaultUnitate,
	); err != nil {
		return err
	}
	for i, p := range seedProducts {
		if _, err := tx.Exec(
			`INSERT INTO products (denumire, um, pret_cu_tva, ordine) VALUES (?, ?, ?, ?)`,
			p.Denumire, p.UM, p.PretCuTVA, i,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}
