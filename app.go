package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pkg/browser"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"proces-verbal-transare/internal/appdir"
	"proces-verbal-transare/internal/calc"
	"proces-verbal-transare/internal/model"
	"proces-verbal-transare/internal/pdfdoc"
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

// ListDocuments returns the sidebar history, newest first.
func (a *App) ListDocuments() ([]model.DocumentSummary, error) {
	return a.store.ListDocuments()
}

// GetDocument loads one saved document.
func (a *App) GetDocument(id int64) (model.Document, error) {
	return a.store.GetDocument(id)
}

// ListTemplates returns every "ce iese" template with its products.
func (a *App) ListTemplates() ([]model.Template, error) {
	return a.store.ListTemplates()
}

// SaveTemplates replaces the stored templates with the given ones.
func (a *App) SaveTemplates(templates []model.Template) error {
	return a.store.SaveTemplates(templates)
}

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

// SaveDocument creates or updates a document. Creating one with a number at or
// above the current counter advances the counter past it. The counter bump is
// performed inside the store's own save transaction (see
// store.Store.SaveDocument), so the document and the counter are committed
// atomically: there is no window where a save can report failure for a
// document that was actually persisted.
func (a *App) SaveDocument(doc model.Document) (model.Document, error) {
	// Diferenta and "suma cu care se incarca/descarca" are derived from the two
	// tables, never entered by hand, so they are re-derived here rather than
	// trusted from the frontend. That keeps a stored document self-consistent
	// even for rows saved by an older build, and means the PDF always prints
	// the same numbers the form showed.
	intrare := calc.TotalsIntrare(doc.Intrare)
	iesire := calc.TotalsIesire(doc.Iesire)
	doc.DiferentaTip, doc.DiferentaValoare = calc.Diferenta(iesire.ValoareCuTVA, intrare.ValoareCuTVA)
	doc.IncarcaDescarcaTip, doc.IncarcaDescarcaValoare = calc.IncarcaDescarca(iesire.ValoareCuTVA, intrare.ValoareCuTVA)

	saved, err := a.store.SaveDocument(doc)
	if err != nil {
		return model.Document{}, err
	}
	return saved, nil
}

// DeleteDocument removes a document and its rows.
func (a *App) DeleteDocument(id int64) error {
	return a.store.DeleteDocument(id)
}

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
		return "", fmt.Errorf("alegere fișier: %w", err)
	}
	if path == "" {
		return "", nil // user cancelled
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("scriere fișier PDF: %w", err)
	}
	if err := browser.OpenFile(path); err != nil {
		// The file is on disk; failing to open the viewer is not fatal.
		runtime.LogWarningf(a.ctx, "nu s-a putut deschide PDF-ul: %v", err)
	}
	return path, nil
}
