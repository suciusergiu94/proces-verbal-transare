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
var colWidths = []float64{12, 58, 12, 21, 21, 22, 22, 22}

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

// drawTable draws a table's header, body and totals row. Row boundaries are
// checked explicitly (via GetY(), rather than fpdf's SetHeaderFunc) so that a
// row is never split across a page break, and so the column header only gets
// redrawn on the table's own overflow pages, not on every page of the
// document: SetHeaderFunc fires on every AddPage, including the ones drawn
// for the surrounding header/footer, which would need extra state to
// suppress there.
func drawTable(pdf *fpdf.Fpdf, table tableCells) {
	// The header row itself must not be split across a page break either: if
	// it were left unguarded, a table starting near the bottom margin could
	// have fpdf's own auto-break fire mid-header-row. Unlike ensureRowFits,
	// this does not also redraw the header on break — the caller draws it
	// right after, so redrawing here would duplicate it.
	ensureHeaderFits(pdf)
	drawTableHeader(pdf, table.header)

	pdf.SetFont("Arial", "", 8)
	for _, row := range table.body {
		ensureRowFits(pdf, table.header)
		pdf.SetFont("Arial", "", 8)
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

	ensureRowFits(pdf, table.header)
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

// drawTableHeader draws one instance of the column header row.
func drawTableHeader(pdf *fpdf.Fpdf, header []string) {
	pdf.SetFont("Arial", "B", 8)
	pdf.SetX(marginLeft)
	for i, cell := range header {
		pdf.CellFormat(colWidths[i], rowHeight, cell, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)
}

// ensureRowFits forces a page break, and redraws the column header on the new
// page, if a single row of rowHeight would not fit above the bottom margin.
// This is a no-op — no new page, no output — as long as the current page
// still has room, which keeps single-page documents byte-for-byte unchanged.
func ensureRowFits(pdf *fpdf.Fpdf, header []string) {
	_, pageHeight := pdf.GetPageSize()
	_, _, _, bottom := pdf.GetMargins()
	if pdf.GetY()+rowHeight > pageHeight-bottom {
		pdf.AddPage()
		drawTableHeader(pdf, header)
	}
}

// ensureHeaderFits forces a page break if the header row itself would not fit
// above the bottom margin. It does not draw the header — the caller does
// that immediately after — so it never duplicates it the way ensureRowFits'
// redraw-on-break would.
func ensureHeaderFits(pdf *fpdf.Fpdf) {
	_, pageHeight := pdf.GetPageSize()
	_, _, _, bottom := pdf.GetMargins()
	if pdf.GetY()+rowHeight > pageHeight-bottom {
		pdf.AddPage()
	}
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

// formatDate turns an ISO date into the dd/mm/yyyy the form uses.
func formatDate(iso string) string {
	if len(iso) != 10 {
		return iso
	}
	return iso[8:10] + "/" + iso[5:7] + "/" + iso[0:4]
}
