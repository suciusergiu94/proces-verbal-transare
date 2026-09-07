# Proces Verbal de Transare Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a local desktop app (Wails + Go + SQLite) that replaces the paper "Proces verbal de transare" form: fill in a document, keep a history, and export a PDF matching the paper layout.

**Architecture:** Go backend split into small packages (`model`, `calc`, `store`, `pdfdoc`, `appdir`) with `app.go` acting as a thin Wails binding layer over a `*store.Store`. Persistence is a single SQLite file opened with the pure-Go `modernc.org/sqlite` driver. The frontend is the existing vanilla Vite/TypeScript scaffold with a hash router, a persistent sidebar, and two views (document form, settings); all money/quantity math is duplicated in TS for live display and re-derived in Go for the PDF.

**Tech Stack:** Go 1.25, Wails v2.15, `modernc.org/sqlite`, `github.com/go-pdf/fpdf`, TypeScript 5.4, Vite 8, Vitest.

**Spec:** `docs/superpowers/specs/2026-09-07-proces-verbal-transare-design.md`

## Global Constraints

- **No CGO.** Every Go dependency must be pure Go so `wails build -platform windows/amd64` works from macOS. Only two new direct dependencies are allowed: `modernc.org/sqlite` and `github.com/go-pdf/fpdf`.
- **All user-facing text is Romanian.** UI labels, buttons, error messages, PDF labels.
- **PDF labels are ASCII, copied verbatim from the paper form** (the printed form has no diacritics): `Unitatea:`, `Gestiunea:`, `Se aproba`, `Administrator`, `Contabil`, `Proces verbal de Transare`, `NR`, `Intocmit in data de`, `Din materia prima primita cu documentul nr`, `sau *`, `urmatoarele sortimente`, `Nr. Crt.`, `Denumire produs`, `U/M`, `Cantitate`, `Pret fara TVA`, `Valoare`, `Pret cu TVA`, `Valoare`, `Diferenta`, `plus`, `minus`, `Suma cu care se`, `gestiunea`, `incarca`, `descarca`, `Gestionar :`, `Calculator:`, `Vizat compartiment productie`.
- **UI labels DO use diacritics** (`Salvează`, `Șterge`, `Setări`, `Printează`, `Diferență`, `Cantitate`, `Preț fără TVA`, `Preț cu TVA`, `Valoare`, `Gestiune`, `Document referință`).
- **Money and quantities are rounded to 2 decimals** at every display and storage boundary, using `calc.Round2` / `round2` (half-away-from-zero).
- **Single-user, local-only.** No auth, no network calls, no cloud sync.
- **DB location:** `os.UserConfigDir()/proces-verbal-transare/data.db`, directory created if missing.
- **The 19 seed products, verbatim from the paper form** (all `U.M.` = `Kg`), in this order:

  | # | Denumire | Preț cu TVA |
  |---|----------|-------------|
  | 1 | Pulpa fara os | 21.9 |
  | 2 | Muschiulet | 39.5 |
  | 3 | Cotlet cu os | 24.0 |
  | 4 | Cotlet fara os | 29.5 |
  | 5 | Ceafa cu os | 24.0 |
  | 6 | Ceafa fara os | 33.9 |
  | 7 | Fleica | 30.5 |
  | 8 | Costita | 29.0 |
  | 9 | Ciolan | 16.5 |
  | 10 | Ciolan dezosat | 25.0 |
  | 11 | CPL | 20.9 |
  | 12 | Spata porc | 21.9 |
  | 13 | Slanina tablii | 18.5 |
  | 14 | Slanina lucru | 12.0 |
  | 15 | Oase garf | 6.5 |
  | 16 | Picioare | 6.5 |
  | 17 | Oase mici | 2.0 |
  | 18 | Sorici | 2.0 |
  | 19 | Deseu fara valoare | 0 |

  Note: row 4 is printed on the form as "Cotiet fara os", an evident typo for "Cotlet fara os". Seed it as **Cotlet fara os**; the user can rename it in Setări.

---

## File Structure

**Go backend**

| File | Responsibility |
|------|----------------|
| `internal/model/model.go` | Plain data structs shared by store, calc, pdf and the Wails bindings. No logic. |
| `internal/calc/calc.go` | Pure arithmetic: row values, table totals, diferență/încarcă-descarcă sign, `Round2`. |
| `internal/calc/calc_test.go` | Unit tests for the above. |
| `internal/appdir/appdir.go` | Resolves and creates the DB directory path. |
| `internal/store/schema.go` | DDL, migration-on-startup, product seeding. |
| `internal/store/store.go` | `Open`/`Close`, settings, products. |
| `internal/store/documents.go` | Document + rows CRUD. |
| `internal/store/store_test.go` | Schema/seed/settings/products round-trips. |
| `internal/store/documents_test.go` | Document CRUD round-trips. |
| `internal/pdfdoc/fold.go` | Diacritic → ASCII folding for user-entered text. |
| `internal/pdfdoc/pdf.go` | Renders a `model.Document` to PDF bytes. |
| `internal/pdfdoc/pdf_test.go` | Fold tests + smoke test that rendering produces a valid non-empty PDF. |
| `app.go` (modify) | Wails-bound methods; owns the `*store.Store`; runs the save dialog and opens the file. |
| `main.go` (modify) | Adds `OnShutdown`. |

Package is named `pdfdoc`, not `pdf`, to avoid shadowing when a variable named `pdf` is used inside it.

**Frontend**

| File | Responsibility |
|------|----------------|
| `frontend/src/calc.ts` | TS mirror of `internal/calc` for live display. |
| `frontend/src/calc.test.ts` | Vitest tests for the above. |
| `frontend/src/format.ts` | Number parsing/formatting (Romanian comma decimals) and date display. |
| `frontend/src/format.test.ts` | Vitest tests for the above. |
| `frontend/src/api.ts` | Typed thin wrapper over `../wailsjs/go/main/App`. |
| `frontend/src/router.ts` | Hash router: `#/document/new`, `#/document/<id>`, `#/setari`. |
| `frontend/src/sidebar.ts` | Renders and refreshes the sidebar (button + document list + Setări link). |
| `frontend/src/views/document.ts` | The document form view. |
| `frontend/src/views/setari.ts` | The settings view. |
| `frontend/src/main.ts` (rewrite) | Shell layout, mounts sidebar, starts router. |
| `frontend/src/style.css` (rewrite) | App styling. |

---

### Task 1: Domain models and calculation helpers

Pure Go, no dependencies. Everything else builds on these types.

**Files:**
- Create: `internal/model/model.go`
- Create: `internal/calc/calc.go`
- Test: `internal/calc/calc_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: the `model` structs listed below, and `calc.Round2(float64) float64`, `calc.ValoareFaraTVA(cantitate, pret float64) float64`, `calc.ValoareCuTVA(cantitate, pret float64) float64`, `calc.TotalsIntrare([]model.IntrareRow) calc.Totals`, `calc.TotalsIesire([]model.IesireRow) calc.Totals`, `calc.Diferenta(totalIesireCuTVA, totalIntrareCuTVA float64) (string, float64)`, `calc.IncarcaDescarca(totalIesireCuTVA, totalIntrareCuTVA float64) (string, float64)`, `calc.Totals{Cantitate, ValoareFaraTVA, ValoareCuTVA float64}`.

- [ ] **Step 1: Create the model package**

Create `internal/model/model.go`:

```go
// Package model holds the plain data structures shared by the store, the
// calculation helpers, the PDF renderer and the Wails bindings.
package model

// Settings is the single-row application configuration.
type Settings struct {
	UnitateNume string `json:"unitateNume"`
	NextNr      int    `json:"nextNr"`
}

// Product is one entry of the fixed "ce iese" product list.
type Product struct {
	ID        int64   `json:"id"`
	Denumire  string  `json:"denumire"`
	UM        string  `json:"um"`
	PretCuTVA float64 `json:"pretCuTva"`
	Ordine    int     `json:"ordine"`
}

// IntrareRow is a free-text row of the "ce intra" table.
type IntrareRow struct {
	ID          int64   `json:"id"`
	Pozitie     int     `json:"pozitie"`
	Denumire    string  `json:"denumire"`
	UM          string  `json:"um"`
	Cantitate   float64 `json:"cantitate"`
	PretFaraTVA float64 `json:"pretFaraTva"`
	PretCuTVA   float64 `json:"pretCuTva"`
}

// IesireRow is a row of the "ce iese" table. Denumire, UM and PretCuTVA are
// snapshots taken from the product at save time so the document keeps its
// meaning after the product list is edited.
type IesireRow struct {
	ID          int64   `json:"id"`
	ProductID   *int64  `json:"productId"`
	Pozitie     int     `json:"pozitie"`
	Denumire    string  `json:"denumire"`
	UM          string  `json:"um"`
	PretCuTVA   float64 `json:"pretCuTva"`
	Cantitate   float64 `json:"cantitate"`
	PretFaraTVA float64 `json:"pretFaraTva"`
}

// Document is a full proces verbal with both tables.
type Document struct {
	ID                         int64        `json:"id"`
	Nr                         int          `json:"nr"`
	Data                       string       `json:"data"` // ISO YYYY-MM-DD
	Gestiune                   string       `json:"gestiune"`
	DocumentReferinta          string       `json:"documentReferinta"`
	DiferentaTip               string       `json:"diferentaTip"` // "plus" | "minus" | ""
	DiferentaValoare           float64      `json:"diferentaValoare"`
	IncarcaDescarcaTip         string       `json:"incarcaDescarcaTip"` // "incarca" | "descarca" | ""
	IncarcaDescarcaValoare     float64      `json:"incarcaDescarcaValoare"`
	Gestionar                  string       `json:"gestionar"`
	Calculator                 string       `json:"calculator"`
	VizatCompartimentProductie string       `json:"vizatCompartimentProductie"`
	CreatedAt                  string       `json:"createdAt"`
	UpdatedAt                  string       `json:"updatedAt"`
	Intrare                    []IntrareRow `json:"intrare"`
	Iesire                     []IesireRow  `json:"iesire"`
}

// DocumentSummary is the sidebar list entry.
type DocumentSummary struct {
	ID       int64  `json:"id"`
	Nr       int    `json:"nr"`
	Data     string `json:"data"`
	Gestiune string `json:"gestiune"`
}
```

- [ ] **Step 2: Write the failing calc tests**

Create `internal/calc/calc_test.go`. The row fixtures are taken from the filled paper form NR 193 (03.09.26).

```go
package calc

import (
	"testing"

	"proces-verbal-transare/internal/model"
)

func TestRound2(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{
		{1.005, 1.01},
		{2.344999, 2.34},
		{-1.005, -1.01},
		{0, 0},
		{328.5, 328.5},
	}
	for _, c := range cases {
		if got := Round2(c.in); got != c.want {
			t.Errorf("Round2(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestValoare(t *testing.T) {
	// From the filled form: Pulpa fara os, 15 Kg x 21.9 = 328.50.
	if got := ValoareCuTVA(15, 21.9); got != 328.5 {
		t.Errorf("ValoareCuTVA(15, 21.9) = %v, want 328.5", got)
	}
	// Ceafa fara os, 8.5 Kg x 33.9 = 288.15.
	if got := ValoareCuTVA(8.5, 33.9); got != 288.15 {
		t.Errorf("ValoareCuTVA(8.5, 33.9) = %v, want 288.15", got)
	}
	// Slanina tablii, 10.5 Kg x 18.5 = 194.25.
	if got := ValoareCuTVA(10.5, 18.5); got != 194.25 {
		t.Errorf("ValoareCuTVA(10.5, 18.5) = %v, want 194.25", got)
	}
	if got := ValoareFaraTVA(15, 16.5); got != 247.5 {
		t.Errorf("ValoareFaraTVA(15, 16.5) = %v, want 247.5", got)
	}
}

func TestTotalsIntrare(t *testing.T) {
	rows := []model.IntrareRow{
		{Cantitate: 162.2, PretFaraTVA: 12.5, PretCuTVA: 14.03},
		{Cantitate: 10, PretFaraTVA: 2, PretCuTVA: 2.5},
	}
	got := TotalsIntrare(rows)
	if got.Cantitate != 172.2 {
		t.Errorf("Cantitate = %v, want 172.2", got.Cantitate)
	}
	if got.ValoareFaraTVA != 2047.5 {
		t.Errorf("ValoareFaraTVA = %v, want 2047.5", got.ValoareFaraTVA)
	}
	if got.ValoareCuTVA != 2300.67 {
		t.Errorf("ValoareCuTVA = %v, want 2300.67", got.ValoareCuTVA)
	}
}

func TestTotalsIesire(t *testing.T) {
	rows := []model.IesireRow{
		{Cantitate: 15, PretFaraTVA: 16.5, PretCuTVA: 21.9},
		{Cantitate: 8.5, PretFaraTVA: 0, PretCuTVA: 33.9},
	}
	got := TotalsIesire(rows)
	if got.Cantitate != 23.5 {
		t.Errorf("Cantitate = %v, want 23.5", got.Cantitate)
	}
	if got.ValoareFaraTVA != 247.5 {
		t.Errorf("ValoareFaraTVA = %v, want 247.5", got.ValoareFaraTVA)
	}
	if got.ValoareCuTVA != 616.65 {
		t.Errorf("ValoareCuTVA = %v, want 616.65", got.ValoareCuTVA)
	}
}

func TestTotalsEmpty(t *testing.T) {
	got := TotalsIesire(nil)
	if got != (Totals{}) {
		t.Errorf("TotalsIesire(nil) = %+v, want zero Totals", got)
	}
}

func TestDiferenta(t *testing.T) {
	tip, val := Diferenta(2300.67, 1000)
	if tip != "plus" || val != 1300.67 {
		t.Errorf("Diferenta(2300.67, 1000) = %q, %v; want \"plus\", 1300.67", tip, val)
	}
	tip, val = Diferenta(1000, 2300.67)
	if tip != "minus" || val != 1300.67 {
		t.Errorf("Diferenta(1000, 2300.67) = %q, %v; want \"minus\", 1300.67", tip, val)
	}
	tip, val = Diferenta(500, 500)
	if tip != "" || val != 0 {
		t.Errorf("Diferenta(500, 500) = %q, %v; want \"\", 0", tip, val)
	}
}

func TestIncarcaDescarca(t *testing.T) {
	tip, val := IncarcaDescarca(2300.67, 1000)
	if tip != "incarca" || val != 1300.67 {
		t.Errorf("IncarcaDescarca(2300.67, 1000) = %q, %v; want \"incarca\", 1300.67", tip, val)
	}
	tip, val = IncarcaDescarca(1000, 2300.67)
	if tip != "descarca" || val != 1300.67 {
		t.Errorf("IncarcaDescarca(1000, 2300.67) = %q, %v; want \"descarca\", 1300.67", tip, val)
	}
	tip, val = IncarcaDescarca(500, 500)
	if tip != "" || val != 0 {
		t.Errorf("IncarcaDescarca(500, 500) = %q, %v; want \"\", 0", tip, val)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/calc/...`
Expected: FAIL — the `calc` package does not exist yet (`no required module provides package`/build error).

- [ ] **Step 4: Write the calc implementation**

Create `internal/calc/calc.go`:

```go
// Package calc holds the arithmetic of the proces verbal: per-row values,
// per-table totals and the difference between the two tables.
package calc

import (
	"math"

	"proces-verbal-transare/internal/model"
)

// Totals is the sum of a table's numeric columns.
type Totals struct {
	Cantitate      float64 `json:"cantitate"`
	ValoareFaraTVA float64 `json:"valoareFaraTva"`
	ValoareCuTVA   float64 `json:"valoareCuTva"`
}

// Round2 rounds to two decimals, half away from zero.
func Round2(v float64) float64 {
	r := math.Round(math.Abs(v)*100) / 100
	if v < 0 {
		return -r
	}
	return r
}

// ValoareFaraTVA is cantitate x pret fara TVA.
func ValoareFaraTVA(cantitate, pret float64) float64 {
	return Round2(cantitate * pret)
}

// ValoareCuTVA is cantitate x pret cu TVA.
func ValoareCuTVA(cantitate, pret float64) float64 {
	return Round2(cantitate * pret)
}

// TotalsIntrare sums the "ce intra" table.
func TotalsIntrare(rows []model.IntrareRow) Totals {
	var t Totals
	for _, r := range rows {
		t.Cantitate += r.Cantitate
		t.ValoareFaraTVA += ValoareFaraTVA(r.Cantitate, r.PretFaraTVA)
		t.ValoareCuTVA += ValoareCuTVA(r.Cantitate, r.PretCuTVA)
	}
	return roundTotals(t)
}

// TotalsIesire sums the "ce iese" table.
func TotalsIesire(rows []model.IesireRow) Totals {
	var t Totals
	for _, r := range rows {
		t.Cantitate += r.Cantitate
		t.ValoareFaraTVA += ValoareFaraTVA(r.Cantitate, r.PretFaraTVA)
		t.ValoareCuTVA += ValoareCuTVA(r.Cantitate, r.PretCuTVA)
	}
	return roundTotals(t)
}

func roundTotals(t Totals) Totals {
	return Totals{
		Cantitate:      Round2(t.Cantitate),
		ValoareFaraTVA: Round2(t.ValoareFaraTVA),
		ValoareCuTVA:   Round2(t.ValoareCuTVA),
	}
}

// Diferenta reports the signed gap between the two tables' "valoare cu TVA"
// totals as a ("plus"|"minus"|"") tip and a non-negative magnitude.
func Diferenta(totalIesireCuTVA, totalIntrareCuTVA float64) (string, float64) {
	d := Round2(totalIesireCuTVA - totalIntrareCuTVA)
	switch {
	case d > 0:
		return "plus", d
	case d < 0:
		return "minus", -d
	default:
		return "", 0
	}
}

// IncarcaDescarca is the same magnitude as Diferenta, labelled the way the
// bottom-right block of the form labels it.
func IncarcaDescarca(totalIesireCuTVA, totalIntrareCuTVA float64) (string, float64) {
	tip, val := Diferenta(totalIesireCuTVA, totalIntrareCuTVA)
	switch tip {
	case "plus":
		return "incarca", val
	case "minus":
		return "descarca", val
	default:
		return "", 0
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/calc/... -v`
Expected: PASS — all 7 tests.

- [ ] **Step 6: Commit**

```bash
git add internal/model/model.go internal/calc/calc.go internal/calc/calc_test.go
git commit -m "feat: add domain models and calculation helpers"
```

---

### Task 2: SQLite store — schema, seed, settings, products

Brings in the SQLite driver, creates the schema on open, seeds the 19 products, and implements the settings and products access.

**Files:**
- Create: `internal/appdir/appdir.go`
- Create: `internal/store/schema.go`
- Create: `internal/store/store.go`
- Test: `internal/store/store_test.go`
- Modify: `go.mod`, `go.sum` (via `go get`)

**Interfaces:**
- Consumes: `model.Settings`, `model.Product` from Task 1.
- Produces: `appdir.DBPath() (string, error)`; `store.Open(path string) (*store.Store, error)`, `(*Store).Close() error`, `(*Store).GetSettings() (model.Settings, error)`, `(*Store).SaveSettings(model.Settings) error`, `(*Store).ListProducts() ([]model.Product, error)`, `(*Store).SaveProducts([]model.Product) error`.

`SaveProducts` takes the full desired list: rows with `ID == 0` are inserted, rows with a known ID are updated, products absent from the list are deleted, and `Ordine` is reassigned from the slice order (0-based).

- [ ] **Step 1: Add the SQLite dependency**

```bash
go get modernc.org/sqlite@latest
go mod tidy
```

Verify it is pure Go (no CGO required):

```bash
CGO_ENABLED=0 go build ./...
```
Expected: succeeds with no output.

- [ ] **Step 2: Write the failing store tests**

Create `internal/store/store_test.go`:

```go
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
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/store/...`
Expected: FAIL — build error, `store.Open` undefined.

- [ ] **Step 4: Write the app-directory helper**

Create `internal/appdir/appdir.go`:

```go
// Package appdir resolves where the application keeps its data.
package appdir

import (
	"os"
	"path/filepath"
)

const dirName = "proces-verbal-transare"

// DBPath returns the SQLite file path inside the per-user config directory,
// creating the directory if it does not exist.
func DBPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, dirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "data.db"), nil
}
```

- [ ] **Step 5: Write the schema and seed**

Create `internal/store/schema.go`:

```go
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
```

- [ ] **Step 6: Write the store implementation**

Create `internal/store/store.go`:

```go
// Package store is the SQLite persistence layer.
package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"

	"proces-verbal-transare/internal/model"
)

// Store owns the database handle.
type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the SQLite file at path, applies the schema
// and seeds first-run data.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("deschidere bazei de date: %w", err)
	}
	// SQLite tolerates a single writer; one connection keeps writes serialized
	// and keeps an in-memory DB from splitting across connections.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("conectare la baza de date: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrare baza de date: %w", err)
	}
	return &Store{db: db}, nil
}

// Close releases the database handle.
func (s *Store) Close() error {
	return s.db.Close()
}

// GetSettings reads the single settings row.
func (s *Store) GetSettings() (model.Settings, error) {
	var out model.Settings
	err := s.db.QueryRow(
		`SELECT unitate_nume, next_nr FROM settings WHERE id = 1`,
	).Scan(&out.UnitateNume, &out.NextNr)
	if err != nil {
		return model.Settings{}, fmt.Errorf("citire setari: %w", err)
	}
	return out, nil
}

// SaveSettings overwrites the settings row.
func (s *Store) SaveSettings(in model.Settings) error {
	_, err := s.db.Exec(
		`INSERT INTO settings (id, unitate_nume, next_nr) VALUES (1, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET unitate_nume = excluded.unitate_nume, next_nr = excluded.next_nr`,
		in.UnitateNume, in.NextNr,
	)
	if err != nil {
		return fmt.Errorf("salvare setari: %w", err)
	}
	return nil
}

// ListProducts returns every product ordered by ordine.
func (s *Store) ListProducts() ([]model.Product, error) {
	rows, err := s.db.Query(
		`SELECT id, denumire, um, pret_cu_tva, ordine FROM products ORDER BY ordine, id`,
	)
	if err != nil {
		return nil, fmt.Errorf("citire produse: %w", err)
	}
	defer rows.Close()

	out := []model.Product{}
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.Denumire, &p.UM, &p.PretCuTVA, &p.Ordine); err != nil {
			return nil, fmt.Errorf("citire produs: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SaveProducts makes the stored product list match in: rows with ID 0 are
// inserted, known IDs are updated, absent IDs are deleted, and ordine is
// reassigned from the slice order.
func (s *Store) SaveProducts(in []model.Product) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("salvare produse: %w", err)
	}
	defer tx.Rollback()

	keep := make(map[int64]bool, len(in))
	for i, p := range in {
		if p.ID == 0 {
			res, err := tx.Exec(
				`INSERT INTO products (denumire, um, pret_cu_tva, ordine) VALUES (?, ?, ?, ?)`,
				p.Denumire, p.UM, p.PretCuTVA, i,
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
			`UPDATE products SET denumire = ?, um = ?, pret_cu_tva = ?, ordine = ? WHERE id = ?`,
			p.Denumire, p.UM, p.PretCuTVA, i, p.ID,
		); err != nil {
			return fmt.Errorf("actualizare produs %q: %w", p.Denumire, err)
		}
		keep[p.ID] = true
	}

	existing, err := tx.Query(`SELECT id FROM products`)
	if err != nil {
		return fmt.Errorf("salvare produse: %w", err)
	}
	var stale []int64
	for existing.Next() {
		var id int64
		if err := existing.Scan(&id); err != nil {
			existing.Close()
			return fmt.Errorf("salvare produse: %w", err)
		}
		if !keep[id] {
			stale = append(stale, id)
		}
	}
	existing.Close()
	if err := existing.Err(); err != nil {
		return fmt.Errorf("salvare produse: %w", err)
	}

	for _, id := range stale {
		if _, err := tx.Exec(`DELETE FROM products WHERE id = ?`, id); err != nil {
			return fmt.Errorf("stergere produs: %w", err)
		}
	}
	return tx.Commit()
}
```

- [ ] **Step 7: Run tests to verify they pass**

Run: `go test ./internal/store/... -v`
Expected: PASS — `TestOpenSeedsProducts`, `TestOpenIsIdempotent`, `TestSettingsDefaultsAndRoundTrip`, `TestSaveProductsInsertsUpdatesDeletesAndReorders`.

- [ ] **Step 8: Commit**

```bash
git add go.mod go.sum internal/appdir/appdir.go internal/store/schema.go internal/store/store.go internal/store/store_test.go
git commit -m "feat: add SQLite store with schema, product seed and settings"
```

---

### Task 3: Document CRUD in the store

**Files:**
- Create: `internal/store/documents.go`
- Test: `internal/store/documents_test.go`

**Interfaces:**
- Consumes: `*store.Store` and `newTestStore` from Task 2; `model.Document`, `model.DocumentSummary`, `model.IntrareRow`, `model.IesireRow` from Task 1.
- Produces: `(*Store).ListDocuments() ([]model.DocumentSummary, error)`, `(*Store).GetDocument(id int64) (model.Document, error)`, `(*Store).SaveDocument(model.Document) (model.Document, error)`, `(*Store).DeleteDocument(id int64) error`, `(*Store).LastDocument() (model.Document, bool, error)`, `store.ErrNotFound`.

`SaveDocument` inserts when `ID == 0` and updates otherwise; on update it deletes and reinserts both row tables (rows are wholly owned by the document, so a replace is simpler and safer than a diff). It returns the stored document read back, with generated IDs and timestamps filled in.

- [ ] **Step 1: Write the failing document tests**

Create `internal/store/documents_test.go`:

```go
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
	s := newTestStore(t)

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
	s := newTestStore(t)

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
	s := newTestStore(t)

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
	s := newTestStore(t)

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

	products, err := s.ListProducts()
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	pid := products[0].ID

	doc := sampleDocument()
	doc.Iesire[0].ProductID = &pid
	saved, err := s.SaveDocument(doc)
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	if err := s.SaveProducts(products[1:]); err != nil {
		t.Fatalf("SaveProducts: %v", err)
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/store/...`
Expected: FAIL — `ErrNotFound`, `SaveDocument`, `GetDocument`, `ListDocuments`, `DeleteDocument`, `LastDocument` undefined.

- [ ] **Step 3: Write the document store implementation**

Create `internal/store/documents.go`:

```go
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"proces-verbal-transare/internal/model"
)

// ErrNotFound is returned when a document id does not exist.
var ErrNotFound = errors.New("documentul nu a fost gasit")

// ListDocuments returns every document, newest first.
func (s *Store) ListDocuments() ([]model.DocumentSummary, error) {
	rows, err := s.db.Query(
		`SELECT id, nr, data, gestiune FROM documents ORDER BY data DESC, id DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("citire documente: %w", err)
	}
	defer rows.Close()

	out := []model.DocumentSummary{}
	for rows.Next() {
		var d model.DocumentSummary
		if err := rows.Scan(&d.ID, &d.Nr, &d.Data, &d.Gestiune); err != nil {
			return nil, fmt.Errorf("citire document: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// GetDocument reads one document with both row tables.
func (s *Store) GetDocument(id int64) (model.Document, error) {
	var d model.Document
	err := s.db.QueryRow(
		`SELECT id, nr, data, gestiune, document_referinta, diferenta_tip, diferenta_valoare,
		        incarca_descarca_tip, incarca_descarca_valoare, gestionar, calculator,
		        vizat_compartiment_productie, created_at, updated_at
		 FROM documents WHERE id = ?`, id,
	).Scan(
		&d.ID, &d.Nr, &d.Data, &d.Gestiune, &d.DocumentReferinta, &d.DiferentaTip, &d.DiferentaValoare,
		&d.IncarcaDescarcaTip, &d.IncarcaDescarcaValoare, &d.Gestionar, &d.Calculator,
		&d.VizatCompartimentProductie, &d.CreatedAt, &d.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Document{}, ErrNotFound
	}
	if err != nil {
		return model.Document{}, fmt.Errorf("citire document: %w", err)
	}

	if d.Intrare, err = s.intrareRows(id); err != nil {
		return model.Document{}, err
	}
	if d.Iesire, err = s.iesireRows(id); err != nil {
		return model.Document{}, err
	}
	return d, nil
}

func (s *Store) intrareRows(documentID int64) ([]model.IntrareRow, error) {
	rows, err := s.db.Query(
		`SELECT id, pozitie, denumire, um, cantitate, pret_fara_tva, pret_cu_tva
		 FROM document_intrare_rows WHERE document_id = ? ORDER BY pozitie, id`, documentID,
	)
	if err != nil {
		return nil, fmt.Errorf("citire randuri intrare: %w", err)
	}
	defer rows.Close()

	out := []model.IntrareRow{}
	for rows.Next() {
		var r model.IntrareRow
		if err := rows.Scan(&r.ID, &r.Pozitie, &r.Denumire, &r.UM, &r.Cantitate, &r.PretFaraTVA, &r.PretCuTVA); err != nil {
			return nil, fmt.Errorf("citire rand intrare: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) iesireRows(documentID int64) ([]model.IesireRow, error) {
	rows, err := s.db.Query(
		`SELECT id, product_id, pozitie, denumire, um, pret_cu_tva, cantitate, pret_fara_tva
		 FROM document_iesire_rows WHERE document_id = ? ORDER BY pozitie, id`, documentID,
	)
	if err != nil {
		return nil, fmt.Errorf("citire randuri iesire: %w", err)
	}
	defer rows.Close()

	out := []model.IesireRow{}
	for rows.Next() {
		var (
			r         model.IesireRow
			productID sql.NullInt64
		)
		if err := rows.Scan(&r.ID, &productID, &r.Pozitie, &r.Denumire, &r.UM, &r.PretCuTVA, &r.Cantitate, &r.PretFaraTVA); err != nil {
			return nil, fmt.Errorf("citire rand iesire: %w", err)
		}
		if productID.Valid {
			id := productID.Int64
			r.ProductID = &id
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SaveDocument inserts d when its ID is 0 and updates it otherwise. Row tables
// are wholly owned by the document, so they are replaced rather than diffed.
// It returns the document as stored, re-read from the database.
func (s *Store) SaveDocument(d model.Document) (model.Document, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	tx, err := s.db.Begin()
	if err != nil {
		return model.Document{}, fmt.Errorf("salvare document: %w", err)
	}
	defer tx.Rollback()

	if d.ID == 0 {
		res, err := tx.Exec(
			`INSERT INTO documents (nr, data, gestiune, document_referinta, diferenta_tip, diferenta_valoare,
			         incarca_descarca_tip, incarca_descarca_valoare, gestionar, calculator,
			         vizat_compartiment_productie, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			d.Nr, d.Data, d.Gestiune, d.DocumentReferinta, d.DiferentaTip, d.DiferentaValoare,
			d.IncarcaDescarcaTip, d.IncarcaDescarcaValoare, d.Gestionar, d.Calculator,
			d.VizatCompartimentProductie, now, now,
		)
		if err != nil {
			return model.Document{}, fmt.Errorf("adaugare document: %w", err)
		}
		if d.ID, err = res.LastInsertId(); err != nil {
			return model.Document{}, fmt.Errorf("adaugare document: %w", err)
		}
	} else {
		res, err := tx.Exec(
			`UPDATE documents SET nr = ?, data = ?, gestiune = ?, document_referinta = ?,
			        diferenta_tip = ?, diferenta_valoare = ?, incarca_descarca_tip = ?,
			        incarca_descarca_valoare = ?, gestionar = ?, calculator = ?,
			        vizat_compartiment_productie = ?, updated_at = ?
			 WHERE id = ?`,
			d.Nr, d.Data, d.Gestiune, d.DocumentReferinta, d.DiferentaTip, d.DiferentaValoare,
			d.IncarcaDescarcaTip, d.IncarcaDescarcaValoare, d.Gestionar, d.Calculator,
			d.VizatCompartimentProductie, now, d.ID,
		)
		if err != nil {
			return model.Document{}, fmt.Errorf("actualizare document: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return model.Document{}, fmt.Errorf("actualizare document: %w", err)
		}
		if n == 0 {
			return model.Document{}, ErrNotFound
		}
		if _, err := tx.Exec(`DELETE FROM document_intrare_rows WHERE document_id = ?`, d.ID); err != nil {
			return model.Document{}, fmt.Errorf("actualizare randuri intrare: %w", err)
		}
		if _, err := tx.Exec(`DELETE FROM document_iesire_rows WHERE document_id = ?`, d.ID); err != nil {
			return model.Document{}, fmt.Errorf("actualizare randuri iesire: %w", err)
		}
	}

	for i, r := range d.Intrare {
		if _, err := tx.Exec(
			`INSERT INTO document_intrare_rows (document_id, pozitie, denumire, um, cantitate, pret_fara_tva, pret_cu_tva)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			d.ID, i, r.Denumire, r.UM, r.Cantitate, r.PretFaraTVA, r.PretCuTVA,
		); err != nil {
			return model.Document{}, fmt.Errorf("salvare rand intrare: %w", err)
		}
	}
	for i, r := range d.Iesire {
		var productID any
		if r.ProductID != nil {
			productID = *r.ProductID
		}
		if _, err := tx.Exec(
			`INSERT INTO document_iesire_rows (document_id, product_id, pozitie, denumire, um, pret_cu_tva, cantitate, pret_fara_tva)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			d.ID, productID, i, r.Denumire, r.UM, r.PretCuTVA, r.Cantitate, r.PretFaraTVA,
		); err != nil {
			return model.Document{}, fmt.Errorf("salvare rand iesire: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return model.Document{}, fmt.Errorf("salvare document: %w", err)
	}
	return s.GetDocument(d.ID)
}

// DeleteDocument removes a document; its rows go with it via ON DELETE CASCADE.
func (s *Store) DeleteDocument(id int64) error {
	res, err := s.db.Exec(`DELETE FROM documents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("stergere document: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("stergere document: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// LastDocument returns the most recently saved document, if any. It is the
// source of the prefilled "ce intra" rows on a new document.
func (s *Store) LastDocument() (model.Document, bool, error) {
	var id int64
	err := s.db.QueryRow(`SELECT id FROM documents ORDER BY id DESC LIMIT 1`).Scan(&id)
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

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/store/... -v`
Expected: PASS — all store tests including the seven new document tests.

- [ ] **Step 5: Commit**

```bash
git add internal/store/documents.go internal/store/documents_test.go
git commit -m "feat: add document CRUD to the SQLite store"
```

---

### Task 4: Wails bindings

Replaces the scaffold's `Greet` with the real API surface and wires the store into the app lifecycle.

**Files:**
- Modify: `app.go` (replace the whole file)
- Modify: `main.go:24-30` (add `OnShutdown`)
- Regenerate: `frontend/wailsjs/go/main/App.js`, `frontend/wailsjs/go/main/App.d.ts`, `frontend/wailsjs/go/models.ts`

**Interfaces:**
- Consumes: everything from Tasks 1–3.
- Produces: bound methods `GetSettings`, `SaveSettings`, `ListProducts`, `SaveProducts`, `ListDocuments`, `GetDocument`, `NewDocumentDraft`, `SaveDocument`, `DeleteDocument`. `ExportPDF` is added in Task 8 — do not add it yet.

`NewDocumentDraft` builds an unsaved `model.Document` with: `Nr` = `settings.NextNr`, `Data` = today, `Gestiune` = last document's gestiune, `Intrare` = a copy of the last document's intrare rows (IDs zeroed) or one blank row if there is no previous document, `Iesire` = one row per product in `ordine` with quantities zeroed.

`SaveDocument` bumps `settings.NextNr` to `doc.Nr + 1` after an insert whose `Nr` is greater than or equal to the current `NextNr`.

- [ ] **Step 1: Write the failing app tests**

Create `app_test.go` at the repo root:

```go
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

func TestNewDocumentDraftWithoutHistory(t *testing.T) {
	a := newTestApp(t)

	draft, err := a.NewDocumentDraft()
	if err != nil {
		t.Fatalf("NewDocumentDraft: %v", err)
	}
	if draft.ID != 0 {
		t.Errorf("draft.ID = %d, want 0", draft.ID)
	}
	if draft.Nr != 1 {
		t.Errorf("draft.Nr = %d, want 1 (settings.NextNr)", draft.Nr)
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
	if draft.Intrare[0].Cantitate != 162.20 {
		t.Errorf("draft.Intrare[0].Cantitate = %v, want the previous quantity", draft.Intrare[0].Cantitate)
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test .`
Expected: FAIL — `App` has no field `store`, `NewDocumentDraft` undefined.

- [ ] **Step 3: Replace app.go**

Replace the entire contents of `app.go`:

```go
package main

import (
	"context"
	"time"

	"proces-verbal-transare/internal/appdir"
	"proces-verbal-transare/internal/model"
	"proces-verbal-transare/internal/store"
)

// App is the Wails-bound application object. Every exported method here is
// callable from the frontend.
type App struct {
	ctx   context.Context
	store *store.Store
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{}
}

// startup opens the database and keeps the Wails context for runtime calls.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	path, err := appdir.DBPath()
	if err != nil {
		panic("nu s-a putut determina locatia bazei de date: " + err.Error())
	}
	s, err := store.Open(path)
	if err != nil {
		panic("nu s-a putut deschide baza de date: " + err.Error())
	}
	a.store = s
}

// shutdown closes the database.
func (a *App) shutdown(ctx context.Context) {
	if a.store != nil {
		a.store.Close()
	}
}

// GetSettings returns the application settings.
func (a *App) GetSettings() (model.Settings, error) {
	return a.store.GetSettings()
}

// SaveSettings overwrites the application settings.
func (a *App) SaveSettings(s model.Settings) error {
	return a.store.SaveSettings(s)
}

// ListProducts returns the "ce iese" product list in display order.
func (a *App) ListProducts() ([]model.Product, error) {
	return a.store.ListProducts()
}

// SaveProducts replaces the product list with the given one.
func (a *App) SaveProducts(products []model.Product) error {
	return a.store.SaveProducts(products)
}

// ListDocuments returns the sidebar history, newest first.
func (a *App) ListDocuments() ([]model.DocumentSummary, error) {
	return a.store.ListDocuments()
}

// GetDocument loads one saved document.
func (a *App) GetDocument(id int64) (model.Document, error) {
	return a.store.GetDocument(id)
}

// NewDocumentDraft builds an unsaved document prefilled from the settings, the
// product list and the previously saved document.
func (a *App) NewDocumentDraft() (model.Document, error) {
	settings, err := a.store.GetSettings()
	if err != nil {
		return model.Document{}, err
	}
	products, err := a.store.ListProducts()
	if err != nil {
		return model.Document{}, err
	}
	last, hasLast, err := a.store.LastDocument()
	if err != nil {
		return model.Document{}, err
	}

	draft := model.Document{
		Nr:      settings.NextNr,
		Data:    time.Now().Format("2006-01-02"),
		Intrare: []model.IntrareRow{},
		Iesire:  []model.IesireRow{},
	}

	if hasLast {
		draft.Gestiune = last.Gestiune
		for i, r := range last.Intrare {
			draft.Intrare = append(draft.Intrare, model.IntrareRow{
				Pozitie:     i,
				Denumire:    r.Denumire,
				UM:          r.UM,
				Cantitate:   r.Cantitate,
				PretFaraTVA: r.PretFaraTVA,
				PretCuTVA:   r.PretCuTVA,
			})
		}
	}
	if len(draft.Intrare) == 0 {
		draft.Intrare = append(draft.Intrare, model.IntrareRow{Pozitie: 0, UM: "Kg"})
	}

	for i, p := range products {
		id := p.ID
		draft.Iesire = append(draft.Iesire, model.IesireRow{
			ProductID: &id,
			Pozitie:   i,
			Denumire:  p.Denumire,
			UM:        p.UM,
			PretCuTVA: p.PretCuTVA,
		})
	}
	return draft, nil
}

// SaveDocument creates or updates a document. Creating one with a number at or
// above the current counter advances the counter past it.
func (a *App) SaveDocument(doc model.Document) (model.Document, error) {
	isNew := doc.ID == 0

	saved, err := a.store.SaveDocument(doc)
	if err != nil {
		return model.Document{}, err
	}
	if !isNew {
		return saved, nil
	}

	settings, err := a.store.GetSettings()
	if err != nil {
		return model.Document{}, err
	}
	if saved.Nr >= settings.NextNr {
		settings.NextNr = saved.Nr + 1
		if err := a.store.SaveSettings(settings); err != nil {
			return model.Document{}, err
		}
	}
	return saved, nil
}

// DeleteDocument removes a document and its rows.
func (a *App) DeleteDocument(id int64) error {
	return a.store.DeleteDocument(id)
}
```

- [ ] **Step 4: Wire shutdown in main.go**

In `main.go`, add `OnShutdown` next to `OnStartup`:

```go
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
```

Also change the window title and size to suit the form:

```go
		Title:  "Proces Verbal de Transare",
		Width:  1280,
		Height: 900,
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./... -v`
Expected: PASS — calc, store and the four new app tests.

- [ ] **Step 6: Regenerate the TypeScript bindings**

```bash
wails generate module
```

Confirm `frontend/wailsjs/go/models.ts` now exists and contains `Document`, `Product`, `Settings` and `DocumentSummary` under `export namespace model`, and that `frontend/wailsjs/go/main/App.d.ts` no longer declares `Greet`.

- [ ] **Step 7: Commit**

```bash
git add app.go app_test.go main.go frontend/wailsjs
git commit -m "feat: expose store operations as Wails bindings"
```

---

### Task 5: Frontend foundation — formatting, calc mirror, API wrapper, router, shell

Replaces the scaffold's demo page with the app shell. After this task the app runs and the sidebar lists documents, but the views are stubs.

**Files:**
- Modify: `frontend/package.json` (add Vitest)
- Create: `frontend/src/format.ts`, `frontend/src/format.test.ts`
- Create: `frontend/src/calc.ts`, `frontend/src/calc.test.ts`
- Create: `frontend/src/api.ts`
- Create: `frontend/src/router.ts`
- Create: `frontend/src/sidebar.ts`
- Modify: `frontend/src/main.ts` (replace the whole file)
- Modify: `frontend/src/style.css` (replace the whole file)

**Interfaces:**
- Consumes: the generated bindings from Task 4.
- Produces:
  - `format.ts`: `parseNumber(s: string): number`, `formatNumber(n: number, decimals?: number): string`, `formatDateRO(iso: string): string`.
  - `calc.ts`: `round2(v: number): number`, `valoare(cantitate: number, pret: number): number`, `type Totals = { cantitate: number; valoareFaraTva: number; valoareCuTva: number }`, `totals(rows: RowLike[]): Totals` where `RowLike = { cantitate: number; pretFaraTva: number; pretCuTva: number }`, `diferenta(iesireCuTva: number, intrareCuTva: number): { tip: 'plus' | 'minus' | ''; valoare: number }`, `incarcaDescarca(iesireCuTva: number, intrareCuTva: number): { tip: 'incarca' | 'descarca' | ''; valoare: number }`.
  - `api.ts`: re-exports the bound calls plus the `model` namespace type.
  - `router.ts`: `startRouter(routes: Route[], outlet: HTMLElement): void`, `navigate(hash: string): void`, `type Route = { pattern: RegExp; render: (outlet: HTMLElement, ...params: string[]) => void | Promise<void> }`.
  - `sidebar.ts`: `renderSidebar(el: HTMLElement): Promise<void>`.

- [ ] **Step 1: Add Vitest**

```bash
cd frontend && npm install -D vitest && cd ..
```

Add a `test` script to `frontend/package.json`'s `scripts`:

```json
    "test": "vitest run"
```

- [ ] **Step 2: Write the failing format and calc tests**

Create `frontend/src/format.test.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { formatDateRO, formatNumber, parseNumber } from './format';

describe('parseNumber', () => {
  it('accepts a dot decimal separator', () => {
    expect(parseNumber('162.20')).toBe(162.2);
  });

  it('accepts a comma decimal separator', () => {
    expect(parseNumber('162,20')).toBe(162.2);
  });

  it('treats blank and unparseable input as zero', () => {
    expect(parseNumber('')).toBe(0);
    expect(parseNumber('   ')).toBe(0);
    expect(parseNumber('abc')).toBe(0);
  });

  it('trims surrounding whitespace', () => {
    expect(parseNumber(' 15 ')).toBe(15);
  });
});

describe('formatNumber', () => {
  it('renders two decimals by default', () => {
    expect(formatNumber(328.5)).toBe('328.50');
    expect(formatNumber(0)).toBe('0.00');
  });

  it('honours an explicit decimal count', () => {
    expect(formatNumber(21.9, 3)).toBe('21.900');
  });
});

describe('formatDateRO', () => {
  it('renders an ISO date as dd.mm.yyyy', () => {
    expect(formatDateRO('2026-09-03')).toBe('03.09.2026');
  });

  it('passes through anything that is not an ISO date', () => {
    expect(formatDateRO('')).toBe('');
    expect(formatDateRO('nope')).toBe('nope');
  });
});
```

Create `frontend/src/calc.test.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { diferenta, incarcaDescarca, round2, totals, valoare } from './calc';

describe('round2', () => {
  it('rounds half away from zero', () => {
    expect(round2(1.005)).toBe(1.01);
    expect(round2(-1.005)).toBe(-1.01);
    expect(round2(2.344999)).toBe(2.34);
  });
});

describe('valoare', () => {
  it('matches the filled paper form', () => {
    expect(valoare(15, 21.9)).toBe(328.5);
    expect(valoare(8.5, 33.9)).toBe(288.15);
    expect(valoare(10.5, 18.5)).toBe(194.25);
  });
});

describe('totals', () => {
  it('sums each column', () => {
    const got = totals([
      { cantitate: 15, pretFaraTva: 16.5, pretCuTva: 21.9 },
      { cantitate: 8.5, pretFaraTva: 0, pretCuTva: 33.9 },
    ]);
    expect(got.cantitate).toBe(23.5);
    expect(got.valoareFaraTva).toBe(247.5);
    expect(got.valoareCuTva).toBe(616.65);
  });

  it('returns zeroes for an empty table', () => {
    expect(totals([])).toEqual({ cantitate: 0, valoareFaraTva: 0, valoareCuTva: 0 });
  });
});

describe('diferenta', () => {
  it('labels a surplus plus and a shortfall minus', () => {
    expect(diferenta(2300.67, 1000)).toEqual({ tip: 'plus', valoare: 1300.67 });
    expect(diferenta(1000, 2300.67)).toEqual({ tip: 'minus', valoare: 1300.67 });
    expect(diferenta(500, 500)).toEqual({ tip: '', valoare: 0 });
  });
});

describe('incarcaDescarca', () => {
  it('uses the bottom-right block wording', () => {
    expect(incarcaDescarca(2300.67, 1000)).toEqual({ tip: 'incarca', valoare: 1300.67 });
    expect(incarcaDescarca(1000, 2300.67)).toEqual({ tip: 'descarca', valoare: 1300.67 });
    expect(incarcaDescarca(500, 500)).toEqual({ tip: '', valoare: 0 });
  });
});
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd frontend && npm test`
Expected: FAIL — `Failed to resolve import "./format"` and `"./calc"`.

- [ ] **Step 4: Write format.ts and calc.ts**

Create `frontend/src/format.ts`:

```ts
/** Parses user input that may use either a dot or a comma decimal separator. */
export function parseNumber(input: string): number {
  const normalised = input.trim().replace(',', '.');
  if (normalised === '') return 0;
  const value = Number(normalised);
  return Number.isFinite(value) ? value : 0;
}

/** Renders a number with a fixed number of decimals (2 by default). */
export function formatNumber(value: number, decimals = 2): string {
  return value.toFixed(decimals);
}

/** Renders an ISO date (YYYY-MM-DD) the way the paper form writes it. */
export function formatDateRO(iso: string): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso);
  if (!match) return iso;
  const [, year, month, day] = match;
  return `${day}.${month}.${year}`;
}
```

Create `frontend/src/calc.ts`:

```ts
/**
 * Mirror of internal/calc, used for live display while the user types. The Go
 * side re-derives the same values for the PDF, so the two must stay in step.
 */

export type RowLike = {
  cantitate: number;
  pretFaraTva: number;
  pretCuTva: number;
};

export type Totals = {
  cantitate: number;
  valoareFaraTva: number;
  valoareCuTva: number;
};

/** Rounds to two decimals, half away from zero. */
export function round2(value: number): number {
  const rounded = Math.round(Math.abs(value) * 100) / 100;
  return value < 0 ? -rounded : rounded;
}

/** cantitate x pret, rounded to two decimals. */
export function valoare(cantitate: number, pret: number): number {
  return round2(cantitate * pret);
}

/** Sums a table's numeric columns. */
export function totals(rows: RowLike[]): Totals {
  const sum = rows.reduce<Totals>(
    (acc, row) => ({
      cantitate: acc.cantitate + row.cantitate,
      valoareFaraTva: acc.valoareFaraTva + valoare(row.cantitate, row.pretFaraTva),
      valoareCuTva: acc.valoareCuTva + valoare(row.cantitate, row.pretCuTva),
    }),
    { cantitate: 0, valoareFaraTva: 0, valoareCuTva: 0 },
  );
  return {
    cantitate: round2(sum.cantitate),
    valoareFaraTva: round2(sum.valoareFaraTva),
    valoareCuTva: round2(sum.valoareCuTva),
  };
}

/** The signed gap between the two tables' "valoare cu TVA" totals. */
export function diferenta(
  iesireCuTva: number,
  intrareCuTva: number,
): { tip: 'plus' | 'minus' | ''; valoare: number } {
  const delta = round2(iesireCuTva - intrareCuTva);
  if (delta > 0) return { tip: 'plus', valoare: delta };
  if (delta < 0) return { tip: 'minus', valoare: -delta };
  return { tip: '', valoare: 0 };
}

/** The same magnitude, labelled the way the bottom-right block labels it. */
export function incarcaDescarca(
  iesireCuTva: number,
  intrareCuTva: number,
): { tip: 'incarca' | 'descarca' | ''; valoare: number } {
  const { tip, valoare: value } = diferenta(iesireCuTva, intrareCuTva);
  if (tip === 'plus') return { tip: 'incarca', valoare: value };
  if (tip === 'minus') return { tip: 'descarca', valoare: value };
  return { tip: '', valoare: 0 };
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd frontend && npm test`
Expected: PASS — 11 tests across two files.

- [ ] **Step 6: Write the API wrapper**

Create `frontend/src/api.ts`:

```ts
import {
  DeleteDocument,
  GetDocument,
  GetSettings,
  ListDocuments,
  ListProducts,
  NewDocumentDraft,
  SaveDocument,
  SaveProducts,
  SaveSettings,
} from '../wailsjs/go/main/App';
import { model } from '../wailsjs/go/models';

export type Document = model.Document;
export type DocumentSummary = model.DocumentSummary;
export type IntrareRow = model.IntrareRow;
export type IesireRow = model.IesireRow;
export type Product = model.Product;
export type Settings = model.Settings;

export {
  DeleteDocument,
  GetDocument,
  GetSettings,
  ListDocuments,
  ListProducts,
  NewDocumentDraft,
  SaveDocument,
  SaveProducts,
  SaveSettings,
};

/** Shows a Go-side error to the user in Romanian. */
export function showError(prefix: string, err: unknown): void {
  const message = err instanceof Error ? err.message : String(err);
  window.alert(`${prefix}: ${message}`);
}
```

- [ ] **Step 7: Write the router**

Create `frontend/src/router.ts`:

```ts
export type Route = {
  pattern: RegExp;
  render: (outlet: HTMLElement, ...params: string[]) => void | Promise<void>;
};

let currentRoutes: Route[] = [];
let currentOutlet: HTMLElement | null = null;

/** Navigates to a hash route, forcing a re-render even if the hash is unchanged. */
export function navigate(hash: string): void {
  if (window.location.hash === hash) {
    void handleRoute();
    return;
  }
  window.location.hash = hash;
}

async function handleRoute(): Promise<void> {
  if (!currentOutlet) return;
  const hash = window.location.hash || '#/document/new';

  for (const route of currentRoutes) {
    const match = route.pattern.exec(hash);
    if (match) {
      await route.render(currentOutlet, ...match.slice(1));
      return;
    }
  }
  currentOutlet.innerHTML = '<p class="empty">Pagina nu a fost gasita.</p>';
}

/** Installs the routes and renders the current hash. */
export function startRouter(routes: Route[], outlet: HTMLElement): void {
  currentRoutes = routes;
  currentOutlet = outlet;
  window.addEventListener('hashchange', () => {
    void handleRoute();
  });
  void handleRoute();
}
```

- [ ] **Step 8: Write the sidebar**

Create `frontend/src/sidebar.ts`:

```ts
import { ListDocuments, showError } from './api';
import { formatDateRO } from './format';
import { navigate } from './router';

/** Renders the sidebar: new-document button, document history, settings link. */
export async function renderSidebar(el: HTMLElement): Promise<void> {
  let documents;
  try {
    documents = await ListDocuments();
  } catch (err) {
    showError('Nu s-a putut incarca lista de documente', err);
    documents = [];
  }

  const items = documents
    .map(
      (doc) => `
        <li>
          <a class="doc-link" href="#/document/${doc.id}">
            <span class="doc-nr">NR ${doc.nr}</span>
            <span class="doc-meta">${formatDateRO(doc.data)}${doc.gestiune ? ` — ${escapeHtml(doc.gestiune)}` : ''}</span>
          </a>
        </li>`,
    )
    .join('');

  el.innerHTML = `
    <button class="btn btn-primary" id="new-doc">+ Document nou</button>
    ${documents.length > 0 ? `<ul class="doc-list">${items}</ul>` : ''}
    <a class="settings-link" href="#/setari">Setări</a>
  `;

  el.querySelector<HTMLButtonElement>('#new-doc')!.addEventListener('click', () => {
    navigate('#/document/new');
  });

  markActive(el);
  window.addEventListener('hashchange', () => markActive(el));
}

function markActive(el: HTMLElement): void {
  el.querySelectorAll('a').forEach((link) => {
    link.classList.toggle('active', link.getAttribute('href') === window.location.hash);
  });
}

/** Escapes text that goes into an innerHTML template. */
export function escapeHtml(value: string): string {
  const div = document.createElement('div');
  div.textContent = value;
  return div.innerHTML;
}
```

- [ ] **Step 9: Replace main.ts with the shell**

Replace the entire contents of `frontend/src/main.ts`:

```ts
import './style.css';
import { renderSidebar } from './sidebar';
import { startRouter } from './router';

document.querySelector<HTMLDivElement>('#app')!.innerHTML = `
  <aside class="sidebar" id="sidebar"></aside>
  <main class="main" id="outlet"></main>
`;

const sidebar = document.getElementById('sidebar') as HTMLElement;
const outlet = document.getElementById('outlet') as HTMLElement;

/** Re-reads the document history; called after any save or delete. */
export function refreshSidebar(): Promise<void> {
  return renderSidebar(sidebar);
}

void refreshSidebar();

startRouter(
  [
    {
      pattern: /^#\/document\/new$/,
      render: (el) => {
        el.innerHTML = '<p class="empty">Document nou (in curs de implementare)</p>';
      },
    },
    {
      pattern: /^#\/document\/(\d+)$/,
      render: (el, id) => {
        el.innerHTML = `<p class="empty">Document ${id} (in curs de implementare)</p>`;
      },
    },
    {
      pattern: /^#\/setari$/,
      render: (el) => {
        el.innerHTML = '<p class="empty">Setări (in curs de implementare)</p>';
      },
    },
  ],
  outlet,
);
```

- [ ] **Step 10: Replace style.css**

Replace the entire contents of `frontend/src/style.css`:

```css
:root {
  --bg: #f5f6f8;
  --panel: #ffffff;
  --border: #d5d8de;
  --text: #1b2636;
  --muted: #6b7280;
  --accent: #1b6ec2;
  --danger: #b42318;
  --readonly-bg: #f0f1f4;
  font-family: system-ui, -apple-system, "Segoe UI", Roboto, sans-serif;
  font-size: 14px;
  color: var(--text);
}

* {
  box-sizing: border-box;
}

body {
  margin: 0;
  background: var(--bg);
}

#app {
  display: grid;
  grid-template-columns: 260px 1fr;
  height: 100vh;
}

/* Sidebar */

.sidebar {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px 12px;
  background: var(--panel);
  border-right: 1px solid var(--border);
  overflow-y: auto;
}

.doc-list {
  list-style: none;
  margin: 0;
  padding: 0;
  flex: 1;
  overflow-y: auto;
}

.doc-link {
  display: block;
  padding: 8px 10px;
  border-radius: 6px;
  text-decoration: none;
  color: var(--text);
}

.doc-link:hover {
  background: var(--readonly-bg);
}

.doc-link.active {
  background: #e7f0fa;
}

.doc-nr {
  display: block;
  font-weight: 600;
}

.doc-meta {
  display: block;
  font-size: 12px;
  color: var(--muted);
}

.settings-link {
  margin-top: auto;
  padding: 8px 10px;
  border-radius: 6px;
  color: var(--text);
  text-decoration: none;
}

.settings-link:hover,
.settings-link.active {
  background: var(--readonly-bg);
}

/* Main panel */

.main {
  padding: 20px 24px 40px;
  overflow-y: auto;
}

.empty {
  color: var(--muted);
}

h1 {
  margin: 0 0 4px;
  font-size: 20px;
}

h2 {
  margin: 24px 0 8px;
  font-size: 15px;
}

/* Forms */

.header-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
  margin-bottom: 8px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.field label {
  font-size: 12px;
  color: var(--muted);
}

input,
select {
  padding: 6px 8px;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: var(--panel);
  color: var(--text);
  font: inherit;
}

input[readonly] {
  background: var(--readonly-bg);
  color: var(--muted);
}

input.num {
  text-align: right;
}

/* Tables */

table {
  width: 100%;
  border-collapse: collapse;
  background: var(--panel);
}

th,
td {
  border: 1px solid var(--border);
  padding: 4px 6px;
  text-align: left;
}

th {
  background: var(--readonly-bg);
  font-size: 12px;
  font-weight: 600;
}

td.num,
th.num {
  text-align: right;
}

td input {
  width: 100%;
  border: none;
  padding: 2px 4px;
  background: transparent;
}

td input:focus {
  outline: 2px solid var(--accent);
  border-radius: 2px;
}

tfoot td {
  font-weight: 600;
  background: var(--readonly-bg);
}

/* Buttons */

.btn {
  padding: 7px 14px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--panel);
  color: var(--text);
  font: inherit;
  cursor: pointer;
}

.btn:hover {
  background: var(--readonly-bg);
}

.btn-primary {
  background: var(--accent);
  border-color: var(--accent);
  color: #fff;
}

.btn-primary:hover {
  filter: brightness(1.08);
}

.btn-danger {
  color: var(--danger);
  border-color: var(--danger);
}

.btn-row {
  display: flex;
  gap: 8px;
  margin-top: 20px;
}

.btn-icon {
  padding: 2px 8px;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: var(--panel);
  cursor: pointer;
}

.table-actions {
  margin-top: 6px;
}

.footer-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 12px;
  margin-top: 20px;
}

.inline-field {
  display: flex;
  gap: 6px;
}

.inline-field select {
  flex: 0 0 110px;
}

.inline-field input {
  flex: 1;
}
```

- [ ] **Step 11: Verify the shell builds and runs**

Run: `cd frontend && npm run build`
Expected: `tsc` reports no errors and Vite writes `dist/`.

Run: `wails dev`
Expected: the window opens with a left sidebar showing "+ Document nou" and "Setări", no document list (empty history), and the main panel showing the placeholder text. Close the window.

- [ ] **Step 12: Commit**

```bash
git add frontend/package.json frontend/package-lock.json frontend/src
git commit -m "feat: add frontend shell, router, sidebar and calc/format helpers"
```

---

### Task 6: Document form view

The core screen. Renders both tables with live-computed value columns, saves, and deletes.

**Files:**
- Create: `frontend/src/views/document.ts`
- Modify: `frontend/src/main.ts` (replace the three placeholder routes)

**Interfaces:**
- Consumes: `api.ts`, `calc.ts`, `format.ts`, `router.ts`, `sidebar.ts` from Task 5; the bound methods from Task 4.
- Produces: `renderDocumentView(outlet: HTMLElement, id: string | undefined, refreshSidebar: () => Promise<void>): Promise<void>` — `id` is `undefined` for a new document.

State handling: the view keeps a mutable `doc` object, re-reads it from the inputs on every `input` event, recomputes the value columns and totals in place (without re-rendering the whole table, so focus and caret position survive), and only re-renders wholly when rows are added or removed.

- [ ] **Step 1: Create the document view**

Create `frontend/src/views/document.ts`:

```ts
import {
  DeleteDocument,
  Document,
  GetDocument,
  GetSettings,
  NewDocumentDraft,
  SaveDocument,
  showError,
} from '../api';
import { diferenta, incarcaDescarca, totals, valoare } from '../calc';
import { formatNumber, parseNumber } from '../format';
import { navigate } from '../router';
import { escapeHtml } from '../sidebar';

/** Renders the document form. Pass no id for a new document. */
export async function renderDocumentView(
  outlet: HTMLElement,
  id: string | undefined,
  refreshSidebar: () => Promise<void>,
): Promise<void> {
  let doc: Document;
  let unitate = '';

  try {
    const [loaded, settings] = await Promise.all([
      id === undefined ? NewDocumentDraft() : GetDocument(Number(id)),
      GetSettings(),
    ]);
    doc = loaded;
    unitate = settings.unitateNume;
  } catch (err) {
    showError('Nu s-a putut incarca documentul', err);
    outlet.innerHTML = '<p class="empty">Documentul nu a putut fi incarcat.</p>';
    return;
  }

  renderAll();

  function renderAll(): void {
    outlet.innerHTML = `
      <h1>${doc.id === 0 ? 'Document nou' : `Proces verbal NR ${doc.nr}`}</h1>

      <div class="header-grid">
        <div class="field">
          <label for="f-unitate">Unitate</label>
          <input id="f-unitate" value="${escapeHtml(unitate)}" readonly />
        </div>
        <div class="field">
          <label for="f-gestiune">Gestiune</label>
          <input id="f-gestiune" value="${escapeHtml(doc.gestiune)}" />
        </div>
        <div class="field">
          <label for="f-nr">NR</label>
          <input id="f-nr" class="num" type="number" min="1" step="1" value="${doc.nr}" />
        </div>
        <div class="field">
          <label for="f-data">Data</label>
          <input id="f-data" type="date" value="${escapeHtml(doc.data)}" />
        </div>
        <div class="field">
          <label for="f-referinta">Document referință</label>
          <input id="f-referinta" value="${escapeHtml(doc.documentReferinta)}" />
        </div>
      </div>

      <h2>Ce intră</h2>
      ${intrareTable()}
      <div class="table-actions">
        <button class="btn" id="add-intrare">+ Adaugă rând</button>
      </div>

      <h2>Ce iese</h2>
      ${iesireTable()}

      <div class="footer-grid">
        <div class="field">
          <label for="f-dif-tip">Diferență</label>
          <div class="inline-field">
            <select id="f-dif-tip">
              <option value=""></option>
              <option value="plus">plus</option>
              <option value="minus">minus</option>
            </select>
            <input id="f-dif-val" class="num" value="${formatNumber(doc.diferentaValoare)}" />
          </div>
        </div>
        <div class="field">
          <label for="f-id-tip">Suma cu care se … gestiunea</label>
          <div class="inline-field">
            <select id="f-id-tip">
              <option value=""></option>
              <option value="incarca">încarcă</option>
              <option value="descarca">descarcă</option>
            </select>
            <input id="f-id-val" class="num" value="${formatNumber(doc.incarcaDescarcaValoare)}" />
          </div>
        </div>
        <div class="field">
          <label for="f-gestionar">Gestionar</label>
          <input id="f-gestionar" value="${escapeHtml(doc.gestionar)}" />
        </div>
        <div class="field">
          <label for="f-calculator">Calculator</label>
          <input id="f-calculator" value="${escapeHtml(doc.calculator)}" />
        </div>
        <div class="field">
          <label for="f-vizat">Vizat compartiment producție</label>
          <input id="f-vizat" value="${escapeHtml(doc.vizatCompartimentProductie)}" />
        </div>
      </div>

      <div class="btn-row">
        <button class="btn btn-primary" id="save">Salvează</button>
        <button class="btn" id="print" ${doc.id === 0 ? 'disabled title="Salvează documentul întâi"' : ''}>Printează (PDF)</button>
        ${doc.id === 0 ? '' : '<button class="btn btn-danger" id="delete">Șterge</button>'}
      </div>
    `;

    (outlet.querySelector('#f-dif-tip') as HTMLSelectElement).value = doc.diferentaTip;
    (outlet.querySelector('#f-id-tip') as HTMLSelectElement).value = doc.incarcaDescarcaTip;

    wireEvents();
    recompute();
  }

  function intrareTable(): string {
    const rows = doc.intrare
      .map(
        (row, i) => `
        <tr data-table="intrare" data-index="${i}">
          <td>${i + 1}</td>
          <td><input data-field="denumire" value="${escapeHtml(row.denumire)}" /></td>
          <td><input data-field="um" value="${escapeHtml(row.um)}" /></td>
          <td class="num"><input class="num" data-field="cantitate" value="${row.cantitate || ''}" /></td>
          <td class="num"><input class="num" data-field="pretFaraTva" value="${row.pretFaraTva || ''}" /></td>
          <td class="num" data-out="valoareFaraTva"></td>
          <td class="num"><input class="num" data-field="pretCuTva" value="${row.pretCuTva || ''}" /></td>
          <td class="num" data-out="valoareCuTva"></td>
          <td><button class="btn-icon" data-remove="${i}" title="Șterge rândul">✕</button></td>
        </tr>`,
      )
      .join('');

    return `
      <table id="t-intrare">
        <thead>
          <tr>
            <th>Nr. crt.</th><th>Denumire produs</th><th>U/M</th>
            <th class="num">Cantitate</th><th class="num">Preț fără TVA</th><th class="num">Valoare</th>
            <th class="num">Preț cu TVA</th><th class="num">Valoare</th><th></th>
          </tr>
        </thead>
        <tbody>${rows}</tbody>
        <tfoot>
          <tr>
            <td colspan="3">Total</td>
            <td class="num" data-total="intrare-cantitate"></td>
            <td></td>
            <td class="num" data-total="intrare-faraTva"></td>
            <td></td>
            <td class="num" data-total="intrare-cuTva"></td>
            <td></td>
          </tr>
        </tfoot>
      </table>`;
  }

  function iesireTable(): string {
    const rows = doc.iesire
      .map(
        (row, i) => `
        <tr data-table="iesire" data-index="${i}">
          <td>${i + 1}</td>
          <td><input data-field="denumire" value="${escapeHtml(row.denumire)}" /></td>
          <td><input data-field="um" value="${escapeHtml(row.um)}" /></td>
          <td class="num"><input class="num" data-field="cantitate" value="${row.cantitate || ''}" /></td>
          <td class="num"><input class="num" data-field="pretFaraTva" value="${row.pretFaraTva || ''}" /></td>
          <td class="num" data-out="valoareFaraTva"></td>
          <td class="num"><input class="num" data-field="pretCuTva" value="${row.pretCuTva || ''}" /></td>
          <td class="num" data-out="valoareCuTva"></td>
        </tr>`,
      )
      .join('');

    return `
      <table id="t-iesire">
        <thead>
          <tr>
            <th>Nr. crt.</th><th>Denumire produs</th><th>U/M</th>
            <th class="num">Cantitate</th><th class="num">Preț fără TVA</th><th class="num">Valoare</th>
            <th class="num">Preț cu TVA</th><th class="num">Valoare</th>
          </tr>
        </thead>
        <tbody>${rows}</tbody>
        <tfoot>
          <tr>
            <td colspan="3">Total</td>
            <td class="num" data-total="iesire-cantitate"></td>
            <td></td>
            <td class="num" data-total="iesire-faraTva"></td>
            <td></td>
            <td class="num" data-total="iesire-cuTva"></td>
          </tr>
        </tfoot>
      </table>`;
  }

  function wireEvents(): void {
    outlet.addEventListener('input', onInput);

    outlet.querySelector('#add-intrare')!.addEventListener('click', () => {
      readForm();
      doc.intrare.push({
        id: 0,
        pozitie: doc.intrare.length,
        denumire: '',
        um: 'Kg',
        cantitate: 0,
        pretFaraTva: 0,
        pretCuTva: 0,
      });
      renderAll();
    });

    outlet.querySelectorAll<HTMLButtonElement>('[data-remove]').forEach((btn) => {
      btn.addEventListener('click', () => {
        readForm();
        doc.intrare.splice(Number(btn.dataset.remove), 1);
        renderAll();
      });
    });

    outlet.querySelector('#save')!.addEventListener('click', () => void onSave());

    const deleteBtn = outlet.querySelector('#delete');
    if (deleteBtn) deleteBtn.addEventListener('click', () => void onDelete());
  }

  function onInput(event: Event): void {
    const target = event.target as HTMLElement;
    readForm();

    // Retyping either total means the user is overriding the computed value, so
    // only refresh the footer when the edit came from somewhere else.
    const isFooterOverride =
      target.id === 'f-dif-val' ||
      target.id === 'f-dif-tip' ||
      target.id === 'f-id-val' ||
      target.id === 'f-id-tip';
    recompute(!isFooterOverride);
  }

  /** Copies every input's current value back into doc. */
  function readForm(): void {
    doc.gestiune = value('#f-gestiune');
    doc.nr = Number(value('#f-nr')) || 0;
    doc.data = value('#f-data');
    doc.documentReferinta = value('#f-referinta');
    doc.diferentaTip = value('#f-dif-tip');
    doc.diferentaValoare = parseNumber(value('#f-dif-val'));
    doc.incarcaDescarcaTip = value('#f-id-tip');
    doc.incarcaDescarcaValoare = parseNumber(value('#f-id-val'));
    doc.gestionar = value('#f-gestionar');
    doc.calculator = value('#f-calculator');
    doc.vizatCompartimentProductie = value('#f-vizat');

    outlet.querySelectorAll<HTMLTableRowElement>('tr[data-table]').forEach((tr) => {
      const index = Number(tr.dataset.index);
      const row = tr.dataset.table === 'intrare' ? doc.intrare[index] : doc.iesire[index];
      if (!row) return;
      row.pozitie = index;
      row.denumire = fieldValue(tr, 'denumire');
      row.um = fieldValue(tr, 'um');
      row.cantitate = parseNumber(fieldValue(tr, 'cantitate'));
      row.pretFaraTva = parseNumber(fieldValue(tr, 'pretFaraTva'));
      row.pretCuTva = parseNumber(fieldValue(tr, 'pretCuTva'));
    });
  }

  /** Refreshes the computed cells, and the footer unless the user overrode it. */
  function recompute(refreshFooter = true): void {
    outlet.querySelectorAll<HTMLTableRowElement>('tr[data-table]').forEach((tr) => {
      const index = Number(tr.dataset.index);
      const row = tr.dataset.table === 'intrare' ? doc.intrare[index] : doc.iesire[index];
      if (!row) return;
      setOut(tr, 'valoareFaraTva', valoare(row.cantitate, row.pretFaraTva));
      setOut(tr, 'valoareCuTva', valoare(row.cantitate, row.pretCuTva));
    });

    const intrare = totals(doc.intrare);
    const iesire = totals(doc.iesire);
    setTotal('intrare-cantitate', intrare.cantitate);
    setTotal('intrare-faraTva', intrare.valoareFaraTva);
    setTotal('intrare-cuTva', intrare.valoareCuTva);
    setTotal('iesire-cantitate', iesire.cantitate);
    setTotal('iesire-faraTva', iesire.valoareFaraTva);
    setTotal('iesire-cuTva', iesire.valoareCuTva);

    if (!refreshFooter) return;

    const dif = diferenta(iesire.valoareCuTva, intrare.valoareCuTva);
    const inc = incarcaDescarca(iesire.valoareCuTva, intrare.valoareCuTva);
    doc.diferentaTip = dif.tip;
    doc.diferentaValoare = dif.valoare;
    doc.incarcaDescarcaTip = inc.tip;
    doc.incarcaDescarcaValoare = inc.valoare;
    (outlet.querySelector('#f-dif-tip') as HTMLSelectElement).value = dif.tip;
    (outlet.querySelector('#f-dif-val') as HTMLInputElement).value = formatNumber(dif.valoare);
    (outlet.querySelector('#f-id-tip') as HTMLSelectElement).value = inc.tip;
    (outlet.querySelector('#f-id-val') as HTMLInputElement).value = formatNumber(inc.valoare);
  }

  async function onSave(): Promise<void> {
    readForm();
    if (doc.nr <= 0) {
      window.alert('Completați numărul documentului (NR).');
      return;
    }
    if (!doc.data) {
      window.alert('Completați data documentului.');
      return;
    }
    try {
      const saved = await SaveDocument(doc);
      await refreshSidebar();
      navigate(`#/document/${saved.id}`);
    } catch (err) {
      showError('Documentul nu a putut fi salvat', err);
    }
  }

  async function onDelete(): Promise<void> {
    if (!window.confirm(`Ștergeți documentul NR ${doc.nr}?`)) return;
    try {
      await DeleteDocument(doc.id);
      await refreshSidebar();
      navigate('#/document/new');
    } catch (err) {
      showError('Documentul nu a putut fi șters', err);
    }
  }

  function value(selector: string): string {
    return (outlet.querySelector(selector) as HTMLInputElement | HTMLSelectElement).value;
  }

  function fieldValue(tr: HTMLTableRowElement, field: string): string {
    const input = tr.querySelector<HTMLInputElement>(`[data-field="${field}"]`);
    return input ? input.value : '';
  }

  function setOut(tr: HTMLTableRowElement, name: string, amount: number): void {
    const cell = tr.querySelector<HTMLTableCellElement>(`[data-out="${name}"]`);
    if (cell) cell.textContent = amount === 0 ? '' : formatNumber(amount);
  }

  function setTotal(name: string, amount: number): void {
    const cell = outlet.querySelector<HTMLTableCellElement>(`[data-total="${name}"]`);
    if (cell) cell.textContent = formatNumber(amount);
  }
}
```

- [ ] **Step 2: Wire the routes in main.ts**

In `frontend/src/main.ts`, add the import:

```ts
import { renderDocumentView } from './views/document';
```

and replace the two document routes (leave the `#/setari` placeholder for Task 7):

```ts
    {
      pattern: /^#\/document\/new$/,
      render: (el) => renderDocumentView(el, undefined, refreshSidebar),
    },
    {
      pattern: /^#\/document\/(\d+)$/,
      render: (el, id) => renderDocumentView(el, id, refreshSidebar),
    },
```

- [ ] **Step 3: Verify the build type-checks**

Run: `cd frontend && npm run build`
Expected: no `tsc` errors.

- [ ] **Step 4: Verify manually**

Run: `wails dev`

Walk through these and confirm each:
1. The app opens on a new document; Unitate reads "S.C. Largiana Carn S.R.L." and is greyed out; NR is 1; Data is today; the "Ce iese" table has 19 rows starting with "Pulpa fara os / Kg / … / 21.9".
2. Type `15` in Pulpa's Cantitate → its "Valoare" (cu TVA) column shows `328.50` and the table total updates.
3. Type `16.5` in Pulpa's Preț fără TVA → the first Valoare column shows `247.50`.
4. Type `8,5` (comma) in Ceafa fara os's Cantitate → shows `288.15`, proving comma input works.
5. Add an intrare row with Cantitate `162.20`, Preț cu TVA `14.03` → Diferență flips to `minus` and the value updates as you type.
6. Overwrite the Diferență value by hand → it stays overwritten while you type in that box.
7. Fill Gestiune "Magazin Bradet", click **Salvează** → the sidebar gains "NR 1 — <today> — Magazin Bradet" and the URL becomes `#/document/1`.
8. Click "+ Document nou" → NR is 2, Gestiune is prefilled "Magazin Bradet", and the intrare row is prefilled with the previous one.
9. Reopen NR 1 from the sidebar → every value is as saved. Click **Șterge**, confirm → it disappears from the sidebar.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/document.ts frontend/src/main.ts
git commit -m "feat: add the document form view"
```

---

### Task 7: Settings view

**Files:**
- Create: `frontend/src/views/setari.ts`
- Modify: `frontend/src/main.ts` (replace the `#/setari` placeholder route)

**Interfaces:**
- Consumes: `GetSettings`, `SaveSettings`, `ListProducts`, `SaveProducts` from `api.ts`.
- Produces: `renderSetariView(outlet: HTMLElement): Promise<void>`.

- [ ] **Step 1: Create the settings view**

Create `frontend/src/views/setari.ts`:

```ts
import {
  GetSettings,
  ListProducts,
  Product,
  SaveProducts,
  SaveSettings,
  Settings,
  showError,
} from '../api';
import { formatNumber, parseNumber } from '../format';
import { escapeHtml } from '../sidebar';

/** Renders the settings screen: unit name and the "ce iese" product list. */
export async function renderSetariView(outlet: HTMLElement): Promise<void> {
  let settings: Settings;
  let products: Product[];

  try {
    [settings, products] = await Promise.all([GetSettings(), ListProducts()]);
  } catch (err) {
    showError('Nu s-au putut încărca setările', err);
    outlet.innerHTML = '<p class="empty">Setările nu au putut fi încărcate.</p>';
    return;
  }

  renderAll();

  function renderAll(): void {
    const rows = products
      .map(
        (p, i) => `
        <tr data-index="${i}">
          <td>${i + 1}</td>
          <td><input data-field="denumire" value="${escapeHtml(p.denumire)}" /></td>
          <td><input data-field="um" value="${escapeHtml(p.um)}" /></td>
          <td class="num"><input class="num" data-field="pretCuTva" value="${formatNumber(p.pretCuTva)}" /></td>
          <td>
            <button class="btn-icon" data-up="${i}" title="Mută în sus" ${i === 0 ? 'disabled' : ''}>↑</button>
            <button class="btn-icon" data-down="${i}" title="Mută în jos" ${i === products.length - 1 ? 'disabled' : ''}>↓</button>
            <button class="btn-icon" data-remove="${i}" title="Șterge produsul">✕</button>
          </td>
        </tr>`,
      )
      .join('');

    outlet.innerHTML = `
      <h1>Setări</h1>

      <div class="header-grid">
        <div class="field">
          <label for="s-unitate">Unitate</label>
          <input id="s-unitate" value="${escapeHtml(settings.unitateNume)}" />
        </div>
        <div class="field">
          <label for="s-nextnr">Următorul NR</label>
          <input id="s-nextnr" class="num" type="number" min="1" step="1" value="${settings.nextNr}" />
        </div>
      </div>

      <h2>Produse (tabelul "ce iese")</h2>
      <table>
        <thead>
          <tr>
            <th>Nr. crt.</th><th>Denumire produs</th><th>U/M</th>
            <th class="num">Preț cu TVA</th><th></th>
          </tr>
        </thead>
        <tbody>${rows}</tbody>
      </table>
      <div class="table-actions">
        <button class="btn" id="add-product">+ Adaugă produs</button>
      </div>

      <div class="btn-row">
        <button class="btn btn-primary" id="save">Salvează</button>
      </div>
      <p class="empty" id="status"></p>
    `;

    wireEvents();
  }

  function wireEvents(): void {
    outlet.querySelector('#add-product')!.addEventListener('click', () => {
      readForm();
      products.push({ id: 0, denumire: '', um: 'Kg', pretCuTva: 0, ordine: products.length });
      renderAll();
    });

    outlet.querySelectorAll<HTMLButtonElement>('[data-up]').forEach((btn) => {
      btn.addEventListener('click', () => move(Number(btn.dataset.up), -1));
    });
    outlet.querySelectorAll<HTMLButtonElement>('[data-down]').forEach((btn) => {
      btn.addEventListener('click', () => move(Number(btn.dataset.down), 1));
    });
    outlet.querySelectorAll<HTMLButtonElement>('[data-remove]').forEach((btn) => {
      btn.addEventListener('click', () => {
        readForm();
        products.splice(Number(btn.dataset.remove), 1);
        renderAll();
      });
    });

    outlet.querySelector('#save')!.addEventListener('click', () => void onSave());
  }

  function move(index: number, delta: number): void {
    const target = index + delta;
    if (target < 0 || target >= products.length) return;
    readForm();
    const [row] = products.splice(index, 1);
    products.splice(target, 0, row);
    renderAll();
  }

  /** Copies every input's current value back into settings and products. */
  function readForm(): void {
    settings.unitateNume = (outlet.querySelector('#s-unitate') as HTMLInputElement).value;
    settings.nextNr = Number((outlet.querySelector('#s-nextnr') as HTMLInputElement).value) || 1;

    outlet.querySelectorAll<HTMLTableRowElement>('tbody tr[data-index]').forEach((tr) => {
      const product = products[Number(tr.dataset.index)];
      if (!product) return;
      product.denumire = tr.querySelector<HTMLInputElement>('[data-field="denumire"]')!.value;
      product.um = tr.querySelector<HTMLInputElement>('[data-field="um"]')!.value;
      product.pretCuTva = parseNumber(
        tr.querySelector<HTMLInputElement>('[data-field="pretCuTva"]')!.value,
      );
    });
  }

  async function onSave(): Promise<void> {
    readForm();
    if (products.some((p) => p.denumire.trim() === '')) {
      window.alert('Fiecare produs trebuie să aibă o denumire.');
      return;
    }
    try {
      await SaveSettings(settings);
      await SaveProducts(products);
      products = await ListProducts();
      renderAll();
      outlet.querySelector('#status')!.textContent = 'Setările au fost salvate.';
    } catch (err) {
      showError('Setările nu au putut fi salvate', err);
    }
  }
}
```

- [ ] **Step 2: Wire the route**

In `frontend/src/main.ts`, add the import:

```ts
import { renderSetariView } from './views/setari';
```

and replace the `#/setari` placeholder route:

```ts
    {
      pattern: /^#\/setari$/,
      render: (el) => renderSetariView(el),
    },
```

- [ ] **Step 3: Verify the build type-checks**

Run: `cd frontend && npm run build`
Expected: no `tsc` errors.

- [ ] **Step 4: Verify manually**

Run: `wails dev`

1. Click **Setări** → Unitate shows the company name; the product table lists all 19 in order.
2. Change "Fleica" price to `31.00`, move "Sorici" up one place, add a product "Test / Kg / 1.00", click **Salvează** → the status line says "Setările au fost salvate."
3. Click "+ Document nou" → the ieșire table reflects the new order, the new price and the added product.
4. Return to Setări, delete "Test", save. Reopen the previously saved document → its rows still show their saved denumire and price (snapshot survived).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/setari.ts frontend/src/main.ts
git commit -m "feat: add the settings view for unit name and products"
```

---

### Task 8: PDF export

Renders the paper layout to PDF, asks where to save it, and opens it with the OS default handler.

**Files:**
- Create: `internal/pdfdoc/fold.go`
- Create: `internal/pdfdoc/pdf.go`
- Test: `internal/pdfdoc/pdf_test.go`
- Modify: `app.go` (add `ExportPDF`)
- Modify: `frontend/src/api.ts` (export `ExportPDF`)
- Modify: `frontend/src/views/document.ts` (wire the Printează button)
- Regenerate: `frontend/wailsjs`

**Interfaces:**
- Consumes: `model.Document`, `calc.*`.
- Produces: `pdfdoc.Fold(string) string`, `pdfdoc.Render(doc model.Document, unitateNume string) ([]byte, error)`; bound method `(*App).ExportPDF(id int64) (string, error)` returning the saved path, or `""` when the user cancels the dialog.

The core fpdf fonts are Latin-1, so Romanian diacritics must be folded to ASCII before being drawn. The paper form's own labels are already diacritic-free, so only user-entered text needs folding.

- [ ] **Step 1: Add the PDF dependency**

```bash
go get github.com/go-pdf/fpdf@latest
go mod tidy
CGO_ENABLED=0 go build ./...
```
Expected: builds cleanly.

- [ ] **Step 2: Write the failing PDF tests**

Create `internal/pdfdoc/pdf_test.go`:

```go
package pdfdoc

import (
	"bytes"
	"testing"

	"proces-verbal-transare/internal/model"
)

func TestFold(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Măcelărie", "Macelarie"},
		{"Șerban Țugui", "Serban Tugui"},
		{"şerban ţugui", "serban tugui"}, // cedilla variants
		{"Âîăâ", "Aiaa"},
		{"Magazin Bradet", "Magazin Bradet"},
		{"", ""},
	}
	for _, c := range cases {
		if got := Fold(c.in); got != c.want {
			t.Errorf("Fold(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRenderProducesAPDF(t *testing.T) {
	doc := model.Document{
		Nr:                         193,
		Data:                       "2026-09-03",
		Gestiune:                   "Magazin Brădet",
		DocumentReferinta:          "fact. 6721411/03.09.26",
		DiferentaTip:               "plus",
		DiferentaValoare:           2812.30,
		IncarcaDescarcaTip:         "incarca",
		IncarcaDescarcaValoare:     2812.30,
		Gestionar:                  "Ion Popescu",
		Calculator:                 "Maria Ionescu",
		VizatCompartimentProductie: "Șef producție",
		Intrare: []model.IntrareRow{
			{Pozitie: 0, Denumire: "Carcasa porc f cap", UM: "Kg", Cantitate: 162.20, PretFaraTVA: 12.50, PretCuTVA: 14.03},
		},
		Iesire: []model.IesireRow{
			{Pozitie: 0, Denumire: "Pulpa fara os", UM: "Kg", PretCuTVA: 21.9, Cantitate: 15, PretFaraTVA: 16.5},
			{Pozitie: 1, Denumire: "Muschiulet", UM: "Kg", PretCuTVA: 39.5, Cantitate: 1.5},
		},
	}

	out, err := Render(doc, "S.C. Largiana Carn S.R.L.")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Errorf("output does not start with %%PDF-: %q", out[:min(8, len(out))])
	}
	if len(out) < 1000 {
		t.Errorf("len(out) = %d, want a non-trivial PDF", len(out))
	}
}

func TestRenderEmptyDocument(t *testing.T) {
	out, err := Render(model.Document{Nr: 1, Data: "2026-09-07"}, "Unitate")
	if err != nil {
		t.Fatalf("Render on an empty document: %v", err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Error("output is not a PDF")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/pdfdoc/...`
Expected: FAIL — package does not exist.

- [ ] **Step 4: Write the fold helper**

Create `internal/pdfdoc/fold.go`:

```go
// Package pdfdoc renders a proces verbal to a PDF mirroring the paper form.
package pdfdoc

import "strings"

// diacritics maps the Romanian letters to their ASCII equivalents. Both the
// comma-below (correct) and cedilla (legacy) forms of s and t are covered.
var diacritics = strings.NewReplacer(
	"ă", "a", "Ă", "A",
	"â", "a", "Â", "A",
	"î", "i", "Î", "I",
	"ș", "s", "Ș", "S",
	"ş", "s", "Ş", "S",
	"ț", "t", "Ț", "T",
	"ţ", "t", "Ţ", "T",
)

// Fold replaces Romanian diacritics with plain ASCII. The core PDF fonts are
// Latin-1, which cannot represent them, and the paper form is written without
// diacritics anyway.
func Fold(s string) string {
	return diacritics.Replace(s)
}
```

- [ ] **Step 5: Write the PDF renderer**

Create `internal/pdfdoc/pdf.go`:

```go
package pdfdoc

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/go-pdf/fpdf"

	"proces-verbal-transare/internal/calc"
	"proces-verbal-transare/internal/model"
)

// Column widths in mm, summing to the 190mm A4 printable width.
var colWidths = []float64{14, 62, 14, 22, 22, 22, 22, 22}

const (
	marginLeft = 10.0
	rowHeight  = 5.5
)

// Render draws doc onto an A4 page and returns the PDF bytes.
func Render(doc model.Document, unitateNume string) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(marginLeft, 10, marginLeft)
	pdf.SetAutoPageBreak(true, 12)
	pdf.AddPage()

	drawHeader(pdf, doc, unitateNume)
	drawTable(pdf, intrareCells(doc))
	pdf.Ln(6)
	drawTable(pdf, iesireCells(doc))
	drawFooter(pdf, doc)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("generare PDF: %w", err)
	}
	return buf.Bytes(), nil
}

func drawHeader(pdf *fpdf.Fpdf, doc model.Document, unitateNume string) {
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(110, 5, "Unitatea: "+Fold(unitateNume), "", 0, "L", false, 0, "")
	pdf.CellFormat(80, 5, "Se aproba", "", 1, "C", false, 0, "")

	pdf.CellFormat(110, 5, "Gestiunea: "+Fold(doc.Gestiune), "", 0, "L", false, 0, "")
	pdf.CellFormat(40, 5, "Administrator", "", 0, "C", false, 0, "")
	pdf.CellFormat(40, 5, "Contabil", "", 1, "C", false, 0, "")

	pdf.Ln(6)
	pdf.SetFont("Arial", "B", 13)
	pdf.CellFormat(190, 7, fmt.Sprintf("Proces verbal de Transare    NR %d", doc.Nr), "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(190, 6, "Intocmit in data de "+formatDate(doc.Data), "", 1, "L", false, 0, "")
	pdf.CellFormat(
		190, 6,
		"Din materia prima primita cu documentul nr "+Fold(doc.DocumentReferinta)+" sau * ______ urmatoarele sortimente",
		"", 1, "L", false, 0, "",
	)
	pdf.Ln(3)
}

// tableCells is a rendered table: a header row, body rows and a totals row.
type tableCells struct {
	header []string
	body   [][]string
	total  []string
}

func headerRow() []string {
	return []string{"Nr. Crt.", "Denumire produs", "U/M", "Cantitate", "Pret fara TVA", "Valoare", "Pret cu TVA", "Valoare"}
}

func intrareCells(doc model.Document) tableCells {
	body := make([][]string, 0, len(doc.Intrare))
	for i, r := range doc.Intrare {
		body = append(body, []string{
			strconv.Itoa(i + 1),
			Fold(r.Denumire),
			Fold(r.UM),
			num(r.Cantitate),
			num(r.PretFaraTVA),
			num(calc.ValoareFaraTVA(r.Cantitate, r.PretFaraTVA)),
			num(r.PretCuTVA),
			num(calc.ValoareCuTVA(r.Cantitate, r.PretCuTVA)),
		})
	}
	t := calc.TotalsIntrare(doc.Intrare)
	return tableCells{
		header: headerRow(),
		body:   body,
		total:  totalRow(t),
	}
}

func iesireCells(doc model.Document) tableCells {
	body := make([][]string, 0, len(doc.Iesire))
	for i, r := range doc.Iesire {
		body = append(body, []string{
			strconv.Itoa(i + 1),
			Fold(r.Denumire),
			Fold(r.UM),
			num(r.Cantitate),
			num(r.PretFaraTVA),
			num(calc.ValoareFaraTVA(r.Cantitate, r.PretFaraTVA)),
			num(r.PretCuTVA),
			num(calc.ValoareCuTVA(r.Cantitate, r.PretCuTVA)),
		})
	}
	t := calc.TotalsIesire(doc.Iesire)
	return tableCells{
		header: headerRow(),
		body:   body,
		total:  totalRow(t),
	}
}

func totalRow(t calc.Totals) []string {
	return []string{"", "Total", "", num(t.Cantitate), "", num(t.ValoareFaraTVA), "", num(t.ValoareCuTVA)}
}

func drawTable(pdf *fpdf.Fpdf, table tableCells) {
	pdf.SetFont("Arial", "B", 8)
	pdf.SetX(marginLeft)
	for i, cell := range table.header {
		pdf.CellFormat(colWidths[i], rowHeight, cell, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Arial", "", 8)
	for _, row := range table.body {
		pdf.SetX(marginLeft)
		for i, cell := range row {
			align := "L"
			if i == 0 || i >= 3 {
				align = "R"
			}
			pdf.CellFormat(colWidths[i], rowHeight, cell, "1", 0, align, false, 0, "")
		}
		pdf.Ln(-1)
	}

	pdf.SetFont("Arial", "B", 8)
	pdf.SetX(marginLeft)
	for i, cell := range table.total {
		align := "L"
		if i >= 3 {
			align = "R"
		}
		pdf.CellFormat(colWidths[i], rowHeight, cell, "1", 0, align, false, 0, "")
	}
	pdf.Ln(-1)
}

func drawFooter(pdf *fpdf.Fpdf, doc model.Document) {
	pdf.Ln(6)
	pdf.SetFont("Arial", "", 10)

	pdf.CellFormat(95, 6, fmt.Sprintf("Diferenta %s %s", tipLabel(doc.DiferentaTip), num(doc.DiferentaValoare)), "", 0, "L", false, 0, "")
	pdf.CellFormat(
		95, 6,
		fmt.Sprintf("Suma cu care se %s gestiunea %s", tipLabel(doc.IncarcaDescarcaTip), num(doc.IncarcaDescarcaValoare)),
		"", 1, "L", false, 0, "",
	)

	pdf.Ln(8)
	pdf.CellFormat(63, 6, "Gestionar : "+Fold(doc.Gestionar), "", 0, "L", false, 0, "")
	pdf.CellFormat(63, 6, "Calculator: "+Fold(doc.Calculator), "", 0, "L", false, 0, "")
	pdf.CellFormat(64, 6, "Vizat compartiment productie", "", 1, "L", false, 0, "")
	pdf.CellFormat(126, 6, "", "", 0, "L", false, 0, "")
	pdf.CellFormat(64, 6, Fold(doc.VizatCompartimentProductie), "", 1, "L", false, 0, "")
}

// tipLabel renders a tip as the paper form's wording, or "..... / ....." when
// the user left it blank.
func tipLabel(tip string) string {
	switch tip {
	case "plus", "minus", "incarca", "descarca":
		return tip
	default:
		return "............"
	}
}

// num renders a money/quantity value, leaving zero blank the way the paper
// form leaves unused cells empty.
func num(v float64) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatFloat(calc.Round2(v), 'f', 2, 64)
}

// formatDate turns an ISO date into the dd.mm.yyyy the form uses.
func formatDate(iso string) string {
	if len(iso) != 10 {
		return iso
	}
	return iso[8:10] + "." + iso[5:7] + "." + iso[0:4]
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/pdfdoc/... -v`
Expected: PASS — `TestFold`, `TestRenderProducesAPDF`, `TestRenderEmptyDocument`.

- [ ] **Step 7: Add ExportPDF to app.go**

Add these imports to `app.go`:

```go
	"fmt"
	"os"

	"github.com/pkg/browser"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"proces-verbal-transare/internal/pdfdoc"
```

`github.com/pkg/browser` is already an indirect dependency of Wails; promote it with `go mod tidy` after this step.

Append to `app.go`:

```go
// ExportPDF renders a saved document, asks the user where to put the PDF and
// opens it with the system default handler. It returns the saved path, or an
// empty string when the user cancels the dialog.
func (a *App) ExportPDF(id int64) (string, error) {
	doc, err := a.store.GetDocument(id)
	if err != nil {
		return "", err
	}
	settings, err := a.store.GetSettings()
	if err != nil {
		return "", err
	}

	data, err := pdfdoc.Render(doc, settings.UnitateNume)
	if err != nil {
		return "", err
	}

	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Salvează PDF",
		DefaultFilename: fmt.Sprintf("proces-verbal-%d-%s.pdf", doc.Nr, doc.Data),
		Filters: []runtime.FileFilter{
			{DisplayName: "Fișiere PDF (*.pdf)", Pattern: "*.pdf"},
		},
	})
	if err != nil {
		return "", fmt.Errorf("alegere fisier: %w", err)
	}
	if path == "" {
		return "", nil // user cancelled
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("scriere fisier PDF: %w", err)
	}
	if err := browser.OpenFile(path); err != nil {
		// The file is on disk; failing to open the viewer is not fatal.
		runtime.LogWarningf(a.ctx, "nu s-a putut deschide PDF-ul: %v", err)
	}
	return path, nil
}
```

Then run:

```bash
go mod tidy
go build ./...
```

- [ ] **Step 8: Regenerate the bindings and wire the button**

```bash
wails generate module
```

In `frontend/src/api.ts`, add `ExportPDF` to both the import list from `../wailsjs/go/main/App` and the `export { … }` block.

In `frontend/src/views/document.ts`, add `ExportPDF` to the import from `../api`, and register the handler inside `wireEvents()` right after the save handler:

```ts
    const printBtn = outlet.querySelector('#print') as HTMLButtonElement | null;
    if (printBtn && !printBtn.disabled) {
      printBtn.addEventListener('click', () => void onPrint());
    }
```

and add the handler alongside `onSave`:

```ts
  async function onPrint(): Promise<void> {
    try {
      const path = await ExportPDF(doc.id);
      if (path === '') return; // dialog cancelled
    } catch (err) {
      showError('PDF-ul nu a putut fi generat', err);
    }
  }
```

- [ ] **Step 9: Verify the build**

Run: `go test ./... && cd frontend && npm run build && npm test`
Expected: all Go tests pass, `tsc` is clean, all Vitest tests pass.

- [ ] **Step 10: Verify manually**

Run: `wails dev`

1. Open a saved document, click **Printează (PDF)** → a native save dialog appears with a default name like `proces-verbal-193-2026-09-03.pdf`.
2. Save it → the PDF opens in the system viewer.
3. Compare against the photo of the paper form: header block (Unitatea/Gestiunea left, Se aproba/Administrator/Contabil right), title with NR, "Intocmit in data de", the reference line, the two tables with the eight columns in the printed order, and the footer with Diferenta / Suma cu care se … gestiunea / Gestionar / Calculator / Vizat compartiment productie.
4. Cancel the dialog on a second attempt → no error dialog appears.
5. Confirm a document whose Gestiune contains diacritics (e.g. "Magazin Brădet") renders as "Magazin Bradet" rather than as mojibake.

- [ ] **Step 11: Commit**

```bash
git add go.mod go.sum internal/pdfdoc app.go frontend/src/api.ts frontend/src/views/document.ts frontend/wailsjs
git commit -m "feat: render and export the proces verbal as a PDF"
```

---

### Task 9: Cross-platform build verification and README

Confirms the no-CGO constraint actually holds end to end and documents how to build.

**Files:**
- Modify: `README.md`
- Modify: `.gitignore` (ignore the stray root binary)

- [ ] **Step 1: Verify the macOS build**

```bash
wails build
```
Expected: succeeds; `build/bin/proces-verbal-transare.app` exists.

- [ ] **Step 2: Verify the Windows cross-build**

```bash
wails build -platform windows/amd64
```
Expected: succeeds with no CGO/toolchain errors; `build/bin/proces-verbal-transare.exe` is rewritten.

If this fails with a CGO error, the cause is a dependency pulled in that is not pure Go — check `go list -deps ./... | grep -i sqlite` and confirm only `modernc.org/*` appears, never `mattn/go-sqlite3`.

- [ ] **Step 3: Confirm the release binaries carry no CGO**

```bash
CGO_ENABLED=0 go build ./... && echo "pure Go OK"
```
Expected: prints `pure Go OK`.

- [ ] **Step 4: Stop tracking the stray root binary**

The repo currently has a committed `proces-verbal-transare` binary at the root. Remove it and ignore it:

```bash
git rm --cached proces-verbal-transare
```

Append to `.gitignore`:

```
# Stray dev build at the repo root
/proces-verbal-transare
```

- [ ] **Step 5: Replace the README body**

Replace `README.md` with:

```markdown
# Proces Verbal de Transare

Aplicație desktop pentru completarea, arhivarea și tipărirea procesului verbal
de transare folosit de S.C. Largiana Carn S.R.L.

## Cerințe

- Go 1.25+
- Node 20+
- Wails CLI v2 (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

## Dezvoltare

```bash
wails dev
```

## Teste

```bash
go test ./...
cd frontend && npm test
```

## Build

```bash
wails build                          # macOS
wails build -platform windows/amd64  # Windows, de pe macOS
```

Ambele dependențe native (`modernc.org/sqlite`, `github.com/go-pdf/fpdf`) sunt
pur Go, deci cross-compilarea nu are nevoie de un toolchain Windows.

## Date

Baza de date SQLite se creează la prima pornire în directorul de configurare al
utilizatorului:

- macOS: `~/Library/Application Support/proces-verbal-transare/data.db`
- Windows: `%AppData%\proces-verbal-transare\data.db`

La prima pornire se populează lista celor 19 produse de pe formularul tipărit;
poate fi modificată din ecranul **Setări**.
```

- [ ] **Step 6: Run the full test suite one more time**

```bash
go test ./... && (cd frontend && npm test)
```
Expected: all Go and Vitest tests pass.

- [ ] **Step 7: Commit**

```bash
git add README.md .gitignore
git commit -m "docs: document build, test and data locations"
```

---

## Self-Review

**Spec coverage**

| Spec requirement | Task |
|---|---|
| Go backend bound via `App` methods, no HTTP | 4 |
| SQLite via `modernc.org/sqlite`, migrate on startup | 2 |
| DB in `os.UserConfigDir()` (open question default accepted) | 2 |
| PDF via `go-pdf/fpdf` | 8 |
| Vite/TS scaffold, no framework, hash router, sidebar + main | 5 |
| All schema tables as specified | 2, 3 |
| Derived `valoare_*` computed, not stored | 1, 6, 8 |
| Seed the 19 products in printed order with printed prices | 2 |
| Row/table calculations, diferență sign rule, pre-filled but editable | 1, 5, 6 |
| Sidebar: new-document button, history newest first, empty state | 5 |
| Header fields incl. read-only Unitate, remembered Gestiune, prefilled NR/Data | 4, 6 |
| Table 1 dynamic rows, prefilled from last document | 4, 6 |
| Table 2 from `products` in `ordine`, editable prices | 4, 6 |
| Footer fields | 6 |
| Salvează / Printează / Șterge with confirmation | 6, 8 |
| `next_nr` bumped on new NR | 4 |
| Setări: unitate_nume + product add/edit/reorder/remove | 7 |
| Ieșire snapshots survive product deletion | 3 (tested), 7 (verified) |
| Native save dialog then open with OS handler | 8 |
| No CGO, Windows cross-build works | 2, 8, 9 |
| Go unit tests for calc helpers and repository CRUD | 1, 2, 3 |
| Manual `wails dev` verification of the full flow | 5, 6, 7, 8 |

No gaps.

**Notes on deliberate additions beyond the spec**

- `settings.next_nr` is editable from Setări (the spec only names `unitate_nume`); it is one field on a screen that already exists and avoids the counter being un-fixable after a mistyped NR.
- Vitest is added so the TS mirror of the calculations is tested rather than assumed to match Go.
- The `Printează` button is disabled on an unsaved document, since the spec's PDF flow renders an already-saved document.

**Type consistency** — checked: Go `model` field names match the JSON tags the TS views read (`pretCuTva`, `pretFaraTva`, `valoareCuTva`, `documentReferinta`, `incarcaDescarcaTip`, `vizatCompartimentProductie`); `calc.Totals` fields match the TS `Totals` shape; `store` method names used in `app.go` match Task 2/3 definitions; `escapeHtml` is defined once in `sidebar.ts` and imported by both views.
