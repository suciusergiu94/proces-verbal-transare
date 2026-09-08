# Șabloane pentru tabelul "ce iese" — design

## Purpose

Today the app has exactly one "ce iese" product list, seeded with the 19 cuts
of a pig carcass, editable in Setări. It is a global singleton: every document
splits the same way, and "Salvează procentele noi" rewrites the one list there
is.

That works for a shop butchering pigs. It does not work for a shop that also
butchers calves — the cuts, the prices and above all the ratios are a different
profile entirely, and there is nowhere to put a second one.

This design turns that single list into a **template** (Romanian: *șablon*),
of which there can be as many as the user wants. A template is a name plus its
own "ce iese" list. Creating a document starts by picking one; the document
remembers which template it used, so "Salvează procentele noi" writes back to
that template rather than to a global list.

## Non-goals

- **No per-template TVA.** The TVA rate is the same for every kind of meat; it
  stays a single global default in Setări.
- **No per-template numbering.** The NR counter stays global: a document
  created from any template advances the one counter, so the numbering of the
  whole register stays continuous.
- **No per-template unit or gestiune.** Both stay global defaults.
- **Nothing changes on the printed PDF.** The template is a source for the
  form, not a field of the document; `pdfdoc` is untouched.
- **No template import/export or sharing between installs.**

## Terminology

- **Șablon / template** — a named "ce iese" profile: `nume` + its product list
  (denumire, U/M, preț cu TVA, % din intrare).
- **Product** — one row of a template's list. Products already exist; they gain
  an owning template.

## Data model

### One clean schema, no migration

The app is pre-release: there is no installed database whose contents have to
survive. So templates are introduced by **changing the schema in place** rather
than by migrating onto it, and the accumulated v2/v3/v4 upgrade machinery goes
with it. See *Removing the upgrade machinery* below.

`schemaSQL` declares the full shape, and a fresh install is created from it
directly:

```sql
CREATE TABLE IF NOT EXISTS templates (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  nume TEXT NOT NULL,
  ordine INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS products (
  ...                              -- as today
  template_id INTEGER NOT NULL REFERENCES templates(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS documents (
  ...                              -- as today
  template_id INTEGER REFERENCES templates(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_products_template ON products(template_id, ordine);
```

`products.template_id` is `ON DELETE CASCADE`: a template's list has no meaning
without the template.

`documents.template_id` is `ON DELETE SET NULL`, deliberately not CASCADE. A
stored document is self-contained — `document_iesire_rows` already snapshots
denumire, U/M and price at save time — so deleting a template must not delete
history. What the document loses is only the ability to write ratios back; see
*A document whose template was deleted* under **Frontend**.

Both constraints are enforced at runtime: `store.Open` already connects with
`_pragma=foreign_keys(1)`. So deleting a template row is enough — its products
go with it, and its documents survive with `template_id NULL`. `SaveTemplates`
does not need to delete products by hand.

### Removing the upgrade machinery

`migrate` currently carries three one-time upgrades, all of which existed to
carry a v1 database forward to v4. None of them has a database left to serve,
and all of them get in the way of a clean schema, so they are deleted:

- the `ALTER TABLE ... ADD COLUMN cota_tva` loop (v2);
- `backfillProcente` and its `procent_din_intrare` ALTER (v3);
- `backfillGestiune` and its `gestiune` ALTER (v4);
- the `hasColumn` helper, which has no other caller.

`procenteDinCantitati` **stays** — it is not migration code; `seedProcente`
derives the shipped ratios with it.

`internal/store/migrate_test.go` loses the six upgrade tests that cover the
deleted paths and keeps nothing else; the file is removed. The seed coverage
those tests overlapped with lives in `store_test.go` already and is extended
below.

What remains in `migrate` is: apply `schemaSQL`, stamp
`PRAGMA user_version = 5`, and — if the settings row is absent — seed.

**Pre-release reset.** A database stamped below 5 predates templates and has a
`products` table with no `template_id`, which every query would then fail on.
Rather than crash, `migrate` drops every table when it finds one and continues
into a fresh create-and-seed. This is guarded on `user_version < 5` and a
non-zero version, so it fires exactly once per pre-release install and never on
a new one. It is explicitly a pre-release convenience — commented as such, and
to be replaced by a real migration the first time a shipped database matters.

**This wipes the documents on any machine where a pre-release build has already
been installed** — the current dev database and the friend's, if they have run
the installer. Accepted: the app is not in production, and the value is in
seeding correctly from here on.

### Seeding a fresh install

The existing seed, plus templates:

1. Insert the settings row, as today.
2. Insert one template, `nume = "Carcasa Porc"`, `ordine = 0`.
3. Insert the 19 seed products with that `template_id`, in printed order, with
   the ratios `seedProcente` derives — unchanged except for the new column.
4. Insert the shipped proces verbal NR 1 with that `template_id`, linked to the
   seeded products exactly as today. Its "ce intră" row keeps its current
   denumire, `"Carcasa"`.

All of it stays in the one transaction it is in today, so a fresh install is
either fully seeded or not seeded at all.

The name "Carcasa Porc" is chosen to read as the first "ce intră" row of a
document created from it — which is exactly what `NewDocumentDraft` will do
with it for a template that has no documents yet.

### Go model

```go
// Template is one named "ce iese" profile.
type Template struct {
    ID       int64     `json:"id"`
    Nume     string    `json:"nume"`
    Ordine   int       `json:"ordine"`
    Products []Product `json:"products"`
}
```

`Product` gains `TemplateID int64` (`json:"templateId"`).
`Document` gains `TemplateID *int64` (`json:"templateId"`), nullable to mirror
the column.

`Settings` is unchanged.

## Backend API (Wails bindings on `App`)

| Before | After |
|---|---|
| `ListProducts() []Product` | `ListTemplates() []Template` — templates in `ordine`, each with its products in `ordine` |
| `SaveProducts([]Product)` | `SaveTemplates([]Template)` |
| `NewDocumentDraft() Document` | `NewDocumentDraft(templateID int64) Document` |

`GetSettings` / `SaveSettings` / `ListDocuments` / `GetDocument` /
`SaveDocument` / `DeleteDocument` / `ExportPDF` are unchanged.

`ListProducts` and `SaveProducts` are removed **from the Wails bindings**
rather than kept alongside the new pair. They have exactly two frontend call
sites (Setări, and the ratio write-back in the document view), both of which
move to the template-aware pair; leaving a global-list API bound would be an
invitation to write against the wrong one.

`store.Store` keeps an internal `listProducts(templateID int64)` used by
`ListTemplates` and by `NewDocumentDraft`. It is unexported from the bindings,
not from the package.

### `SaveTemplates` semantics

One transaction, mirroring today's `SaveProducts` diff, one level deeper:

- Templates with `ID == 0` are inserted; known IDs are updated
  (`nume`, `ordine` from slice position); IDs absent from the input are
  deleted, which cascades to their products.
- Within each template, products diff exactly as `SaveProducts` does today:
  `ID == 0` inserts, known IDs update, absent IDs delete, `ordine` reassigned
  from slice position. `template_id` is taken from the owning template, so a
  product cannot be smuggled into a template it does not belong to.
- **Refuses an empty slice** with a Romanian error: there must always be at
  least one template. The frontend also hides "Șterge" on the last remaining
  section, so this is a backstop, not the primary guard.
- Template names are not required to be unique. Two templates called "Porc"
  are the user's business; the dropdown shows both.
- A template name may not be blank — refused with a Romanian error, matching
  how a blank product denumire is already refused (frontend-side today; this
  moves the check to both sides).

Ratio validation (the 100% rule) stays in the frontend, where it is today.
The store does not enforce it. The rule belongs to the form that consumes the
ratios, not to storage, and a store-side refusal would make an intermediate
state — a template mid-edit, or one whose products were just reordered —
unwritable for no gain.

### `NewDocumentDraft(templateID)`

Same as today, with four changes:

1. Products come from `listProducts(templateID)` instead of the global list.
2. The "ce intră" shape is copied from the last document **of the same
   template** (`LastDocument(templateID)`), not from the last document overall.
   Copying a pig's "ce intră" shape onto a calf document would be wrong on the
   first row, which is the row that names what was butchered.
3. When that template has no previous document, the draft starts with a single
   "ce intră" row whose `denumire` is the template's name — "Carcasa Vitel" —
   rather than the blank row it starts with today. After the first saved
   document, rule 2 carries the same name forward, so the two rules agree in
   practice.
4. `TemplateID` is set on the draft.

An unknown `templateID` returns a Romanian error; the frontend routes on ids
that came from `ListTemplates`, so this is a guard against a stale bookmark.

### `SaveDocument`

Unchanged except that `template_id` is written through, on insert and on
update. NR allocation and the counter bump are untouched — the counter is
global, so numbering stays continuous across templates.

## Frontend

### Sidebar — picking a template

"+ Document nou" opens a small menu listing the templates in `ordine`. Picking
one navigates to `#/document/new/<templateId>`.

With exactly one template the menu is skipped: the button navigates straight to
that template's draft route, so an install that never creates a second template
behaves exactly as it does today.

The draft entry in the document list keeps working: `DRAFT_HASH` becomes a
prefix test (`#/document/new/`) rather than an equality test, and the draft
entry's `href` and label follow the template currently open — "Document nou"
with the template name as its meta line.

Route table gains `#/document/new/<id>`; the bare `#/document/new` is kept as a
redirect to the first template, so the router's existing empty-hash fallback
and any stale window state still land somewhere valid.

### Setări — all templates on one page

Layout, top to bottom:

1. **Setări generale** — unitate, gestiune implicită, următorul NR, cotă TVA
   implicită. Unchanged.
2. **Șabloane** — one collapsible section per template:
   - header: the template name in an editable input, the section's ratio total
     (`nnn,nnn %`, marked invalid unless exactly 100), and *Duplică* /
     *Șterge* buttons;
   - body: today's product table verbatim — denumire, U/M, preț cu TVA,
     % din intrare, move up/down, remove, "+ Adaugă produs".
   - *Șterge* asks for confirmation, naming the template. It is hidden when
     only one template remains.
   - *Duplică* appends a copy of the section (name + " (copie)", all products
     with `id: 0`) — the practical way to start a calf list from the pig one
     when the two share most of their structure. A new install adding a
     genuinely different profile uses "+ Șablon nou" instead.
3. **+ Șablon nou** — appends an empty section (blank name, no products),
   expanded, with focus in the name field.
4. **Salvează** — one button for the whole page.

Which sections are expanded is view state only; nothing about it is persisted.
On load, the first template is expanded and the rest collapsed, so a shop with
five templates does not open Setări on a wall of tables.

### Setări — validation on save

`Salvează` runs, in order:

1. Every template has a non-blank name → otherwise alert, expand and mark the
   offending section.
2. Every product in every template has a non-blank denumire → same treatment.
3. Cota TVA is not negative; no `procentDinIntrare` is negative → same messages
   as today, scoped to the section.
4. **Every template's ratios sum to exactly 100%** → otherwise the save is
   refused, the first offending section is expanded and marked, and the alert
   names the template and the gap, reusing today's wording:
   *„Șablonul «Carcasa Vitel»: procentele din intrare însumează 97,500 %, nu
   100 %. Mai trebuie repartizate 2,500 % (de obicei la deșeu)."*

A newly added, still-empty template therefore blocks saving until it is filled
in — a deliberate consequence of the single-button model, and the reason
*Duplică* exists.

Only when all checks pass: `SaveSettings(settings)` then
`SaveTemplates(templates)`, then reload from `ListTemplates()` and re-render,
as today.

### Document view

- Loads via `NewDocumentDraft(templateID)` for the draft route, `GetDocument`
  otherwise. Both come back carrying `templateId`.
- The `procente` map is built from that template's products
  (`ListTemplates()`, pick the document's template), not from a global list.
- **"Salvează procentele noi"** works as today, scoped: it reads the template's
  product list, computes each product's share from this document's quantities,
  and saves that template back through `SaveTemplates` with only that
  template's products changed. The confirmation text names the template:
  *„Salvați procentele noi? Procentele șablonului «Carcasa Porc» din Setări vor
  fi înlocuite cu cele rezultate din cantitățile de pe acest document."*
- **A document whose template was deleted** (`templateId == null`) renders
  normally — every printed value is on its own rows — but the margin +/−
  buttons and "Salvează procentele noi" are hidden, and the "ce iese"
  quantities are not refilled when the input quantity changes, because there
  are no ratios to refill them from. A short note under the "Ce iese" heading
  says so: *„Șablonul acestui document a fost șters; cantitățile nu se mai
  completează automat."*
- The header shows the template name next to Gestiune as a read-only field, so
  it is visible which profile a stored document was built from. It is not
  editable: changing a saved document's template is out of scope.

## Testing

### Go (`internal/store`)

Seeding is where the whole change is load-bearing now, so it gets the weight:

- **Fresh seed**: exactly one template, named "Carcasa Porc"; all 19 products
  on it, in printed order, with the ratios summing to exactly 100%; the seeded
  proces verbal NR 1 stamped with it and linked to those product ids.
- The existing seed assertions in `store_test.go` still hold — dated today,
  footer consistent with its rows, `next_nr` past it, no reseed after the
  document is deleted, prices and percentages intact.
- `PRAGMA user_version` is 5 on a fresh database.
- `Open` is idempotent: opening twice seeds once, and does not create a second
  template.
- **Pre-release reset**: a database stamped at 4 with the old shape is dropped
  and reseeded, and comes back with one template and the 19 products. A
  database stamped at 5 is left alone.
- `ListTemplates` returns templates in `ordine`, each with its products in
  `ordine`.
- `SaveTemplates`: insert / update / delete of templates; insert / update /
  delete of products within a template; deleting a template removes its
  products (FK cascade) and leaves its documents intact with `template_id NULL`.
- `SaveTemplates` refuses an empty slice and refuses a blank template name.
- `LastDocument(templateID)` ignores documents of other templates.

### Go (`app_test.go`)

- `NewDocumentDraft(id)` fills "ce iese" from that template only.
- With no previous document of that template, the draft's single "ce intră" row
  is named after the template.
- With a previous document of that template, its "ce intră" shape is copied and
  the previous document of *another* template is ignored.
- `NewDocumentDraft` on an unknown id returns an error.
- NR is shared: drafts from two different templates, saved in turn, take
  consecutive numbers.

### Frontend (vitest)

The project has no jsdom: vitest runs in node, and every existing test is over
pure logic (`calc`, `format`, `escapeHtml`, `currentHash`, and the CSS cascade
in `buttonHover.test.ts`). The view modules themselves have no tests today.
Rather than add a DOM environment for this change, the decisions that would
otherwise be buried in the views move into a new pure module,
`frontend/src/templates.ts`, and the tests go there. The views keep only
wiring.

`templates.ts` exports, and is tested on:

- `draftHash(templateId)` / `isDraftHash(hash)` / `draftTemplateId(hash)` —
  round-trip; `isDraftHash` true for `#/document/new` and
  `#/document/new/7`, false for `#/document/12` and `#/setari`;
  `draftTemplateId` returns `undefined` for the bare draft hash.
- `emptyTemplate()` — blank name, no products, `id: 0`.
- `duplicateTemplate(t)` — name gets " (copie)", the copy's `id` and every
  product `id` are 0, and the source is not mutated.
- `validateTemplates(templates)` — returns the first problem as
  `{ templateIndex, message }`, or `undefined`. Covered: blank template name;
  blank product denumire; negative ratio; ratios summing to less than 100 (the
  message names the template and the missing amount); ratios summing to more
  than 100; a template with no products at all; and a valid set returning
  `undefined`.
- `templatesCuProcenteNoi(templates, templateId, cantitatiPerProdus)` — the
  ratio write-back. Returns a new list in which only the named template's
  ratios changed, every other template deep-equal to its input, and a product
  the document did not produce set to 0. Returns `undefined` when the
  quantities are all zero.

`sidebar.test.ts` keeps `currentHash` and gains coverage that the draft route
is recognised for a templated hash.

## Risk

Deleting the upgrade machinery is a one-way door: after this, no database
written by an earlier build can be read, only discarded and reseeded. That is
the deliberate trade — the app is pre-release, and the alternative was carrying
a missing-foreign-key asymmetry and a hand-rolled cascade into every install
forever.

The obligation it creates is that the **seed** must be right, since it is now
the only path into a working database. Hence the weight on seed tests above.

## Open questions

None.
