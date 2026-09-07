/**
 * Mirror of internal/calc, used for live display while the user types. The Go
 * side re-derives the same values for the PDF, so the two must stay in step.
 */

export type RowLike = {
  cantitate: number;
  pretFaraTva: number;
  pretCuTva: number;
};

export type Totals = {
  cantitate: number;
  valoareFaraTva: number;
  valoareCuTva: number;
};

/** Rounds to two decimals, half away from zero. */
export function round2(value: number): number {
  const rounded = Math.round(Math.abs(value) * 100) / 100;
  return value < 0 ? -rounded : rounded;
}

/** cantitate x pret, rounded to two decimals. */
export function valoare(cantitate: number, pret: number): number {
  return round2(cantitate * pret);
}

/** Sums a table's numeric columns. */
export function totals(rows: RowLike[]): Totals {
  const sum = rows.reduce<Totals>(
    (acc, row) => ({
      cantitate: acc.cantitate + row.cantitate,
      valoareFaraTva: acc.valoareFaraTva + valoare(row.cantitate, row.pretFaraTva),
      valoareCuTva: acc.valoareCuTva + valoare(row.cantitate, row.pretCuTva),
    }),
    { cantitate: 0, valoareFaraTva: 0, valoareCuTva: 0 },
  );
  return {
    cantitate: round2(sum.cantitate),
    valoareFaraTva: round2(sum.valoareFaraTva),
    valoareCuTva: round2(sum.valoareCuTva),
  };
}

/** The signed gap between the two tables' "valoare cu TVA" totals. */
export function diferenta(
  iesireCuTva: number,
  intrareCuTva: number,
): { tip: 'plus' | 'minus' | ''; valoare: number } {
  const delta = round2(iesireCuTva - intrareCuTva);
  if (delta > 0) return { tip: 'plus', valoare: delta };
  if (delta < 0) return { tip: 'minus', valoare: -delta };
  return { tip: '', valoare: 0 };
}

/** The same magnitude, labelled the way the bottom-right block labels it. */
export function incarcaDescarca(
  iesireCuTva: number,
  intrareCuTva: number,
): { tip: 'incarca' | 'descarca' | ''; valoare: number } {
  const { tip, valoare: value } = diferenta(iesireCuTva, intrareCuTva);
  if (tip === 'plus') return { tip: 'incarca', valoare: value };
  if (tip === 'minus') return { tip: 'descarca', valoare: value };
  return { tip: '', valoare: 0 };
}
