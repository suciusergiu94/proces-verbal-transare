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
