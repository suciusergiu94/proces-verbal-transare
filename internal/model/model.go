// Package model holds the plain data structures shared by the store, the
// calculation helpers, the PDF renderer and the Wails bindings.
package model

// Settings is the single-row application configuration.
type Settings struct {
	UnitateNume string `json:"unitateNume"`
	NextNr      int    `json:"nextNr"`
	// CotaTVA is the TVA percentage new rows start from, e.g. 11 for the
	// Romanian food rate. It is a default, not a rule: every row carries its
	// own rate and may depart from this one.
	CotaTVA float64 `json:"cotaTva"`
	// Gestiune is the gestiune new documents start from. Like CotaTVA it is a
	// default, not a rule: every document carries its own and may depart from
	// this one.
	Gestiune string `json:"gestiune"`
}

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

// Product is one entry of a template's "ce iese" product list.
type Product struct {
	ID         int64   `json:"id"`
	TemplateID int64   `json:"templateId"`
	Denumire   string  `json:"denumire"`
	UM         string  `json:"um"`
	PretCuTVA  float64 `json:"pretCuTva"`
	// ProcentDinIntrare is the share of what goes in that this product comes
	// out as, in percent (9.248, not 0.09248), to three decimals. Across the
	// whole list it sums to 100: everything that goes in has to come out as
	// something, waste included. The form uses it to fill the "ce iese"
	// quantities from the carcass weight; it is never printed.
	ProcentDinIntrare float64 `json:"procentDinIntrare"`
	Ordine            int     `json:"ordine"`
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
	CotaTVA     float64 `json:"cotaTva"`
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
	CotaTVA     float64 `json:"cotaTva"`
}

// Document is a full proces verbal with both tables.
type Document struct {
	ID int64 `json:"id"`
	Nr int   `json:"nr"`
	// TemplateID is the template this document was created from. It is nil for
	// a document whose template has since been deleted: the rows below keep
	// their own denumire, UM and price, so the document stays complete, but it
	// no longer has ratios to refill its quantities from.
	TemplateID                 *int64       `json:"templateId"`
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
