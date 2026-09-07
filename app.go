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
