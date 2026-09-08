# Șabloane pentru tabelul "ce iese" — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the single global "ce iese" product list into any number of named templates (*șabloane*), pick one when creating a document, and make "Salvează procentele noi" write back to that document's template.

**Architecture:** A new `templates` table owns products (`products.template_id`, cascade) and is remembered by documents (`documents.template_id`, set null). The app is pre-release, so this goes into `schemaSQL` directly and the v2/v3/v4 upgrade machinery is deleted; a database stamped below version 5 is dropped and reseeded. Wails bindings swap `ListProducts`/`SaveProducts` for `ListTemplates`/`SaveTemplates` and `NewDocumentDraft` takes a template id. On the frontend, "+ Document nou" gains a template menu, Setări renders one collapsible section per template with a single Salvează, and the logic that would otherwise hide inside those views moves into a new pure module.

**Tech Stack:** Go 1.25, Wails v2, `modernc.org/sqlite` (pure Go), TypeScript + Vite, vitest (node environment, no jsdom).

**Spec:** `docs/superpowers/specs/2026-09-08-sabloane-ce-iese-design.md`

## Global Constraints

- **All user-facing text is Romanian, with diacritics.** Go error strings in this codebase are Romanian *without* diacritics (`"documentul nu a fost gasit"`); frontend text uses full diacritics. Match the file you are editing.
- **Code comments and identifiers are English**, except domain nouns already in Romanian (`denumire`, `cantitate`, `procent`, `iesire`, `intrare`, `gestiune`, `sablon`). Follow the existing files exactly.
- **No new dependencies**, frontend or backend. In particular **do not add jsdom** — vitest runs in node and every test in this plan is over pure logic.
- **Both native deps stay pure Go** so Windows cross-compilation from macOS keeps working.
- Schema version is **5**. Template seeded on a fresh install is named exactly **`Carcasa Porc`** (no diacritics — it is copied onto a document's "ce intră" row, and the existing seed row reads `Carcasa`).
- The **NR counter stays global** in `settings.next_nr`. No task may make numbering per-template.
- TVA rate, unitate and gestiune implicită stay **global** in `settings`.
- Run `go test ./...` from the repo root and `npm test` from `frontend/` — both must pass before every commit.

---

## File Structure

**Backend**

| File | Responsibility |
|---|---|
| `internal/model/model.go` | add `Template`; `Product.TemplateID`; `Document.TemplateID` |
| `internal/store/schema.go` | clean `schemaSQL` at v5, pre-release reset, seed with a template; **delete** the v2/v3/v4 upgrades, `hasColumn`, `backfillProcente`, `backfillGestiune` |
| `internal/store/templates.go` | **new** — `ListTemplates`, `GetTemplate`, `SaveTemplates`, `listProducts` |
| `internal/store/store.go` | **delete** `ListProducts` / `SaveProducts` (superseded by `templates.go`) |
| `internal/store/documents.go` | read/write `template_id`; `LastDocument(templateID)` |
| `internal/store/migrate_test.go` | **delete** — every test in it covers a deleted upgrade path |
| `internal/store/templates_test.go` | **new** |
| `internal/store/store_test.go` | seed tests updated for templates |
| `app.go` | `ListTemplates`, `SaveTemplates`, `NewDocumentDraft(templateID)` |
| `app_test.go` | draft tests per template |

**Frontend**

| File | Responsibility |
|---|---|
| `frontend/src/templates.ts` | **new** — all pure template logic: hashes, empty/duplicate, validation, ratio write-back |
| `frontend/src/templates.test.ts` | **new** |
| `frontend/src/api.ts` | re-export the new bindings; `Template` type |
| `frontend/src/sidebar.ts` | template menu on "+ Document nou"; draft entry over the new hash shape |
| `frontend/src/sidebar.test.ts` | `currentHash` over templated draft hashes |
| `frontend/src/views/setari.ts` | one collapsible section per template, one Salvează |
| `frontend/src/views/document.ts` | template-aware ratios and write-back; deleted-template state |
| `frontend/src/main.ts` | route `#/document/new/:id` |
| `frontend/src/style.css` | menu + collapsible section styles |

Tasks 1–4 are backend and land in order; 5 is pure frontend logic with no dependency on 1–4 being finished, but its types come from Task 4's regenerated bindings, so run it after. 6–8 are the views.

---

### Task 1: Clean schema at v5, seeded with one template

**Files:**
- Modify: `internal/model/model.go`
- Modify: `internal/store/schema.go` (rewrite of `schemaSQL`, `migrate`, seed)
- Delete: `internal/store/migrate_test.go`
- Modify: `internal/store/store_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `model.Template{ID int64; Nume string; Ordine int; Products []model.Product}`; `model.Product.TemplateID int64`; `model.Document.TemplateID *int64`; package const `schemaVersion = 5`; package const `defaultTemplateNume = "Carcasa Porc"`.

- [ ] **Step 1: Add the model types**

In `internal/model/model.go`, add after `Settings`:

```go
// Template is one named "ce iese" profile: a list of products and the ratios
// they come out of a carcass in. A document is created from a template and
// remembers it, so the ratios it yields can be written back to the right one.
type Template struct {
	ID     int64  `json:"id"`
	Nume   string `json:"nume"`
	Ordine int    `json:"ordine"`
	// Products is the template's "ce iese" list, in display order. It is
	// carried on the template rather than fetched separately because the two
	// are always read and written together.
	Products []Product `json:"products"`
}
```

Add `TemplateID` to `Product`, as its first field:

```go
type Product struct {
	ID         int64 `json:"id"`
	TemplateID int64 `json:"templateId"`
	Denumire   string  `json:"denumire"`
	// ... rest unchanged
}
```

Add to `Document`, after `Nr`:

```go
	// TemplateID is the template this document was created from. It is nil for
	// a document whose template has since been deleted: the rows below keep
	// their own denumire, UM and price, so the document stays complete, but it
	// no longer has ratios to refill its quantities from.
	TemplateID *int64 `json:"templateId"`
```

- [ ] **Step 2: Write the failing seed tests**

Replace `TestOpenSeedsProducts` in `internal/store/store_test.go` and add the new ones:

```go
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
```

Add `"database/sql"` to the test file's imports (needed by the pre-release test).

Update `TestOpenSetsUserVersion` to expect `5`.

- [ ] **Step 3: Run the tests to verify they fail**

Run: `go test ./internal/store/ -run 'TestOpenSeeds|TestOpenResets|TestOpenDoesNotReset|TestOpenSetsUserVersion' -v`
Expected: compile failure — `s.ListTemplates undefined`, `doc.TemplateID undefined`. That is the correct failure for this step; `ListTemplates` arrives in Task 2, so this task's tests go green only once both are done. Write Step 4 and Step 5 now, then implement a minimal `ListTemplates` here as described in Step 6.

- [ ] **Step 4: Rewrite `schemaSQL`**

In `internal/store/schema.go`, replace the whole `schemaSQL` const with:

```go
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
```

- [ ] **Step 5: Rewrite `migrate` and delete the upgrade machinery**

Replace `migrate` in `internal/store/schema.go` with:

```go
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
```

Then **delete** from `internal/store/schema.go`:
- the `for _, c := range []struct{ table, column, ddl string }` loop (v2),
- the `hasColumn` function,
- the `hasProcent` / `backfillProcente` block and `backfillProcente` itself,
- the `hasGestiune` / `backfillGestiune` block and `backfillGestiune` itself,
- the old `PRAGMA user_version` stamping block.

Keep `procenteDinCantitati` and `seedProcente` — they are seed code, not migration code, and `seedProcente` is what produces a column summing to exactly 100.

Adjust imports: `errors` is no longer used once the two backfills are gone; `fmt` is now needed.

- [ ] **Step 6: Seed a template, and add a minimal `ListTemplates`**

Extract the seeding body of the old `migrate` into `seed`, with the template inserted first:

```go
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
```

Change `seedFirstDocument`'s signature to `func seedFirstDocument(tx *sql.Tx, templateID int64, productIDs []int64) error` and add `template_id` to its `INSERT INTO documents`:

```go
	res, err := tx.Exec(
		`INSERT INTO documents (template_id, nr, data, gestiune, document_referinta, diferenta_tip,
		         diferenta_valoare, incarca_descarca_tip, incarca_descarca_valoare,
		         gestionar, calculator, vizat_compartiment_productie, created_at, updated_at)
		 VALUES (?, ?, ?, ?, '', ?, ?, ?, ?, '', '', '', ?, ?)`,
		templateID, seedDocumentNr, now.Format("2006-01-02"), defaultGestiune, difTip, difVal,
		incTip, incVal, now.UTC().Format(time.RFC3339), now.UTC().Format(time.RFC3339),
	)
```

Now create `internal/store/templates.go` with just enough for this task's tests to run — the rest arrives in Task 2:

```go
package store

import (
	"fmt"

	"proces-verbal-transare/internal/model"
)

// ListTemplates returns every template in display order, each carrying its own
// "ce iese" products in theirs.
func (s *Store) ListTemplates() ([]model.Template, error) {
	rows, err := s.db.Query(`SELECT id, nume, ordine FROM templates ORDER BY ordine, id`)
	if err != nil {
		return nil, fmt.Errorf("citire sabloane: %w", err)
	}
	defer rows.Close()

	out := []model.Template{}
	for rows.Next() {
		var t model.Template
		if err := rows.Scan(&t.ID, &t.Nume, &t.Ordine); err != nil {
			return nil, fmt.Errorf("citire sablon: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("citire sabloane: %w", err)
	}

	for i := range out {
		if out[i].Products, err = s.listProducts(out[i].ID); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// listProducts returns one template's products in display order.
func (s *Store) listProducts(templateID int64) ([]model.Product, error) {
	rows, err := s.db.Query(
		`SELECT id, template_id, denumire, um, pret_cu_tva, procent_din_intrare, ordine
		 FROM products WHERE template_id = ? ORDER BY ordine, id`, templateID,
	)
	if err != nil {
		return nil, fmt.Errorf("citire produse: %w", err)
	}
	defer rows.Close()

	out := []model.Product{}
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.TemplateID, &p.Denumire, &p.UM,
			&p.PretCuTVA, &p.ProcentDinIntrare, &p.Ordine); err != nil {
			return nil, fmt.Errorf("citire produs: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
```

Delete `ListProducts` and `SaveProducts` from `internal/store/store.go`.

- [ ] **Step 7: Delete `migrate_test.go` and fix the fallout**

```bash
rm internal/store/migrate_test.go
```

Every test in it (`TestOpenUpgradesV1DatabaseWithDefaultTvaRate`, `TestOpenIsIdempotentOnAnUpgradedDatabase`, `TestOpenBackfillsProcentDinIntrareFromTheOldestDocument`, `TestOpenDoesNotOverwriteProcentDinIntrareOnLaterStarts`, `TestOpenBackfillsTheDefaultGestiuneFromTheNewestDocument`, `TestOpenLeavesTheDefaultGestiuneEmptyWithNothingToCarryOver`) covers a code path deleted in Step 5. They are not replaced.

In `internal/store/store_test.go`, the tests that call the removed `ListProducts` / `SaveProducts` — `TestOpenSeedsProducts`, `TestSaveProductsInsertsUpdatesDeletesAndReorders`, `TestOpenSeedsFirstDocumentLinkedToProducts`, `TestOpenSeedsProductPercentages`, `TestSaveProductsRoundTripsProcentDinIntrare` — must be updated. For this task, change each to read products via `ListTemplates()[0].Products`. The two `SaveProducts` tests move to `templates_test.go` in Task 2; delete them here and re-add them there.

- [ ] **Step 8: Run the full backend suite**

Run: `go test ./...`
Expected: PASS. `app.go` still calls `a.store.ListProducts`, so if the build fails there, add a temporary shim only if needed — otherwise proceed to Task 4, which replaces the call properly. If `go build` blocks the test run, do Task 4's `app.go` edits now rather than adding a shim.

- [ ] **Step 9: Commit**

```bash
git add internal/model/model.go internal/store/ app.go
git commit -m "feat(store): one clean schema at v5, seeded with a template

The single product list becomes a template's product list. The app is
pre-release, so this replaces the schema rather than migrating onto it,
and the v2/v3/v4 upgrade machinery goes with it — a database stamped
below 5 is dropped and reseeded.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014ngEUTuc3QwZza4FWtUudT"
```

---

### Task 2: `GetTemplate` and `SaveTemplates`

**Files:**
- Modify: `internal/store/templates.go`
- Create: `internal/store/templates_test.go`

**Interfaces:**
- Consumes: `model.Template`, `Store.ListTemplates`, `Store.listProducts` (Task 1).
- Produces: `func (s *Store) GetTemplate(id int64) (model.Template, error)` (returns `ErrTemplateNotFound`); `func (s *Store) SaveTemplates(in []model.Template) error`; `var ErrTemplateNotFound = errors.New("sablonul nu a fost gasit")`.

- [ ] **Step 1: Write the failing tests**

Create `internal/store/templates_test.go`:

```go
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
	if len(templates) != 1 {
		t.Fatalf("len(templates) = %d, want 1", len(templates))
	}
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
	if len(stored) != 2 {
		t.Fatalf("len(templates) = %d, want 2", len(stored))
	}
	vitel := stored[1]
	if vitel.Nume != "Carcasa Vitel" {
		t.Errorf("nume = %q, want %q", vitel.Nume, "Carcasa Vitel")
	}
	if vitel.Ordine != 1 {
		t.Errorf("ordine = %d, want 1 (its position in the saved slice)", vitel.Ordine)
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
	// Swap the two and rename the first.
	stored[0].Nume = "Carcasa Porc mare"
	swapped := []model.Template{stored[1], stored[0]}
	if err := s.SaveTemplates(swapped); err != nil {
		t.Fatalf("SaveTemplates: %v", err)
	}

	after, err := s.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if after[0].Nume != "Carcasa Vitel" {
		t.Errorf("after[0] = %q, want %q", after[0].Nume, "Carcasa Vitel")
	}
	if after[1].Nume != "Carcasa Porc mare" {
		t.Errorf("after[1] = %q, want %q", after[1].Nume, "Carcasa Porc mare")
	}
	if after[1].ID != stored[0].ID {
		t.Errorf("the renamed template got a new id (%d, was %d) — it was replaced, not updated",
			after[1].ID, stored[0].ID)
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
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/store/ -run 'TestGetTemplate|TestSaveTemplates' -v`
Expected: FAIL — `s.GetTemplate undefined`, `s.SaveTemplates undefined`, `ErrTemplateNotFound undefined`.

- [ ] **Step 3: Implement `GetTemplate` and `SaveTemplates`**

Append to `internal/store/templates.go` (and add `"database/sql"`, `"errors"`, `"strings"` to its imports):

```go
// ErrTemplateNotFound is returned when a template id does not exist.
var ErrTemplateNotFound = errors.New("sablonul nu a fost gasit")

// GetTemplate reads one template with its products.
func (s *Store) GetTemplate(id int64) (model.Template, error) {
	var t model.Template
	err := s.db.QueryRow(`SELECT id, nume, ordine FROM templates WHERE id = ?`, id).
		Scan(&t.ID, &t.Nume, &t.Ordine)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Template{}, ErrTemplateNotFound
	}
	if err != nil {
		return model.Template{}, fmt.Errorf("citire sablon: %w", err)
	}
	if t.Products, err = s.listProducts(t.ID); err != nil {
		return model.Template{}, err
	}
	return t, nil
}

// SaveTemplates makes the stored templates match in, products included: rows
// with ID 0 are inserted, known IDs are updated, absent IDs are deleted, and
// ordine is reassigned from the slice order at both levels.
//
// Deleting a template takes its products with it through the foreign key, and
// leaves its documents standing with a null template_id: a stored document
// carries its own denumire, UM and price on every row, so it remains complete
// and printable without the template it came from.
func (s *Store) SaveTemplates(in []model.Template) error {
	// There is nowhere to create a document from without at least one template,
	// and no screen from which to make one — Setari edits templates, it does
	// not bootstrap them. The frontend hides the delete button on the last
	// remaining template; this is the backstop behind it.
	if len(in) == 0 {
		return errors.New("trebuie sa existe cel putin un sablon")
	}
	for _, t := range in {
		if strings.TrimSpace(t.Nume) == "" {
			return errors.New("fiecare sablon trebuie sa aiba un nume")
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("salvare sabloane: %w", err)
	}
	defer tx.Rollback()

	keep := make(map[int64]bool, len(in))
	for i, t := range in {
		id := t.ID
		if id == 0 {
			res, err := tx.Exec(`INSERT INTO templates (nume, ordine) VALUES (?, ?)`, t.Nume, i)
			if err != nil {
				return fmt.Errorf("adaugare sablon %q: %w", t.Nume, err)
			}
			if id, err = res.LastInsertId(); err != nil {
				return fmt.Errorf("adaugare sablon %q: %w", t.Nume, err)
			}
		} else if _, err := tx.Exec(
			`UPDATE templates SET nume = ?, ordine = ? WHERE id = ?`, t.Nume, i, id,
		); err != nil {
			return fmt.Errorf("actualizare sablon %q: %w", t.Nume, err)
		}
		keep[id] = true

		if err := saveTemplateProducts(tx, id, t.Products); err != nil {
			return err
		}
	}

	stale, err := staleIDs(tx, `SELECT id FROM templates`, keep)
	if err != nil {
		return fmt.Errorf("salvare sabloane: %w", err)
	}
	for _, id := range stale {
		if _, err := tx.Exec(`DELETE FROM templates WHERE id = ?`, id); err != nil {
			return fmt.Errorf("stergere sablon: %w", err)
		}
	}
	return tx.Commit()
}

// saveTemplateProducts diffs one template's product list the way SaveTemplates
// diffs the templates themselves. template_id is taken from the owning
// template rather than from the incoming row, so a product cannot be moved
// into a template it was not sent under.
func saveTemplateProducts(tx *sql.Tx, templateID int64, products []model.Product) error {
	keep := make(map[int64]bool, len(products))
	for i, p := range products {
		if p.ID == 0 {
			res, err := tx.Exec(
				`INSERT INTO products (template_id, denumire, um, pret_cu_tva, procent_din_intrare, ordine)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				templateID, p.Denumire, p.UM, p.PretCuTVA, p.ProcentDinIntrare, i,
			)
			if err != nil {
				return fmt.Errorf("adaugare produs %q: %w", p.Denumire, err)
			}
			id, err := res.LastInsertId()
			if err != nil {
				return fmt.Errorf("adaugare produs %q: %w", p.Denumire, err)
			}
			keep[id] = true
			continue
		}
		if _, err := tx.Exec(
			`UPDATE products SET template_id = ?, denumire = ?, um = ?, pret_cu_tva = ?,
			        procent_din_intrare = ?, ordine = ? WHERE id = ?`,
			templateID, p.Denumire, p.UM, p.PretCuTVA, p.ProcentDinIntrare, i, p.ID,
		); err != nil {
			return fmt.Errorf("actualizare produs %q: %w", p.Denumire, err)
		}
		keep[p.ID] = true
	}

	stale, err := staleIDs(tx, `SELECT id FROM products WHERE template_id = `+fmt.Sprint(templateID), keep)
	if err != nil {
		return fmt.Errorf("salvare produse: %w", err)
	}
	for _, id := range stale {
		if _, err := tx.Exec(`DELETE FROM products WHERE id = ?`, id); err != nil {
			return fmt.Errorf("stergere produs: %w", err)
		}
	}
	return nil
}

// staleIDs runs a query returning one id column and reports those absent from
// keep — the rows a save has dropped.
func staleIDs(tx *sql.Tx, query string, keep map[int64]bool) ([]int64, error) {
	rows, err := tx.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stale []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if !keep[id] {
			stale = append(stale, id)
		}
	}
	return stale, rows.Err()
}
```

Note on `staleIDs`: the `products` query interpolates `templateID` because it is an `int64` this code produced, never user input, and `tx.Query` with an argument would need the argument threaded through a helper shared with the argument-free templates query. If you prefer, give `staleIDs` a `args ...any` parameter and pass `templateID` properly — either is fine, but do not interpolate anything that came from the frontend.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/store/ -v`
Expected: PASS, including the seed tests from Task 1.

- [ ] **Step 5: Re-add the product-diff tests dropped in Task 1**

The two tests deleted from `store_test.go` in Task 1 Step 7 are now covered by `TestSaveTemplatesInsertsUpdatesDeletesAndReordersProducts`. Confirm nothing else references `SaveProducts`:

Run: `grep -rn "SaveProducts\|ListProducts" --include='*.go' .`
Expected: only `app.go` (fixed in Task 4). If `internal/store` still has a hit, remove it.

- [ ] **Step 6: Commit**

```bash
git add internal/store/
git commit -m "feat(store): read and write templates with their products

SaveTemplates diffs templates and their product lists in one
transaction, the way SaveProducts diffed the single list.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014ngEUTuc3QwZza4FWtUudT"
```

---

### Task 3: Documents carry their template

**Files:**
- Modify: `internal/store/documents.go`
- Modify: `internal/store/documents_test.go`

**Interfaces:**
- Consumes: `model.Document.TemplateID` (Task 1), `ErrTemplateNotFound` (Task 2).
- Produces: `func (s *Store) LastDocument(templateID int64) (model.Document, bool, error)`; `GetDocument` and `SaveDocument` round-trip `TemplateID`.

- [ ] **Step 1: Write the failing tests**

Append to `internal/store/documents_test.go`:

```go
func TestSaveDocumentRoundTripsTheTemplate(t *testing.T) {
	s := newEmptyTestStore(t)
	template := seededTemplate(t, s)

	saved, err := s.SaveDocument(model.Document{
		TemplateID: &template.ID,
		Nr:         7,
		Data:       "2026-09-08",
		Gestiune:   "Magazin Bradet",
	})
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	if saved.TemplateID == nil || *saved.TemplateID != template.ID {
		t.Fatalf("saved.TemplateID = %v, want %d", saved.TemplateID, template.ID)
	}

	reread, err := s.GetDocument(saved.ID)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if reread.TemplateID == nil || *reread.TemplateID != template.ID {
		t.Errorf("reread.TemplateID = %v, want %d", reread.TemplateID, template.ID)
	}
}

func TestSaveDocumentKeepsTheTemplateOnUpdate(t *testing.T) {
	s := newEmptyTestStore(t)
	template := seededTemplate(t, s)

	saved, err := s.SaveDocument(model.Document{
		TemplateID: &template.ID, Nr: 7, Data: "2026-09-08", Gestiune: "Magazin Bradet",
	})
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	saved.Gestiune = "Altundeva"
	updated, err := s.SaveDocument(saved)
	if err != nil {
		t.Fatalf("SaveDocument (update): %v", err)
	}
	if updated.TemplateID == nil || *updated.TemplateID != template.ID {
		t.Errorf("updated.TemplateID = %v, want %d — the update dropped it", updated.TemplateID, template.ID)
	}
}

// The "ce intra" prefill on a new document must come from a butchering of the
// same kind. A pig's carcass row on a calf document would be wrong on the one
// line that names what was cut up.
func TestLastDocumentIsScopedToItsTemplate(t *testing.T) {
	s := newEmptyTestStore(t)
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
	vitel := stored[1]

	if _, err := s.SaveDocument(model.Document{
		TemplateID: &porc.ID, Nr: 1, Data: "2026-09-01", Gestiune: "G",
		Intrare: []model.IntrareRow{{Denumire: "Carcasa Porc", UM: "Kg", Cantitate: 100}},
	}); err != nil {
		t.Fatalf("SaveDocument porc: %v", err)
	}
	if _, err := s.SaveDocument(model.Document{
		TemplateID: &vitel.ID, Nr: 2, Data: "2026-09-02", Gestiune: "G",
		Intrare: []model.IntrareRow{{Denumire: "Carcasa Vitel", UM: "Kg", Cantitate: 200}},
	}); err != nil {
		t.Fatalf("SaveDocument vitel: %v", err)
	}

	last, ok, err := s.LastDocument(porc.ID)
	if err != nil {
		t.Fatalf("LastDocument: %v", err)
	}
	if !ok {
		t.Fatal("LastDocument(porc) found nothing")
	}
	if last.Intrare[0].Denumire != "Carcasa Porc" {
		t.Errorf("intrare[0] = %q, want %q — it picked up the calf document",
			last.Intrare[0].Denumire, "Carcasa Porc")
	}
}

func TestLastDocumentOnATemplateWithNoDocuments(t *testing.T) {
	s := newEmptyTestStore(t)
	template := seededTemplate(t, s)

	_, ok, err := s.LastDocument(template.ID)
	if err != nil {
		t.Fatalf("LastDocument: %v", err)
	}
	if ok {
		t.Error("LastDocument found a document in an empty store")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/store/ -run 'TestSaveDocumentRoundTrips|TestSaveDocumentKeeps|TestLastDocument' -v`
Expected: FAIL — `unknown field TemplateID`, and `LastDocument` takes no argument.

- [ ] **Step 3: Read and write `template_id`**

In `internal/store/documents.go`, `GetDocument`: add `template_id` to the SELECT list and scan it through a `sql.NullInt64`:

```go
	var (
		d         model.Document
		templateID sql.NullInt64
	)
	err := s.db.QueryRow(
		`SELECT id, template_id, nr, data, gestiune, document_referinta, diferenta_tip, diferenta_valoare,
		        incarca_descarca_tip, incarca_descarca_valoare, gestionar, calculator,
		        vizat_compartiment_productie, created_at, updated_at
		 FROM documents WHERE id = ?`, id,
	).Scan(
		&d.ID, &templateID, &d.Nr, &d.Data, &d.Gestiune, &d.DocumentReferinta, &d.DiferentaTip,
		&d.DiferentaValoare, &d.IncarcaDescarcaTip, &d.IncarcaDescarcaValoare, &d.Gestionar,
		&d.Calculator, &d.VizatCompartimentProductie, &d.CreatedAt, &d.UpdatedAt,
	)
```

and after the `ErrNotFound` / error checks:

```go
	if templateID.Valid {
		tid := templateID.Int64
		d.TemplateID = &tid
	}
```

In `SaveDocument`, the insert branch: add `template_id` as the first column and bind it. Use a `any` that is nil when the document carries no template, the same way `product_id` is bound on iesire rows:

```go
	var templateID any
	if d.TemplateID != nil {
		templateID = *d.TemplateID
	}
```

then

```go
		res, err := tx.Exec(
			`INSERT INTO documents (template_id, nr, data, gestiune, document_referinta, diferenta_tip,
			         diferenta_valoare, incarca_descarca_tip, incarca_descarca_valoare, gestionar,
			         calculator, vizat_compartiment_productie, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			templateID, d.Nr, d.Data, d.Gestiune, d.DocumentReferinta, d.DiferentaTip, d.DiferentaValoare,
			d.IncarcaDescarcaTip, d.IncarcaDescarcaValoare, d.Gestionar, d.Calculator,
			d.VizatCompartimentProductie, now, now,
		)
```

and in the update branch, add `template_id = ?` as the first assignment with `templateID` as the first bind. Declare `templateID` once, above the `if d.ID == 0` branch, so both use it.

- [ ] **Step 4: Scope `LastDocument` to a template**

```go
// LastDocument returns the most recently saved document created from the given
// template, if any. It is the source of the prefilled "ce intra" rows on a new
// document, which is why it is scoped: a new calf document must not inherit the
// row that names a pig carcass.
func (s *Store) LastDocument(templateID int64) (model.Document, bool, error) {
	var id int64
	err := s.db.QueryRow(
		`SELECT id FROM documents WHERE template_id = ? ORDER BY id DESC LIMIT 1`, templateID,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Document{}, false, nil
	}
	if err != nil {
		return model.Document{}, false, fmt.Errorf("citire ultimul document: %w", err)
	}
	d, err := s.GetDocument(id)
	if err != nil {
		return model.Document{}, false, err
	}
	return d, true, nil
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/store/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/store/
git commit -m "feat(store): documents remember the template they came from

LastDocument is scoped to a template so a new document's \"ce intra\"
prefill comes from a butchering of the same kind.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014ngEUTuc3QwZza4FWtUudT"
```

---

### Task 4: Wails bindings

**Files:**
- Modify: `app.go`
- Modify: `app_test.go`
- Regenerate: `frontend/wailsjs/go/main/App.js`, `App.d.ts`, `frontend/wailsjs/go/models.ts`

**Interfaces:**
- Consumes: `Store.ListTemplates`, `Store.GetTemplate`, `Store.SaveTemplates`, `Store.LastDocument(templateID)`.
- Produces, bound to the frontend: `ListTemplates(): Promise<model.Template[]>`, `SaveTemplates(t: model.Template[]): Promise<void>`, `NewDocumentDraft(templateId: number): Promise<model.Document>`. `ListProducts` and `SaveProducts` are gone.

- [ ] **Step 1: Write the failing tests**

In `app_test.go`, replace `TestNewDocumentDraftOnFreshInstall` and add:

```go
// seededTemplateID is the one template a fresh app starts with.
func seededTemplateID(t *testing.T, a *App) int64 {
	t.Helper()
	templates, err := a.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("len(templates) = %d, want 1", len(templates))
	}
	return templates[0].ID
}

func TestNewDocumentDraftOnFreshInstall(t *testing.T) {
	a := newTestApp(t)
	id := seededTemplateID(t, a)

	draft, err := a.NewDocumentDraft(id)
	if err != nil {
		t.Fatalf("NewDocumentDraft: %v", err)
	}
	if draft.Nr != 2 {
		t.Errorf("draft.Nr = %d, want 2 (the seeded document took NR 1)", draft.Nr)
	}
	if draft.TemplateID == nil || *draft.TemplateID != id {
		t.Fatalf("draft.TemplateID = %v, want %d", draft.TemplateID, id)
	}
	if len(draft.Iesire) == 0 {
		t.Fatal("draft has no iesire rows")
	}
	if draft.Iesire[0].Denumire != "Pulpa fara os" {
		t.Errorf("iesire[0] = %q, want the template's first product", draft.Iesire[0].Denumire)
	}
}

// A template nobody has butchered under yet has no previous document to copy a
// "ce intra" shape from, so the draft names the row after the template — which
// is what the template is named for.
func TestNewDocumentDraftNamesTheIntrareRowAfterAnUnusedTemplate(t *testing.T) {
	a := newTestApp(t)
	templates, err := a.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	templates = append(templates, model.Template{
		Nume:     "Carcasa Vitel",
		Products: []model.Product{{Denumire: "Antricot", UM: "Kg", PretCuTVA: 45, ProcentDinIntrare: 100}},
	})
	if err := a.SaveTemplates(templates); err != nil {
		t.Fatalf("SaveTemplates: %v", err)
	}
	stored, err := a.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	vitel := stored[1]

	draft, err := a.NewDocumentDraft(vitel.ID)
	if err != nil {
		t.Fatalf("NewDocumentDraft: %v", err)
	}
	if len(draft.Intrare) != 1 {
		t.Fatalf("len(intrare) = %d, want 1", len(draft.Intrare))
	}
	if draft.Intrare[0].Denumire != "Carcasa Vitel" {
		t.Errorf("intrare[0].Denumire = %q, want %q", draft.Intrare[0].Denumire, "Carcasa Vitel")
	}
	if len(draft.Iesire) != 1 || draft.Iesire[0].Denumire != "Antricot" {
		t.Errorf("iesire = %+v, want only the calf template's product", draft.Iesire)
	}
}

func TestNewDocumentDraftOnAnUnknownTemplate(t *testing.T) {
	a := newTestApp(t)

	if _, err := a.NewDocumentDraft(9999); err == nil {
		t.Error("NewDocumentDraft(9999) = nil error, want a refusal")
	}
}

// The register is one register: a document created from any template takes the
// next number, so the numbering stays continuous across kinds of meat.
func TestNrIsSharedAcrossTemplates(t *testing.T) {
	a := newTestApp(t)
	templates, err := a.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	porcID := templates[0].ID
	templates = append(templates, model.Template{
		Nume:     "Carcasa Vitel",
		Products: []model.Product{{Denumire: "Antricot", UM: "Kg", ProcentDinIntrare: 100}},
	})
	if err := a.SaveTemplates(templates); err != nil {
		t.Fatalf("SaveTemplates: %v", err)
	}
	stored, err := a.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	vitelID := stored[1].ID

	first, err := a.NewDocumentDraft(porcID)
	if err != nil {
		t.Fatalf("NewDocumentDraft porc: %v", err)
	}
	first.Data = "2026-09-08"
	saved, err := a.SaveDocument(first)
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	second, err := a.NewDocumentDraft(vitelID)
	if err != nil {
		t.Fatalf("NewDocumentDraft vitel: %v", err)
	}
	if second.Nr != saved.Nr+1 {
		t.Errorf("second draft Nr = %d, want %d — the counter is not shared", second.Nr, saved.Nr+1)
	}
}
```

Update every other `a.NewDocumentDraft()` call in `app_test.go` to `a.NewDocumentDraft(seededTemplateID(t, a))`.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test . -v`
Expected: FAIL — `a.ListTemplates undefined`, `NewDocumentDraft` takes no argument.

- [ ] **Step 3: Replace the bindings in `app.go`**

Delete `ListProducts` and `SaveProducts`. Add:

```go
// ListTemplates returns every "ce iese" template with its products.
func (a *App) ListTemplates() ([]model.Template, error) {
	return a.store.ListTemplates()
}

// SaveTemplates replaces the stored templates with the given ones.
func (a *App) SaveTemplates(templates []model.Template) error {
	return a.store.SaveTemplates(templates)
}
```

Rewrite `NewDocumentDraft`:

```go
// NewDocumentDraft builds an unsaved document from the given template,
// prefilled from the settings and the previous document made from that same
// template.
func (a *App) NewDocumentDraft(templateID int64) (model.Document, error) {
	settings, err := a.store.GetSettings()
	if err != nil {
		return model.Document{}, err
	}
	template, err := a.store.GetTemplate(templateID)
	if err != nil {
		return model.Document{}, err
	}
	last, hasLast, err := a.store.LastDocument(templateID)
	if err != nil {
		return model.Document{}, err
	}

	// Named tid, not id: the product loop below binds its own id, and a
	// shadowed pointer target there would be a very quiet bug.
	tid := template.ID
	draft := model.Document{
		TemplateID: &tid,
		Nr:         settings.NextNr,
		Data:       time.Now().Format("2006-01-02"),
		Intrare:    []model.IntrareRow{},
		Iesire:     []model.IesireRow{},
	}

	// The gestiune comes from the settings rather than from the previous
	// document: a default that only ever repeated the last document would make
	// a one-off change to one document stick to every document after it, which
	// is the thing having a default is meant to prevent. The form still lets
	// this document depart from it.
	draft.Gestiune = settings.Gestiune

	if hasLast {
		for i, r := range last.Intrare {
			// Only the shape of the delivery travels: the same lines, named
			// and measured in the same units, taxed at the rate the user
			// deliberately gave them. A line taxed at something other than
			// the default keeps that rate rather than being reset to it.
			//
			// Neither quantity nor price travels. Both are established afresh
			// for every delivery — the carcass is weighed and the price is
			// what was paid this time — so a value left over from the previous
			// document is wrong by definition. The quantity matters most:
			// the "ce iese" split is derived from it, so a field left
			// unnoticed would quietly fill the whole second table with the
			// previous carcass's numbers.
			draft.Intrare = append(draft.Intrare, model.IntrareRow{
				Pozitie:  i,
				Denumire: r.Denumire,
				UM:       r.UM,
				CotaTVA:  r.CotaTVA,
			})
		}
	}
	if len(draft.Intrare) == 0 {
		// Nothing has been butchered under this template yet, so there is no
		// shape to carry forward. The template's own name is the best answer
		// for what went in — it is named for exactly that.
		draft.Intrare = append(draft.Intrare, model.IntrareRow{
			Pozitie:  0,
			Denumire: template.Nume,
			UM:       "Kg",
			CotaTVA:  settings.CotaTVA,
		})
	}

	// Products carry only a price with TVA, so the "fara TVA" column of a fresh
	// document is derived from it at the default rate. That makes a new
	// document arrive fully priced instead of with a column of zeros; the user
	// can still override either price, or the row's rate, afterwards. An
	// unusable rate leaves the price at zero rather than storing an infinity.
	for i, p := range template.Products {
		id := p.ID
		pretFaraTVA, _ := calc.PretFaraTVA(p.PretCuTVA, settings.CotaTVA)
		draft.Iesire = append(draft.Iesire, model.IesireRow{
			ProductID:   &id,
			Pozitie:     i,
			Denumire:    p.Denumire,
			UM:          p.UM,
			PretCuTVA:   p.PretCuTVA,
			PretFaraTVA: pretFaraTVA,
			CotaTVA:     settings.CotaTVA,
		})
	}
	return draft, nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 5: Regenerate the TypeScript bindings**

Run: `wails generate module`

If the `wails` CLI is missing, install it: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`.

Verify by inspection that `frontend/wailsjs/go/main/App.d.ts` now declares:

```ts
export function ListTemplates():Promise<Array<model.Template>>;
export function NewDocumentDraft(arg1:number):Promise<model.Document>;
export function SaveTemplates(arg1:Array<model.Template>):Promise<void>;
```

and no longer declares `ListProducts` or `SaveProducts`; and that `frontend/wailsjs/go/models.ts` has a `Template` class with `products`, `Product` with `templateId`, and `Document` with `templateId?: number`.

- [ ] **Step 6: Commit**

```bash
git add app.go app_test.go frontend/wailsjs/
git commit -m "feat(app): bind templates, take a template on a new draft

NewDocumentDraft names the \"ce intra\" row after the template when the
template has no documents yet. The NR counter stays global.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014ngEUTuc3QwZza4FWtUudT"
```

---

### Task 5: Pure template logic on the frontend

**Files:**
- Create: `frontend/src/templates.ts`
- Create: `frontend/src/templates.test.ts`
- Modify: `frontend/src/api.ts`

**Interfaces:**
- Consumes: `model.Template` / `model.Product` from Task 4's regenerated bindings; `round3`, `sumaProcente`, `procenteDinCantitati` from `./calc`; `formatNumber` from `./format`.
- Produces: everything the views in Tasks 6–8 build on —
  `DRAFT_PREFIX: string`, `draftHash(templateId: number): string`,
  `isDraftHash(hash: string): boolean`,
  `draftTemplateId(hash: string): number | undefined`,
  `emptyTemplate(): Template`, `duplicateTemplate(t: Template): Template`,
  `type TemplateProblem = { templateIndex: number; message: string }`,
  `validateTemplates(templates: Template[]): TemplateProblem | undefined`,
  `templatesCuProcenteNoi(templates: Template[], templateId: number, cantitatiPerProdus: Map<number, number>): Template[] | undefined`.

- [ ] **Step 1: Export the new bindings from `api.ts`**

In `frontend/src/api.ts`, swap `ListProducts` / `SaveProducts` for `ListTemplates` / `SaveTemplates` in both the import and the re-export, and add the type:

```ts
export type Template = model.Template;
```

- [ ] **Step 2: Write the failing tests**

Create `frontend/src/templates.test.ts`:

```ts
import { describe, expect, it } from 'vitest';
import type { Product, Template } from './api';
import {
  DRAFT_PREFIX,
  draftHash,
  draftTemplateId,
  duplicateTemplate,
  emptyTemplate,
  isDraftHash,
  templatesCuProcenteNoi,
  validateTemplates,
} from './templates';

function product(over: Partial<Product> = {}): Product {
  return {
    id: 0,
    templateId: 0,
    denumire: 'Produs',
    um: 'Kg',
    pretCuTva: 10,
    procentDinIntrare: 100,
    ordine: 0,
    ...over,
  } as Product;
}

function template(over: Partial<Template> = {}): Template {
  return { id: 1, nume: 'Carcasa Porc', ordine: 0, products: [product()], ...over } as Template;
}

describe('draft hashes', () => {
  it('round-trips a template id', () => {
    expect(draftHash(7)).toBe('#/document/new/7');
    expect(draftTemplateId(draftHash(7))).toBe(7);
  });

  it('recognises both the bare and the templated draft route', () => {
    expect(isDraftHash(DRAFT_PREFIX)).toBe(true);
    expect(isDraftHash('#/document/new/7')).toBe(true);
  });

  it('does not mistake a saved document or another page for a draft', () => {
    expect(isDraftHash('#/document/12')).toBe(false);
    expect(isDraftHash('#/setari')).toBe(false);
    expect(isDraftHash('')).toBe(false);
  });

  it('has no template id on the bare draft route', () => {
    expect(draftTemplateId(DRAFT_PREFIX)).toBeUndefined();
    expect(draftTemplateId('#/document/12')).toBeUndefined();
  });
});

describe('emptyTemplate', () => {
  it('is an unsaved template with nothing in it', () => {
    const t = emptyTemplate();
    expect(t.id).toBe(0);
    expect(t.nume).toBe('');
    expect(t.products).toEqual([]);
  });
});

describe('duplicateTemplate', () => {
  it('marks the copy and every product as unsaved', () => {
    const source = template({ id: 3, products: [product({ id: 9, denumire: 'Pulpa' })] });

    const copy = duplicateTemplate(source);

    expect(copy.id).toBe(0);
    expect(copy.nume).toBe('Carcasa Porc (copie)');
    expect(copy.products).toHaveLength(1);
    expect(copy.products[0].id).toBe(0);
    expect(copy.products[0].denumire).toBe('Pulpa');
  });

  it('leaves the source untouched', () => {
    const source = template({ id: 3, products: [product({ id: 9 })] });

    duplicateTemplate(source);

    expect(source.id).toBe(3);
    expect(source.nume).toBe('Carcasa Porc');
    expect(source.products[0].id).toBe(9);
  });
});

describe('validateTemplates', () => {
  it('accepts a template whose ratios account for the whole carcass', () => {
    expect(validateTemplates([template()])).toBeUndefined();
  });

  it('rejects a blank name, by position', () => {
    const problem = validateTemplates([template(), template({ id: 2, nume: '  ' })]);
    expect(problem?.templateIndex).toBe(1);
    expect(problem?.message).toContain('nume');
  });

  it('rejects a template with no products', () => {
    const problem = validateTemplates([template({ products: [] })]);
    expect(problem?.templateIndex).toBe(0);
    expect(problem?.message).toContain('Carcasa Porc');
  });

  it('rejects a blank product denumire', () => {
    const problem = validateTemplates([template({ products: [product({ denumire: ' ' })] })]);
    expect(problem?.templateIndex).toBe(0);
    expect(problem?.message).toContain('denumire');
  });

  it('rejects a negative ratio', () => {
    const problem = validateTemplates([
      template({ products: [product({ procentDinIntrare: -5 }), product({ procentDinIntrare: 105 })] }),
    ]);
    expect(problem?.templateIndex).toBe(0);
    expect(problem?.message).toContain('negativ');
  });

  it('names the template and the shortfall when the ratios do not reach 100', () => {
    const problem = validateTemplates([template({ products: [product({ procentDinIntrare: 97.5 })] })]);
    expect(problem?.templateIndex).toBe(0);
    expect(problem?.message).toContain('Carcasa Porc');
    expect(problem?.message).toContain('97,5');
    expect(problem?.message).toContain('2,5');
  });

  it('says what to subtract when the ratios overshoot 100', () => {
    const problem = validateTemplates([template({ products: [product({ procentDinIntrare: 101 })] })]);
    expect(problem?.message).toContain('scăzute');
    expect(problem?.message).toContain('1');
  });

  it('reports the first bad template, not the last', () => {
    const problem = validateTemplates([
      template({ id: 1, nume: 'Unu', products: [product({ procentDinIntrare: 50 })] }),
      template({ id: 2, nume: 'Doi', products: [product({ procentDinIntrare: 50 })] }),
    ]);
    expect(problem?.templateIndex).toBe(0);
    expect(problem?.message).toContain('Unu');
  });
});

describe('templatesCuProcenteNoi', () => {
  const porc = template({
    id: 1,
    nume: 'Carcasa Porc',
    products: [
      product({ id: 10, denumire: 'Pulpa', procentDinIntrare: 60 }),
      product({ id: 11, denumire: 'Deseu', procentDinIntrare: 40 }),
    ],
  });
  const vitel = template({
    id: 2,
    nume: 'Carcasa Vitel',
    products: [product({ id: 20, denumire: 'Antricot', procentDinIntrare: 100 })],
  });

  it('rewrites only the named template', () => {
    const out = templatesCuProcenteNoi([porc, vitel], 1, new Map([[10, 30], [11, 70]]));

    expect(out).toBeDefined();
    expect(out![0].products.map((p) => p.procentDinIntrare)).toEqual([30, 70]);
    expect(out![1]).toEqual(vitel);
  });

  it('zeroes a product this document did not yield', () => {
    const out = templatesCuProcenteNoi([porc], 1, new Map([[10, 100]]));

    expect(out![0].products[1].procentDinIntrare).toBe(0);
    expect(out![0].products[0].procentDinIntrare).toBe(100);
  });

  it('gives back nothing when there are no quantities to work from', () => {
    expect(templatesCuProcenteNoi([porc], 1, new Map([[10, 0], [11, 0]]))).toBeUndefined();
    expect(templatesCuProcenteNoi([porc], 1, new Map())).toBeUndefined();
  });

  it('gives back nothing for a template that is not in the list', () => {
    expect(templatesCuProcenteNoi([porc], 99, new Map([[10, 5]]))).toBeUndefined();
  });

  it('does not mutate its input', () => {
    templatesCuProcenteNoi([porc, vitel], 1, new Map([[10, 30], [11, 70]]));
    expect(porc.products[0].procentDinIntrare).toBe(60);
  });
});
```

- [ ] **Step 3: Run the tests to verify they fail**

Run: `cd frontend && npm test -- templates`
Expected: FAIL — `Cannot find module './templates'`.

- [ ] **Step 4: Implement `templates.ts`**

Create `frontend/src/templates.ts`:

```ts
import type { Template } from './api';
import { procenteDinCantitati, round3, sumaProcente } from './calc';
import { formatNumber } from './format';

/**
 * Pure logic behind the template screens. It lives outside the views so it can
 * be tested: vitest runs in node here, with no DOM, and the views are wiring.
 */

/** The route an unsaved document lives at, before its template is named. */
export const DRAFT_PREFIX = '#/document/new';

/** The route of an unsaved document created from a given template. */
export function draftHash(templateId: number): string {
  return `${DRAFT_PREFIX}/${templateId}`;
}

/** Whether a hash is the draft route, with or without a template on it. */
export function isDraftHash(hash: string): boolean {
  return hash === DRAFT_PREFIX || hash.startsWith(`${DRAFT_PREFIX}/`);
}

/** The template a draft route names, if it names one. */
export function draftTemplateId(hash: string): number | undefined {
  const match = /^#\/document\/new\/(\d+)$/.exec(hash);
  return match === null ? undefined : Number(match[1]);
}

/** A blank, unsaved template — what "+ Șablon nou" appends. */
export function emptyTemplate(): Template {
  return { id: 0, nume: '', ordine: 0, products: [] } as Template;
}

/**
 * A copy of `t` that has never been stored: the template and every one of its
 * products carry id 0, so saving inserts them rather than moving the originals.
 * This is how a second profile that mostly resembles the first gets started.
 */
export function duplicateTemplate(t: Template): Template {
  return {
    id: 0,
    nume: `${t.nume} (copie)`,
    ordine: 0,
    products: t.products.map((p) => ({ ...p, id: 0, templateId: 0 })),
  } as Template;
}

/** Where a problem is and what to tell the user about it. */
export type TemplateProblem = { templateIndex: number; message: string };

/**
 * The first thing wrong with the template list, or undefined if nothing is.
 *
 * The 100% rule is the substantial one: the form fills the "ce iese"
 * quantities from these ratios, so a column that does not account for the whole
 * carcass would quietly produce a document that is short. It is refused rather
 * than normalised — the figures are the user's to decide, and the gap is named
 * so it can be put where it belongs, usually on deșeu.
 */
export function validateTemplates(templates: Template[]): TemplateProblem | undefined {
  for (const [templateIndex, t] of templates.entries()) {
    const nume = t.nume.trim();
    if (nume === '') {
      return { templateIndex, message: `Șablonul ${templateIndex + 1} nu are nume.` };
    }
    if (t.products.length === 0) {
      return { templateIndex, message: `Șablonul „${nume}” nu are niciun produs.` };
    }
    if (t.products.some((p) => p.denumire.trim() === '')) {
      return {
        templateIndex,
        message: `Șablonul „${nume}”: fiecare produs trebuie să aibă o denumire.`,
      };
    }
    if (t.products.some((p) => p.procentDinIntrare < 0)) {
      return {
        templateIndex,
        message: `Șablonul „${nume}”: procentul din intrare nu poate fi negativ.`,
      };
    }
    const total = sumaProcente(t.products.map((p) => p.procentDinIntrare));
    if (total !== 100) {
      const lipsa = round3(100 - total);
      return {
        templateIndex,
        message:
          `Șablonul „${nume}”: procentele din intrare însumează ` +
          `${formatNumber(total, 3)} %, nu 100 %.\n` +
          (lipsa > 0
            ? `Mai trebuie repartizate ${formatNumber(lipsa, 3)} % (de obicei la deșeu).`
            : `Trebuie scăzute ${formatNumber(-lipsa, 3)} %.`),
      };
    }
  }
  return undefined;
}

/**
 * The template list with one template's ratios replaced by the split a document
 * actually yielded. Every other template comes back untouched.
 *
 * The ratios are worked out per product rather than per row: a row whose
 * product has since been deleted cannot carry a share, and must not take one
 * with it and leave the column short of the 100% the form insists on. A product
 * this document did not yield at all goes to zero for the same reason — the
 * column being written is the whole of it, not a patch over what is stored.
 *
 * Returns undefined when the template is not in the list, or when the document
 * yielded nothing to derive a split from.
 */
export function templatesCuProcenteNoi(
  templates: Template[],
  templateId: number,
  cantitatiPerProdus: Map<number, number>,
): Template[] | undefined {
  const target = templates.find((t) => t.id === templateId);
  if (target === undefined) return undefined;

  const noi = procenteDinCantitati(target.products.map((p) => cantitatiPerProdus.get(p.id) ?? 0));
  if (noi === undefined) return undefined;

  return templates.map((t) =>
    t.id === templateId
      ? { ...t, products: t.products.map((p, i) => ({ ...p, procentDinIntrare: noi[i] })) }
      : t,
  );
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd frontend && npm test`
Expected: PASS — the new file plus every existing test.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/templates.ts frontend/src/templates.test.ts frontend/src/api.ts
git commit -m "feat(frontend): pure template logic, with tests

Validation, duplication, the draft routes and the ratio write-back move
out of the views so vitest can reach them without a DOM.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014ngEUTuc3QwZza4FWtUudT"
```

---

### Task 6: The template menu on "+ Document nou"

**Files:**
- Modify: `frontend/src/sidebar.ts`
- Modify: `frontend/src/sidebar.test.ts`
- Modify: `frontend/src/main.ts`
- Modify: `frontend/src/style.css`

**Interfaces:**
- Consumes: `ListTemplates` from `./api`; `DRAFT_PREFIX`, `draftHash`, `isDraftHash`, `draftTemplateId` from `./templates`.
- Produces: `currentHash(hash?)` unchanged in signature; `DRAFT_HASH` is **removed** from `sidebar.ts` in favour of `DRAFT_PREFIX` from `./templates`. Route `#/document/new/(\d+)` renders `renderDocumentView(el, undefined, refreshSidebar, templateId)` — Task 8 changes that signature; until then, keep `renderDocumentView`'s current three-argument shape and pass the id as a fourth argument that it ignores. Do Tasks 6 and 8 back to back to keep that window short.

- [ ] **Step 1: Write the failing test**

In `frontend/src/sidebar.test.ts`, replace the `DRAFT_HASH` import with `DRAFT_PREFIX` from `./templates` and add:

```ts
  it('recognises a templated draft route as a real route', () => {
    expect(currentHash('#/document/new/3')).toBe('#/document/new/3');
  });
```

Keep the existing three tests, with `DRAFT_HASH` replaced by `DRAFT_PREFIX`.

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd frontend && npm test -- sidebar`
Expected: FAIL — `DRAFT_PREFIX` is not exported from `./templates` into `sidebar.test.ts`'s old import, or `DRAFT_HASH` no longer exists.

- [ ] **Step 3: Rewrite the sidebar**

In `frontend/src/sidebar.ts`: import `ListTemplates` and `Template` from `./api`, and `DRAFT_PREFIX, draftHash, isDraftHash` from `./templates`. Delete the local `DRAFT_HASH` const and re-export the prefix for the one caller that still wants a fallback:

```ts
export { DRAFT_PREFIX } from './templates';
```

`currentHash` keeps its shape but falls back to the prefix:

```ts
export function currentHash(hash: string = window.location.hash): string {
  return hash || DRAFT_PREFIX;
}
```

`renderSidebar` loads the templates alongside the documents and renders the menu:

```ts
export async function renderSidebar(el: HTMLElement): Promise<void> {
  let documents: DocumentSummary[];
  let templates: Template[];
  try {
    [documents, templates] = await Promise.all([ListDocuments(), ListTemplates()]);
  } catch (err) {
    showError('Nu s-a putut încărca lista de documente', err);
    documents = [];
    templates = [];
  }

  const items = documents
    .map(
      (doc) => `
        <li>
          <a class="doc-link" href="#/document/${doc.id}">
            <span class="doc-nr">NR ${doc.nr}</span>
            <span class="doc-meta">${escapeHtml(formatDateRO(doc.data))}${doc.gestiune ? ` — ${escapeHtml(doc.gestiune)}` : ''}</span>
          </a>
        </li>`,
    )
    .join('');

  // With one template there is nothing to choose, so the button goes straight
  // to it and no menu is drawn at all — an install that never adds a second
  // template behaves exactly as it did before templates existed.
  const menu =
    templates.length < 2
      ? ''
      : `<ul class="template-menu" id="template-menu" hidden>
          ${templates
            .map(
              (t) => `
            <li>
              <a class="template-link" href="${draftHash(t.id)}">${escapeHtml(t.nume)}</a>
            </li>`,
            )
            .join('')}
        </ul>`;

  el.innerHTML = `
    <div class="new-doc-wrap">
      <button class="btn btn-primary" id="new-doc">+ Document nou</button>
      ${menu}
    </div>
    <ul class="doc-list">
      <li id="draft-item" hidden>
        <a class="doc-link" id="draft-link" href="${DRAFT_PREFIX}">
          <span class="doc-nr">Document nou</span>
          <span class="doc-meta draft-meta">Nesalvat</span>
        </a>
      </li>
      ${items}
    </ul>
    <a class="settings-link" href="#/setari">Setări</a>
  `;

  const menuEl = el.querySelector<HTMLUListElement>('#template-menu');
  el.querySelector<HTMLButtonElement>('#new-doc')!.addEventListener('click', () => {
    if (menuEl === null) {
      // One template, or none loaded: go where there is only one place to go.
      if (templates.length === 1) navigate(draftHash(templates[0].id));
      return;
    }
    menuEl.toggleAttribute('hidden');
  });

  // Picking a template closes the menu; the href does the navigating.
  menuEl?.querySelectorAll('a').forEach((link) => {
    link.addEventListener('click', () => menuEl.setAttribute('hidden', ''));
  });

  // A click anywhere else closes it, so it does not sit open over the history.
  document.addEventListener('click', (event) => {
    if (menuEl === null || menuEl.hasAttribute('hidden')) return;
    if (!el.querySelector('.new-doc-wrap')!.contains(event.target as Node)) {
      menuEl.setAttribute('hidden', '');
    }
  });

  markActive(el, templates);

  if (!hashchangeListenerRegistered) {
    hashchangeListenerRegistered = true;
    window.addEventListener('hashchange', () => markActive(el, templates));
  }
}
```

`markActive` shows the draft entry for any draft hash and labels it with the template:

```ts
function markActive(el: HTMLElement, templates: Template[]): void {
  const hash = currentHash();
  const draftItem = el.querySelector('#draft-item');
  draftItem?.toggleAttribute('hidden', !isDraftHash(hash));

  // The draft entry points at whatever draft is open, and says which template
  // it came from, so two unsaved documents are never confusable.
  const draftLink = el.querySelector<HTMLAnchorElement>('#draft-link');
  if (draftLink !== null && isDraftHash(hash)) {
    draftLink.setAttribute('href', hash);
    const templateId = draftTemplateId(hash);
    const template = templates.find((t) => t.id === templateId);
    const meta = draftLink.querySelector('.draft-meta');
    if (meta !== null) {
      meta.textContent = template === undefined ? 'Nesalvat' : `Nesalvat — ${template.nume}`;
    }
  }

  el.querySelectorAll('a').forEach((link) => {
    link.classList.toggle('active', link.getAttribute('href') === hash);
  });
}
```

Import `draftTemplateId` alongside the others.

- [ ] **Step 4: Add the route**

In `frontend/src/main.ts`, replace the bare draft route with both shapes:

```ts
    {
      // The bare draft route has no template on it — a stale hash, or the
      // empty-hash fallback on a fresh launch. renderDocumentView sends it to
      // the first template rather than failing.
      pattern: /^#\/document\/new$/,
      render: (el) => renderDocumentView(el, undefined, refreshSidebar, undefined),
    },
    {
      pattern: /^#\/document\/new\/(\d+)$/,
      render: (el, templateId) =>
        renderDocumentView(el, undefined, refreshSidebar, Number(templateId)),
    },
    {
      pattern: /^#\/document\/(\d+)$/,
      render: (el, id) => renderDocumentView(el, id, refreshSidebar, undefined),
    },
```

Order matters: `#/document/new` must be matched before `#/document/(\d+)` — it already is, and `new` is not `\d+`, so either order works, but keep the draft routes first for readability.

- [ ] **Step 5: Style the menu**

Add to `frontend/src/style.css`, following the existing sidebar variables and spacing:

```css
/* The template menu hangs under "+ Document nou". It is a plain list rather
   than a <select> so it matches the document list beneath it. */
.new-doc-wrap {
  position: relative;
}

.template-menu {
  position: absolute;
  z-index: 10;
  left: 0;
  right: 0;
  margin: 0.25rem 0 0;
  padding: 0.25rem;
  list-style: none;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 6px;
  box-shadow: 0 6px 16px rgb(0 0 0 / 18%);
}

.template-link {
  display: block;
  padding: 0.4rem 0.55rem;
  border-radius: 4px;
  color: inherit;
  text-decoration: none;
}

.template-link:hover {
  background: var(--hover);
}
```

Check the actual custom property names in `style.css` first (`--panel`, `--border`, `--hover` are placeholders here) and use the ones the file defines. If the file has no such variables, copy the literal colours the sidebar already uses.

- [ ] **Step 6: Run the tests and the type check**

Run: `cd frontend && npm test && npx tsc --noEmit`
Expected: PASS. `renderDocumentView` takes three parameters until Task 8, so `tsc` will flag the fourth argument — that is expected here; either do Task 8 immediately, or add the unused fourth parameter to `renderDocumentView`'s signature now and use it in Task 8.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/sidebar.ts frontend/src/sidebar.test.ts frontend/src/main.ts frontend/src/style.css
git commit -m "feat(frontend): pick a template from + Document nou

The menu is skipped when there is only one template, so a single-profile
install behaves as it did before.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014ngEUTuc3QwZza4FWtUudT"
```

---

### Task 7: Setări with one section per template

**Files:**
- Modify: `frontend/src/views/setari.ts`
- Modify: `frontend/src/style.css`

**Interfaces:**
- Consumes: `ListTemplates`, `SaveTemplates`, `GetSettings`, `SaveSettings` from `./api`; `emptyTemplate`, `duplicateTemplate`, `validateTemplates` from `../templates`; `showConfirm` from `../dialog`.
- Produces: nothing other tasks consume.

- [ ] **Step 1: Fix the imports and restructure the view's state**

`frontend/src/views/setari.ts` currently starts with:

```ts
import { GetSettings, ListProducts, SaveProducts, SaveSettings, showError } from '../api';
import type { Product, Settings } from '../api';
import { round3, sumaProcente } from '../calc';
import { formatNumber, parseNumber } from '../format';
import { escapeHtml } from '../sidebar';
```

Replace that block with:

```ts
import { GetSettings, ListTemplates, SaveSettings, SaveTemplates, showError } from '../api';
import type { Product, Settings, Template } from '../api';
import { sumaProcente } from '../calc';
import { showAlert } from '../dialog';
import { formatNumber, parseNumber } from '../format';
import { escapeHtml } from '../sidebar';
import { duplicateTemplate, emptyTemplate, validateTemplates } from '../templates';
```

`round3` goes: the shortfall arithmetic it served moved into `validateTemplates`.
`sumaProcente` stays — `refreshTotaluri` still needs it for the per-section
totals on screen.

Then replace the `products` variable with:

```ts
  let settings: Settings;
  let templates: Template[];
  // Which sections are open. View state only, never persisted: a shop with
  // five templates should not open Setări on a wall of tables, so only the
  // first is expanded on load.
  let expanded = new Set<number>();
```

and the load:

```ts
  try {
    [settings, templates] = await Promise.all([GetSettings(), ListTemplates()]);
  } catch (err) {
    showError('Nu s-au putut încărca setările', err);
    outlet.innerHTML = '<p class="empty">Setările nu au putut fi încărcate.</p>';
    return;
  }
  if (templates.length > 0) expanded.add(0);
```

- [ ] **Step 2: Render a section per template**

Replace `renderAll`'s single table with a loop.

The `<h1>` and the whole `<div class="header-grid">` block — unitate, gestiune
implicită, următorul NR, cotă TVA implicită — are **copied across unchanged**,
character for character, from the current file. Everything from `<h2>Produse
(tabelul "ce iese")</h2>` down to and including the closing `</p>` of the status
line is replaced by:

```ts
    const sectiuni = templates.map((t, ti) => templateSection(t, ti)).join('');

    outlet.innerHTML = `
      <h1>Setări</h1>

      <div class="header-grid">
        <!-- copied verbatim from the current file: #s-unitate, #s-gestiune,
             #s-nextnr, #s-cota, with their labels and .field wrappers -->
      </div>

      <h2>Șabloane</h2>
      <p class="empty">
        Un șablon este o listă "ce iese" cu numele ei — de exemplu "Carcasa Porc"
        sau "Carcasa Vitel". Procentele spun cât din ce intră se transformă în
        fiecare produs, deci trebuie să însumeze exact 100% în fiecare șablon,
        inclusiv pierderile, trecute la deșeu.
      </p>

      <div id="templates">${sectiuni}</div>
      <div class="table-actions">
        <button class="btn" id="add-template">+ Șablon nou</button>
      </div>

      <div class="btn-row">
        <button class="btn btn-primary" id="save">Salvează</button>
      </div>
      <p class="empty" id="status"></p>
    `;
```

with:

```ts
  /** One template: a header that is always visible and a table that folds. */
  function templateSection(t: Template, ti: number): string {
    const rows = t.products
      .map(
        (p, i) => `
        <tr data-template="${ti}" data-index="${i}">
          <td>${i + 1}</td>
          <td><input data-field="denumire" value="${escapeHtml(p.denumire)}" /></td>
          <td><input data-field="um" value="${escapeHtml(p.um)}" /></td>
          <td class="num"><input class="num" data-field="pretCuTva" value="${formatNumber(p.pretCuTva)}" /></td>
          <td class="num"><input class="num" data-field="procentDinIntrare" value="${formatNumber(p.procentDinIntrare, 3)}" /></td>
          <td>
            <button class="btn-icon" data-up="${ti}:${i}" title="Mută în sus" ${i === 0 ? 'disabled' : ''}>↑</button>
            <button class="btn-icon" data-down="${ti}:${i}" title="Mută în jos" ${i === t.products.length - 1 ? 'disabled' : ''}>↓</button>
            <button class="btn-icon" data-remove="${ti}:${i}" title="Șterge produsul">✕</button>
          </td>
        </tr>`,
      )
      .join('');

    return `
      <section class="template-section${expanded.has(ti) ? ' open' : ''}" data-template="${ti}">
        <div class="template-head">
          <button class="btn-icon template-toggle" data-toggle="${ti}"
                  title="${expanded.has(ti) ? 'Restrânge' : 'Extinde'}">${expanded.has(ti) ? '▾' : '▸'}</button>
          <input class="template-nume" data-template-nume="${ti}"
                 placeholder="Numele șablonului" value="${escapeHtml(t.nume)}" />
          <span class="template-total" data-total="${ti}"></span>
          <button class="btn" data-duplicate="${ti}">Duplică</button>
          ${templates.length > 1 ? `<button class="btn btn-danger" data-delete="${ti}">Șterge</button>` : ''}
        </div>
        <div class="template-body" ${expanded.has(ti) ? '' : 'hidden'}>
          <table>
            <thead>
              <tr>
                <th>Nr. crt.</th><th>Denumire produs</th><th>U/M</th>
                <th class="num">Preț cu TVA</th><th class="num">% din intrare</th><th></th>
              </tr>
            </thead>
            <tbody>${rows}</tbody>
          </table>
          <div class="table-actions">
            <button class="btn" data-add-product="${ti}">+ Adaugă produs</button>
          </div>
        </div>
      </section>`;
  }
```

Note the `data-*` encoding: product buttons carry `"<templateIndex>:<productIndex>"`, parsed with `split(':').map(Number)`. Keep that consistent everywhere.

- [ ] **Step 3: Rewrite `readForm` and the totals**

```ts
  /** Copies every input's current value back into settings and templates. */
  function readForm(): void {
    settings.unitateNume = (outlet.querySelector('#s-unitate') as HTMLInputElement).value;
    settings.gestiune = (outlet.querySelector('#s-gestiune') as HTMLInputElement).value;
    // Number(...) || 1 alone would let a negative value (e.g. -3) through
    // unchanged, since -3 is truthy; only "" / NaN fall back to 1 that way.
    const rawNextNr = Number((outlet.querySelector('#s-nextnr') as HTMLInputElement).value);
    const truncatedNextNr = Math.trunc(rawNextNr);
    settings.nextNr = Number.isFinite(truncatedNextNr) && truncatedNextNr >= 1 ? truncatedNextNr : 1;
    settings.cotaTva = parseNumber((outlet.querySelector('#s-cota') as HTMLInputElement).value);

    outlet.querySelectorAll<HTMLInputElement>('[data-template-nume]').forEach((input) => {
      const t = templates[Number(input.dataset.templateNume)];
      if (t) t.nume = input.value;
    });

    outlet.querySelectorAll<HTMLTableRowElement>('tr[data-template][data-index]').forEach((tr) => {
      const t = templates[Number(tr.dataset.template)];
      const product = t?.products[Number(tr.dataset.index)];
      if (!product) return;
      product.denumire = tr.querySelector<HTMLInputElement>('[data-field="denumire"]')!.value;
      product.um = tr.querySelector<HTMLInputElement>('[data-field="um"]')!.value;
      product.pretCuTva = parseNumber(
        tr.querySelector<HTMLInputElement>('[data-field="pretCuTva"]')!.value,
      );
      product.procentDinIntrare = parseNumber(
        tr.querySelector<HTMLInputElement>('[data-field="procentDinIntrare"]')!.value,
      );
    });
  }

  /**
   * Shows what each template's ratio column accounts for, marked as wrong
   * unless it is exactly 100%. The figures are the ones validateTemplates
   * checks, so the screen can never disagree with the message the user gets
   * when saving is refused.
   */
  function refreshTotaluri(): void {
    templates.forEach((t, ti) => {
      const cell = outlet.querySelector<HTMLElement>(`[data-total="${ti}"]`);
      if (!cell) return;
      const total = sumaProcente(t.products.map((p) => p.procentDinIntrare));
      cell.textContent = `${formatNumber(total, 3)} %`;
      cell.classList.toggle('invalid', total !== 100);
    });
  }
```

- [ ] **Step 4: Rewrite `wireEvents`**

```ts
  function wireEvents(): void {
    // Keeps the totals in step as the user types, without re-rendering the
    // table under their caret.
    outlet.querySelectorAll<HTMLInputElement>('[data-field="procentDinIntrare"]').forEach((input) => {
      input.addEventListener('input', () => {
        readForm();
        refreshTotaluri();
      });
    });

    outlet.querySelectorAll<HTMLButtonElement>('[data-toggle]').forEach((btn) => {
      btn.addEventListener('click', () => {
        readForm();
        const ti = Number(btn.dataset.toggle);
        if (expanded.has(ti)) expanded.delete(ti);
        else expanded.add(ti);
        renderAll();
      });
    });

    outlet.querySelectorAll<HTMLButtonElement>('[data-add-product]').forEach((btn) => {
      btn.addEventListener('click', () => {
        readForm();
        const ti = Number(btn.dataset.addProduct);
        templates[ti].products.push({
          id: 0,
          templateId: templates[ti].id,
          denumire: '',
          um: 'Kg',
          pretCuTva: 0,
          procentDinIntrare: 0,
          ordine: templates[ti].products.length,
        } as Product);
        renderAll();
      });
    });

    outlet.querySelectorAll<HTMLButtonElement>('[data-up]').forEach((btn) => {
      btn.addEventListener('click', () => moveProduct(btn.dataset.up!, -1));
    });
    outlet.querySelectorAll<HTMLButtonElement>('[data-down]').forEach((btn) => {
      btn.addEventListener('click', () => moveProduct(btn.dataset.down!, 1));
    });
    outlet.querySelectorAll<HTMLButtonElement>('[data-remove]').forEach((btn) => {
      btn.addEventListener('click', () => {
        readForm();
        const [ti, i] = btn.dataset.remove!.split(':').map(Number);
        templates[ti].products.splice(i, 1);
        renderAll();
      });
    });

    outlet.querySelectorAll<HTMLButtonElement>('[data-duplicate]').forEach((btn) => {
      btn.addEventListener('click', () => {
        readForm();
        const ti = Number(btn.dataset.duplicate);
        templates.splice(ti + 1, 0, duplicateTemplate(templates[ti]));
        // Every index past the insertion point has shifted, so the open set is
        // rebuilt rather than patched: the copy opens, and the rest follow the
        // sections they were attached to.
        expanded = new Set(
          [...expanded].map((n) => (n > ti ? n + 1 : n)).concat(ti + 1),
        );
        renderAll();
      });
    });

    outlet.querySelectorAll<HTMLButtonElement>('[data-delete]').forEach((btn) => {
      btn.addEventListener('click', () => void onDeleteTemplate(Number(btn.dataset.delete)));
    });

    outlet.querySelector('#add-template')!.addEventListener('click', () => {
      readForm();
      templates.push(emptyTemplate());
      expanded.add(templates.length - 1);
      renderAll();
      // A blank section is useless until it is named, so start there.
      outlet
        .querySelector<HTMLInputElement>(`[data-template-nume="${templates.length - 1}"]`)
        ?.focus();
    });

    outlet.querySelector('#save')!.addEventListener('click', () => void onSave());
  }

  function moveProduct(key: string, delta: number): void {
    const [ti, index] = key.split(':').map(Number);
    const products = templates[ti].products;
    const target = index + delta;
    if (target < 0 || target >= products.length) return;
    readForm();
    const [row] = products.splice(index, 1);
    products.splice(target, 0, row);
    renderAll();
  }

  /**
   * Removes a template from the list being edited. Nothing is written until
   * Salvează, so this is undone by leaving the page — which the confirmation
   * says, because deleting a profile someone spent an afternoon on is not a
   * click to take lightly.
   */
  async function onDeleteTemplate(ti: number): Promise<void> {
    readForm();
    const nume = templates[ti].nume.trim() || `Șablonul ${ti + 1}`;
    if (
      !(await showConfirm(
        `Ștergeți șablonul „${nume}”? Documentele făcute cu el rămân neatinse, ` +
          'dar nu vor mai putea completa cantitățile automat. Ștergerea are loc la Salvează.',
      ))
    ) {
      return;
    }
    templates.splice(ti, 1);
    expanded = new Set([...expanded].filter((n) => n !== ti).map((n) => (n > ti ? n - 1 : n)));
    renderAll();
  }
```

- [ ] **Step 5: Rewrite `onSave`**

```ts
  async function onSave(): Promise<void> {
    readForm();
    if (settings.cotaTva < 0) {
      void showAlert('Cota TVA nu poate fi negativă.');
      return;
    }
    const problem = validateTemplates(templates);
    if (problem !== undefined) {
      // Open the offending section so the user is looking at what is wrong
      // when they read why the save was refused.
      expanded.add(problem.templateIndex);
      renderAll();
      outlet
        .querySelector(`.template-section[data-template="${problem.templateIndex}"]`)
        ?.scrollIntoView({ block: 'center' });
      void showAlert(problem.message);
      return;
    }
    try {
      await SaveSettings(settings);
      await SaveTemplates(templates);
      templates = await ListTemplates();
      renderAll();
      outlet.querySelector('#status')!.textContent = 'Setările au fost salvate.';
    } catch (err) {
      showError('Setările nu au putut fi salvate', err);
    }
  }
```

Import `showAlert` from `../dialog` and drop the `window.alert` calls: `window.alert` is a dead end in the macOS webview this app runs in (see the comment at the top of `dialog.ts`), and the existing `setari.ts` calls are a latent bug this task inherits. Also drop the now-unused `round3` / `sumaProcente` imports if `refreshTotaluri` is the only remaining user of `sumaProcente` — keep that one.

`renderAll` ends with `wireEvents(); refreshTotaluri();` as it does today.

- [ ] **Step 6: Style the sections**

Add to `frontend/src/style.css`:

```css
/* One template per section. The head stays visible so the ratio total of a
   folded template is still readable — it is the figure that decides whether
   the page can be saved at all. */
.template-section {
  margin-bottom: 0.75rem;
  border: 1px solid var(--border);
  border-radius: 6px;
}

.template-head {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  padding: 0.5rem 0.6rem;
}

.template-nume {
  flex: 1;
  font-weight: 600;
}

.template-total {
  min-width: 6rem;
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.template-total.invalid {
  color: var(--danger);
  font-weight: 600;
}

.template-body {
  padding: 0 0.6rem 0.6rem;
}
```

As in Task 6, use the custom property names `style.css` actually defines; `.invalid` already exists for the old single total, so reuse its colour rather than inventing one.

- [ ] **Step 7: Verify**

Run: `cd frontend && npm test && npx tsc --noEmit`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/views/setari.ts frontend/src/style.css
git commit -m "feat(setari): one collapsible section per template

A single Salvează writes the settings and every template; a template
whose ratios miss 100% opens itself and blocks the save.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014ngEUTuc3QwZza4FWtUudT"
```

---

### Task 8: The document view follows its template

**Files:**
- Modify: `frontend/src/views/document.ts`

**Interfaces:**
- Consumes: `ListTemplates`, `SaveTemplates`, `NewDocumentDraft(templateId)` from `./api`; `templatesCuProcenteNoi`, `draftHash` from `../templates`.
- Produces: `renderDocumentView(outlet, id, refreshSidebar, templateId)` — the fourth parameter Task 6 already passes.

- [ ] **Step 1: Fix the imports**

In `frontend/src/views/document.ts`, the first three import blocks become:

```ts
import {
  DeleteDocument,
  ExportPDF,
  GetDocument,
  GetSettings,
  ListTemplates,
  NewDocumentDraft,
  SaveDocument,
  SaveTemplates,
  showError,
} from '../api';
import type { Document, Template } from '../api';
import {
  ajusteazaMarja,
  cantitatiDinProcente,
  diferenta,
  incarcaDescarca,
  marjaProfit,
  pretCuTvaDin,
  pretFaraTvaDin,
  totals,
  valoare,
} from '../calc';
```

`Product` is no longer referenced here, and `procenteDinCantitati` moved behind
`templatesCuProcenteNoi` in `templates.ts`. Add, next to the existing
`../dialog`, `../router`, `../toast` and `../sidebar` imports:

```ts
import { DRAFT_PREFIX, draftHash, templatesCuProcenteNoi } from '../templates';
```

- [ ] **Step 2: Take the template id and load from it**

Change the signature and the load block:

```ts
export async function renderDocumentView(
  outlet: HTMLElement,
  id: string | undefined,
  refreshSidebar: () => Promise<void>,
  templateId: number | undefined,
): Promise<void> {
  let doc: Document;
  let unitate = '';
  // The rate new rows start from. Rows already on the document keep their own.
  let cotaImplicita = 0;
  // The template this document was made from, and its name for the header. A
  // document whose template has since been deleted has neither.
  let templates: Template[] = [];
  let templateNume = '';
  // Each product's share of the carcass, by product id. The "ce iese"
  // quantities are filled from these whenever the input quantity changes. It
  // is empty for a document whose template is gone, which is what turns the
  // automatic fill off.
  let procente = new Map<number, number>();
  let marjaNudged = false;

  try {
    const [settings, loadedTemplates] = await Promise.all([GetSettings(), ListTemplates()]);
    unitate = settings.unitateNume;
    cotaImplicita = settings.cotaTva;
    templates = loadedTemplates;

    if (id !== undefined) {
      doc = await GetDocument(Number(id));
    } else {
      // A draft route with no template on it is a stale hash or the fresh-launch
      // fallback. There is always at least one template, so send it to the
      // first rather than failing at the user.
      const wanted = templateId ?? templates[0]?.id;
      if (wanted === undefined) {
        outlet.innerHTML = '<p class="empty">Nu există niciun șablon. Creați unul din Setări.</p>';
        return;
      }
      if (templateId === undefined) {
        navigate(draftHash(wanted));
        return;
      }
      doc = await NewDocumentDraft(wanted);
    }

    const template = templates.find((t) => t.id === doc.templateId);
    if (template !== undefined) {
      templateNume = template.nume;
      procente = new Map(template.products.map((p) => [p.id, p.procentDinIntrare]));
    }
  } catch (err) {
    showError('Nu s-a putut încărca documentul', err);
    outlet.innerHTML = '<p class="empty">Documentul nu a putut fi încărcat.</p>';
    return;
  }
```

Add `Template` to the type import from `../api`, drop `ListProducts` / `SaveProducts` and add `ListTemplates` / `SaveTemplates`, and drop the now-unused `procenteDinCantitati` import from `../calc` (it moved into `templates.ts`).

- [ ] **Step 3: Show the template in the header, and the note when it is gone**

Add a read-only field to the header grid, after Gestiune:

```ts
        <div class="field">
          <label for="f-sablon">Șablon</label>
          <input id="f-sablon" value="${escapeHtml(templateNume || 'șters')}" readonly />
        </div>
```

And under the "Ce iese" heading, before the margin row:

```ts
      <h2>Ce iese</h2>
      ${
        procente.size === 0
          ? `<p class="empty">Șablonul acestui document a fost șters; cantitățile nu se
             mai completează automat din cantitatea de la "ce intră".</p>`
          : ''
      }
```

- [ ] **Step 4: Hide the margin controls without a template**

Wrap the whole `<div class="marja">` block:

```ts
      ${procente.size === 0 ? '' : marjaRow()}
```

with `marjaRow()` returning today's markup verbatim. Then guard the wiring, because the elements are no longer always present:

```ts
    const marjaPlus = outlet.querySelector('#marja-plus');
    if (marjaPlus) marjaPlus.addEventListener('click', () => stepMarja(PAS_MARJA));
    const marjaMinus = outlet.querySelector('#marja-minus');
    if (marjaMinus) marjaMinus.addEventListener('click', () => stepMarja(-PAS_MARJA));
    const saveProcenteBtn = outlet.querySelector('#save-procente');
    if (saveProcenteBtn) saveProcenteBtn.addEventListener('click', () => void onSaveProcente());
```

`recompute()` writes `#f-marja`; `setField` is already null-safe, so it needs no change.

- [ ] **Step 5: Point the ratio write-back at the document's template**

Replace `onSaveProcente` with:

```ts
  /**
   * Stores the split the user nudged into as this document's template's carcass
   * ratios, so every later document from that template opens on it — the same
   * column Setări shows and edits by hand. Only this document's template is
   * touched; the others go back exactly as they came.
   */
  async function onSaveProcente(): Promise<void> {
    readForm();
    if (doc.templateId === undefined || doc.templateId === null) return;

    let stored: Template[];
    try {
      stored = await ListTemplates();
    } catch (err) {
      showError('Lista de șabloane nu a putut fi citită', err);
      return;
    }

    const cantitatiPerProdus = new Map<number, number>();
    doc.iesire.forEach((row) => {
      if (row.productId == null) return;
      cantitatiPerProdus.set(
        row.productId,
        (cantitatiPerProdus.get(row.productId) ?? 0) + row.cantitate,
      );
    });

    const actualizate = templatesCuProcenteNoi(stored, doc.templateId, cantitatiPerProdus);
    if (actualizate === undefined) {
      showToast('Nu există cantități din care să se calculeze procentele.');
      return;
    }

    const nume = stored.find((t) => t.id === doc.templateId)?.nume ?? '';
    if (
      !(await showConfirm(
        `Salvați procentele noi? Procentele șablonului „${nume}” din Setări vor fi ` +
          'înlocuite cu cele rezultate din cantitățile de pe acest document.',
      ))
    ) {
      return;
    }

    try {
      await SaveTemplates(actualizate);
      // The form fills "ce iese" from these whenever the carcass weight
      // changes, so the in-memory copy has to move with the stored one or the
      // next keystroke would undo what was just saved.
      const salvat = actualizate.find((t) => t.id === doc.templateId);
      procente = new Map((salvat?.products ?? []).map((p) => [p.id, p.procentDinIntrare]));
      marjaNudged = false;
      syncSalvareProcente();
      showToast('Procentele au fost salvate!');
    } catch (err) {
      showError('Procentele nu au putut fi salvate', err);
    }
  }
```

- [ ] **Step 6: Leave "ce iese" alone when there are no ratios**

`aplicaProcente` already returns early when no row has a ratio (`indici.length === 0`), so a document with an empty `procente` map is handled. Confirm by reading it; no change needed. `onInput`'s call is likewise safe.

- [ ] **Step 7: Fix `onDelete`'s navigation**

`navigate('#/document/new')` now hits the bare draft route, which redirects. Point it at the document's own template when there is one:

```ts
      navigate(doc.templateId == null ? DRAFT_PREFIX : draftHash(doc.templateId));
```

Import `DRAFT_PREFIX` and `draftHash` from `../templates`.

- [ ] **Step 8: Verify**

Run: `cd frontend && npm test && npx tsc --noEmit`
Expected: PASS, with no unused-import or arity errors left over from Task 6.

- [ ] **Step 9: Commit**

```bash
git add frontend/src/views/document.ts
git commit -m "feat(document): ratios follow the document's template

\"Salvează procentele noi\" writes back to the template the document was
made from. A document whose template was deleted stays complete and
printable, without the automatic fill.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014ngEUTuc3QwZza4FWtUudT"
```

---

### Task 9: End-to-end check and docs

**Files:**
- Modify: `README.md`
- Manual: run the app

**Interfaces:**
- Consumes: everything above.
- Produces: nothing.

- [ ] **Step 1: Run everything**

```bash
go test ./... && (cd frontend && npm test && npx tsc --noEmit)
```

Expected: all PASS.

- [ ] **Step 2: Run the app and walk the feature**

Delete the dev database first so the fresh-install path is what gets exercised:

```bash
rm -f ~/Library/Application\ Support/proces-verbal-transare/data.db
wails dev
```

Walk it:
1. The app opens on a new document from **Carcasa Porc**, with the 19 products filled in. "+ Document nou" navigates straight there — no menu, because there is one template.
2. Setări shows one section, **Carcasa Porc**, expanded, total `100,000 %`.
3. *Duplică* → rename the copy to **Carcasa Vitel**, cut it down to a handful of products, make the ratios reach 100, Salvează. Confirm a template left below 100% refuses to save and names itself in the message.
4. "+ Document nou" now opens a menu with both. Pick **Carcasa Vitel**: the "ce intră" row reads *Carcasa Vitel*, and "ce iese" holds only the calf products.
5. Type a carcass weight; the calf quantities fill in from its ratios.
6. Nudge the margin, press *Salvează procentele noi*, confirm. Go to Setări: only the calf section's ratios changed; the pig section is untouched.
7. Save the calf document. Its NR is the next one after the last pig document — one register.
8. Printează: the PDF is unchanged in layout and prints the calf document's own rows.
9. Delete **Carcasa Vitel** from Setări and save. The calf document is still in the sidebar, still opens, still prints; its Șablon field reads *șters*, the margin buttons are gone, and the note under "Ce iese" is there.

Fix anything that does not behave as described before continuing.

- [ ] **Step 3: Update the README**

In `README.md`, replace the last paragraph of the **Date** section:

```markdown
La prima pornire se creează șablonul *Carcasa Porc* cu cele 19 produse de pe
formularul tipărit, împreună cu un proces verbal de exemplu. Un șablon este o
listă "ce iese" cu numele ei; se pot crea oricâte, din ecranul **Setări**, iar
la crearea unui document se alege din care șablon pornește. Numerotarea
documentelor este comună tuturor șabloanelor.
```

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs: describe templates in the README

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014ngEUTuc3QwZza4FWtUudT"
```

---

## Notes for the executor

- **`window.alert` and `window.confirm` do not work** in the macOS webview this app runs in — see the comment at the top of `frontend/src/dialog.ts`. Use `showAlert` / `showConfirm` from `../dialog`. `setari.ts` currently uses `window.alert`; Task 7 fixes that.
- **`escapeHtml` is mandatory** on every value interpolated into an `innerHTML` template, including inside attributes. Template names are user input and go into both.
- **Do not touch `internal/pdfdoc`.** The template is not printed.
- **Do not make numbering per-template.** If a test seems to want that, re-read the spec.
