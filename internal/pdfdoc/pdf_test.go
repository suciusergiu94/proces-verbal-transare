package pdfdoc

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"regexp"
	"strings"
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
		{"Ă", "A"},
		{"Î", "I"},
		{"Ş", "S"}, // uppercase cedilla s
		{"Ţ", "T"}, // uppercase cedilla t
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

// TestRenderContentSurvivesArithmeticAndFolding checks that the computed
// values and the ASCII-folded text actually reach the drawn PDF content
// streams, and that no raw diacritic bytes leak through unfolded.
func TestRenderContentSurvivesArithmeticAndFolding(t *testing.T) {
	doc := model.Document{
		Nr:       1,
		Data:     "2026-09-07",
		Gestiune: "Magazin Brădet",
		Iesire: []model.IesireRow{
			{Pozitie: 0, Denumire: "Pulpa fara os", UM: "Kg", Cantitate: 15, PretFaraTVA: 16.5, PretCuTVA: 21.9},
		},
	}

	out, err := Render(doc, "Unitate")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	text := extractPDFText(t, out)

	// 15 x 21.9 = 328.50, from calc.ValoareCuTVA.
	if !strings.Contains(text, "328.50") {
		t.Errorf("expected computed valoare 328.50 (15 x 21.9) in rendered text, got:\n%s", text)
	}
	// "Magazin Brădet" must reach the page folded to ASCII.
	if !strings.Contains(text, "Magazin Bradet") {
		t.Errorf("expected folded gestiune \"Magazin Bradet\" in rendered text, got:\n%s", text)
	}
	if strings.ContainsAny(text, "ăâîșțĂÂÎȘȚşţŞŢ") {
		t.Errorf("raw diacritic bytes survived into the rendered PDF text: %q", text)
	}
}

// TestRenderOverflowsToMultiplePages checks that a document with enough
// "ce iese" rows to overflow one page spans multiple pages, and that the
// table's column header is redrawn on the pages it continues onto rather
// than a row being cut off with no header above it.
func TestRenderOverflowsToMultiplePages(t *testing.T) {
	doc := model.Document{Nr: 2, Data: "2026-09-07"}
	for i := 0; i < 40; i++ {
		doc.Iesire = append(doc.Iesire, model.IesireRow{
			Pozitie:     i,
			Denumire:    fmt.Sprintf("Produs %d", i),
			UM:          "Kg",
			Cantitate:   1,
			PretFaraTVA: 8,
			PretCuTVA:   10,
		})
	}

	out, err := Render(doc, "Unitate")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	pages := countPages(out)
	if pages <= 1 {
		t.Fatalf("expected the 40-row iesire table to overflow onto multiple pages, got %d page(s)", pages)
	}

	text := extractPDFText(t, out)
	headerCount := strings.Count(text, "Nr. Crt.")
	// The intrare table's header draws once regardless, and the iesire
	// table's header is redrawn once per page it spans, so the header text
	// must appear at least once per page in the document.
	if headerCount < pages {
		t.Errorf("expected the column header to appear at least once per page (pages=%d), got %d occurrences", pages, headerCount)
	}
}

// extractPDFText pulls the text drawn via the Tj operator out of every
// FlateDecode content stream in a rendered PDF, in document order.
func extractPDFText(t *testing.T, data []byte) string {
	t.Helper()
	streamRe := regexp.MustCompile(`(?s)stream\r?\n(.*?)\r?\nendstream`)
	tjRe := regexp.MustCompile(`\(((?:\\.|[^()\\])*)\)\s*Tj`)

	var sb strings.Builder
	for _, m := range streamRe.FindAllSubmatch(data, -1) {
		dec, err := zlibDecompress(m[1])
		if err != nil {
			continue // not a compressed text stream (e.g. embedded font data)
		}
		for _, tm := range tjRe.FindAllSubmatch(dec, -1) {
			sb.WriteString(unescapePDFString(string(tm[1])))
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func zlibDecompress(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

// unescapePDFString undoes the small set of backslash escapes fpdf emits
// inside literal strings.
func unescapePDFString(s string) string {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			switch s[i] {
			case 'n':
				sb.WriteByte('\n')
			case 'r':
				sb.WriteByte('\r')
			case 't':
				sb.WriteByte('\t')
			default:
				sb.WriteByte(s[i])
			}
			continue
		}
		sb.WriteByte(s[i])
	}
	return sb.String()
}

// countPages counts the page objects ("/Type /Page", excluding the
// "/Type /Pages" tree root) in a rendered PDF.
func countPages(data []byte) int {
	re := regexp.MustCompile(`/Type\s*/Page[^s]`)
	return len(re.FindAll(data, -1))
}
