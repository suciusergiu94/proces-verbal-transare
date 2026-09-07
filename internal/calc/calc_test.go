package calc

import (
	"testing"

	"proces-verbal-transare/internal/model"
)

func TestRound2(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{
		{2.346, 2.35},
		{-2.346, -2.35},
		{2.344999, 2.34},
		{0, 0},
		{328.5, 328.5},
		// 1.005 is stored as 1.00499999999999989, so rounding down is correct.
		// Pinned here so the binary-float behaviour is documented, not accidental.
		{1.005, 1.0},
	}
	for _, c := range cases {
		if got := Round2(c.in); got != c.want {
			t.Errorf("Round2(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestValoare(t *testing.T) {
	// From the filled form: Pulpa fara os, 15 Kg x 21.9 = 328.50.
	if got := ValoareCuTVA(15, 21.9); got != 328.5 {
		t.Errorf("ValoareCuTVA(15, 21.9) = %v, want 328.5", got)
	}
	// Ceafa fara os, 8.5 Kg x 33.9 = 288.15.
	if got := ValoareCuTVA(8.5, 33.9); got != 288.15 {
		t.Errorf("ValoareCuTVA(8.5, 33.9) = %v, want 288.15", got)
	}
	// Slanina tablii, 10.5 Kg x 18.5 = 194.25.
	if got := ValoareCuTVA(10.5, 18.5); got != 194.25 {
		t.Errorf("ValoareCuTVA(10.5, 18.5) = %v, want 194.25", got)
	}
	if got := ValoareFaraTVA(15, 16.5); got != 247.5 {
		t.Errorf("ValoareFaraTVA(15, 16.5) = %v, want 247.5", got)
	}
}

func TestTotalsIntrare(t *testing.T) {
	rows := []model.IntrareRow{
		{Cantitate: 162.2, PretFaraTVA: 12.5, PretCuTVA: 14.03},
		{Cantitate: 10, PretFaraTVA: 2, PretCuTVA: 2.5},
	}
	got := TotalsIntrare(rows)
	if got.Cantitate != 172.2 {
		t.Errorf("Cantitate = %v, want 172.2", got.Cantitate)
	}
	if got.ValoareFaraTVA != 2047.5 {
		t.Errorf("ValoareFaraTVA = %v, want 2047.5", got.ValoareFaraTVA)
	}
	if got.ValoareCuTVA != 2300.67 {
		t.Errorf("ValoareCuTVA = %v, want 2300.67", got.ValoareCuTVA)
	}
}

func TestTotalsIesire(t *testing.T) {
	rows := []model.IesireRow{
		{Cantitate: 15, PretFaraTVA: 16.5, PretCuTVA: 21.9},
		{Cantitate: 8.5, PretFaraTVA: 0, PretCuTVA: 33.9},
	}
	got := TotalsIesire(rows)
	if got.Cantitate != 23.5 {
		t.Errorf("Cantitate = %v, want 23.5", got.Cantitate)
	}
	if got.ValoareFaraTVA != 247.5 {
		t.Errorf("ValoareFaraTVA = %v, want 247.5", got.ValoareFaraTVA)
	}
	if got.ValoareCuTVA != 616.65 {
		t.Errorf("ValoareCuTVA = %v, want 616.65", got.ValoareCuTVA)
	}
}

func TestTotalsEmpty(t *testing.T) {
	got := TotalsIesire(nil)
	if got != (Totals{}) {
		t.Errorf("TotalsIesire(nil) = %+v, want zero Totals", got)
	}
}

func TestDiferenta(t *testing.T) {
	tip, val := Diferenta(2300.67, 1000)
	if tip != "plus" || val != 1300.67 {
		t.Errorf("Diferenta(2300.67, 1000) = %q, %v; want \"plus\", 1300.67", tip, val)
	}
	tip, val = Diferenta(1000, 2300.67)
	if tip != "minus" || val != 1300.67 {
		t.Errorf("Diferenta(1000, 2300.67) = %q, %v; want \"minus\", 1300.67", tip, val)
	}
	tip, val = Diferenta(500, 500)
	if tip != "" || val != 0 {
		t.Errorf("Diferenta(500, 500) = %q, %v; want \"\", 0", tip, val)
	}
}

func TestIncarcaDescarca(t *testing.T) {
	tip, val := IncarcaDescarca(2300.67, 1000)
	if tip != "incarca" || val != 1300.67 {
		t.Errorf("IncarcaDescarca(2300.67, 1000) = %q, %v; want \"incarca\", 1300.67", tip, val)
	}
	tip, val = IncarcaDescarca(1000, 2300.67)
	if tip != "descarca" || val != 1300.67 {
		t.Errorf("IncarcaDescarca(1000, 2300.67) = %q, %v; want \"descarca\", 1300.67", tip, val)
	}
	tip, val = IncarcaDescarca(500, 500)
	if tip != "" || val != 0 {
		t.Errorf("IncarcaDescarca(500, 500) = %q, %v; want \"\", 0", tip, val)
	}
}
