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
