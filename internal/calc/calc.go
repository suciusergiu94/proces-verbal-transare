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
