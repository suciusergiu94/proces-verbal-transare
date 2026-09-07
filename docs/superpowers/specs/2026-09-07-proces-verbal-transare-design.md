# Proces Verbal de Transare — design

## Purpose

Digitize the paper "Proces verbal de transare" used by S.C. Largiana Carn
S.R.L. to record raw-meat intake ("ce intră") and its breakdown into cuts
("ce iese"). The app must let staff fill in a new document, keep a history
of past ones, and print/export a PDF that matches the paper layout. Must
build and run on both macOS and Windows (Wails cross-compile target).

Source references: two photos of the paper form (blank template + a filled
example, NR 193, dated 03.09.26) reviewed during brainstorming.

## Non-goals

- No multi-user accounts/auth — single local user, single SQLite file.
- No cloud sync — local-only persistence.
- No arbitrary product catalog for the "ce intră" table — it stays
  free-text per row (see below).

## Architecture

- **Backend**: Go, bound to the frontend via Wails' existing binding
  mechanism (methods on `App` in `app.go`). No REST/HTTP layer needed.
- **Database**: SQLite via `modernc.org/sqlite` (pure Go driver, no CGO) so
  cross-compiling to Windows from macOS stays simple. DB file lives next to
  the executable (or in the OS user-data dir — see Open Question below),
  created/migrated on startup if missing.
- **PDF generation**: `github.com/go-pdf/fpdf` (pure Go, no CGO), drawing a
  layout mirroring the paper form.
- **Frontend**: existing Vite/TypeScript scaffold, no framework added.
  Simple hash-based router. Layout: persistent left sidebar + main panel.
  All Romanian UI text.

## Data model (SQLite)

```sql
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
  data TEXT NOT NULL,               -- ISO date string
  gestiune TEXT NOT NULL,
  document_referinta TEXT NOT NULL DEFAULT '', -- "documentul nr ... sau ..."
  diferenta_tip TEXT NOT NULL DEFAULT '',      -- 'plus' | 'minus' | ''
  diferenta_valoare REAL NOT NULL DEFAULT 0,
  incarca_descarca_tip TEXT NOT NULL DEFAULT '', -- 'incarca' | 'descarca' | ''
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
  denumire TEXT NOT NULL,       -- snapshot, survives product edits/deletes
  um TEXT NOT NULL,
  pret_cu_tva REAL NOT NULL,    -- snapshot of product price at save time
  cantitate REAL NOT NULL DEFAULT 0,
  pret_fara_tva REAL NOT NULL DEFAULT 0
);
```

`valoare_fara_tva` and `valoare_cu_tva` per row are derived
(`cantitate × preț`) and not stored as separate columns — they're
computed on read/PDF-render from the stored `cantitate`/`preț` snapshot,
so results stay consistent even though nothing but the two price
columns is persisted.

Seed data: on first run, `products` is seeded with the 19 fixed items from
the paper form (Pulpa fara os … Deseu fara valoare) in their printed
order, with their printed `pret_cu_tva` defaults.

## Calculations

- Per row: `valoare_fara_tva = cantitate × pret_fara_tva`,
  `valoare_cu_tva = cantitate × pret_cu_tva`.
- Table totals: sum of each value column per table.
- TVA rate is implicit in the two price columns already being tracked
  separately (matches the paper form — no separate rate field needed).
- `Diferență` / `Suma cu care se încarcă/descarcă gestiunea`: computed as
  `total ieșire (valoare cu TVA) − total intrare (valoare cu TVA)`. A
  positive result is "plus" / "încarcă", negative is "minus" / "descarcă".
  Both fields are pre-filled from this computation but remain editable
  (matching that these are hand-annotated on the paper form and may need
  manual correction, e.g. rounding).

## Frontend flow

- **Sidebar** (always visible): "+ Document nou" button at top; below it,
  a scrollable list of saved documents ("NR 193 — 03.09.2026 — Magazin
  Bradet"), newest first. Empty state: just the button, no list.
- **Document form** (new or existing):
  - Header: Unitate (read-only, from settings), Gestiune (free text,
    remembers last value as default for a new doc), NR (prefilled from
    `settings.next_nr`, editable), Data (date picker, defaults today),
    "Document referință" free-text field.
  - Table 1 ("ce intră"): dynamic rows, add/remove buttons, free-text
    Denumire/U.M., numeric Cantitate/Preț fără TVA/Preț cu TVA, computed
    Valoare columns shown read-only. New document prefills these rows
    from the most recently saved document's intrare rows (editable/
    removable, not locked).
  - Table 2 ("ce iese"): fixed rows sourced from `products` (in `ordine`),
    Denumire/U.M./Preț cu TVA prefilled and editable per row, user enters
    Cantitate and Preț fără TVA, Valoare columns computed.
  - Footer: Diferență (plus/minus selector + value, prefilled/computed),
    Gestionar, Calculator, Vizat compartiment producție (free text/name
    fields), Suma cu care se încarcă/descarcă gestiunea (tip + value,
    prefilled/computed).
  - Actions: **Salvează** (creates or updates the document + rows, bumps
    `settings.next_nr` when a new NR was used), **Printează (PDF)**,
    and for an existing document, **Șterge** (delete, with confirmation).
- **Setări** (sidebar link): edit `unitate_nume`; manage `products`
  (add/edit/reorder/remove rows — a product used by past documents can
  still be removed since ieșire rows snapshot their own denumire/preț).

## PDF generation

`Printează` calls a bound Go method that renders the current document
(already saved) to a PDF via `go-pdf/fpdf`, mirroring the paper layout
(header block, table 1, table 2, footer block, all labels in Romanian).
Uses Wails' native save-file dialog to let the user pick the destination,
then opens the generated file with the OS default handler so the user's
normal PDF viewer print flow takes over.

## Cross-platform build

Both new Go dependencies (`modernc.org/sqlite`, `go-pdf/fpdf`) are pure Go
— no CGO, so `wails build -platform windows/amd64` from macOS keeps
working without a Windows toolchain.

## Testing

- Go: unit tests for the calculation helpers (row/table totals, diferență
  sign) and for the SQLite repository layer (CRUD round-trips) using an
  in-memory SQLite DB.
- Manual UI verification via `wails dev`: create a document, verify
  computed columns, save, confirm it appears in history, edit it, delete
  it, generate a PDF and confirm it opens.

## Open questions (defaults chosen, flag if wrong)

- DB file location: defaults to a per-OS user-data directory (e.g.
  `os.UserConfigDir()/proces-verbal-transare/data.db`) rather than next to
  the executable, so it survives app reinstalls/updates. Change if you'd
  rather keep it next to the binary.
