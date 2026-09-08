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

/**
 * The profit margin of the butchering, as a percentage of what went in:
 * (total iesire − total intrare) / total intrare × 100, both "cu TVA" so the
 * figure agrees with Diferență. Negative when the butchering lost value.
 *
 * Returns undefined — not Infinity or NaN — when nothing went in, since a
 * margin on a zero cost has no meaning; callers render that as a blank.
 *
 * Unlike the rest of this module this has no counterpart in internal/calc: it
 * is a working figure shown on screen only, never stored and never printed.
 */
export function marjaProfit(iesireCuTva: number, intrareCuTva: number): number | undefined {
  if (intrareCuTva === 0) return undefined;
  return round2(((iesireCuTva - intrareCuTva) / intrareCuTva) * 100);
}

/**
 * The multiplier a TVA rate applies to a price without TVA, or undefined when
 * the rate cannot be applied at all: at -100% or below the multiplier is zero
 * or negative, which has no usable inverse. Callers treat undefined as "leave
 * the other price alone" rather than writing a zero or an Infinity.
 */
function multiplicatorTva(cota: number): number | undefined {
  const factor = 1 + cota / 100;
  return factor > 0 ? factor : undefined;
}

/** The price with TVA that a price without TVA implies at the given rate. */
export function pretCuTvaDin(pretFaraTva: number, cota: number): number | undefined {
  const factor = multiplicatorTva(cota);
  return factor === undefined ? undefined : round2(pretFaraTva * factor);
}

/** The price without TVA that a price with TVA implies at the given rate. */
export function pretFaraTvaDin(pretCuTva: number, cota: number): number | undefined {
  const factor = multiplicatorTva(cota);
  return factor === undefined ? undefined : round2(pretCuTva / factor);
}
