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

/**
 * Rounds to three decimals, half away from zero. Mirror of calc.Round3: the
 * precision the carcass ratios are kept at.
 */
export function round3(value: number): number {
  const rounded = Math.round(Math.abs(value) * 1000) / 1000;
  return value < 0 ? -rounded : rounded;
}

/**
 * What a column of carcass ratios accounts for, in percent.
 *
 * Rounded, because it is compared against 100: nineteen three-decimal ratios
 * added left to right land on 99.99999999999999, and Setări must not call a
 * correct column wrong over a floating-point crumb.
 */
export function sumaProcente(procente: number[]): number {
  return round3(procente.reduce((sum, procent) => sum + procent, 0));
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

/**
 * The smallest quantity the form works in: two decimals of a kilogram.
 * Adjusted quantities always land on a multiple of this.
 */
const PAS_CANTITATE = 0.01;

/** How many times the solver may enlarge its step before giving up. */
const MAX_INCERCARI = 60;

/** How much it enlarges the step by on each retry. */
const CRESTERE = 1.25;

/**
 * New "ce iese" quantities whose margin is at least `delta` percentage points
 * away from the current one — positive to raise it, negative to lower it.
 *
 * Each row moves by an amount proportional to `cantitate x (pret - pretul
 * mediu ponderat)`: products worth more than the average go up, products worth
 * less go down, and every row moves a little in proportion to its own size
 * rather than one row being emptied into another. Those shifts sum to zero by
 * construction, so the total quantity is conserved — the butchering yielded
 * what it yielded, and this only changes how it was split.
 *
 * Returns undefined when no such redistribution exists: nothing has gone in or
 * come out yet, every product carries the same price (there is no mix to
 * shift), or a product would have to go negative before the target is reached.
 * Callers leave the table alone in that case.
 *
 * Like marjaProfit this is a working aid shown on screen only — it has no
 * counterpart in internal/calc, and only the quantities it produces are ever
 * saved.
 */
export function ajusteazaMarja(
  rows: RowLike[],
  intrareCuTva: number,
  delta: number,
): number[] | undefined {
  // A margin on a zero cost has no meaning, so there is nothing to move.
  if (intrareCuTva <= 0 || delta === 0) return undefined;

  const cantitati = rows.map((row) => row.cantitate);
  const preturi = rows.map((row) => row.pretCuTva);
  const totalCantitate = cantitati.reduce((sum, q) => sum + q, 0);
  if (totalCantitate <= 0) return undefined;

  // The average price a kilogram of output carries. A row above it is worth
  // adding to, a row below it is worth taking from.
  const pretMediu = cantitati.reduce((sum, q, i) => sum + q * preturi[i], 0) / totalCantitate;

  // The direction each row moves in. Sums to zero, which is what conserves
  // the total quantity for any step size.
  const directie = cantitati.map((q, i) => q * (preturi[i] - pretMediu));

  // Moving one step along `directie` changes the output value by this much:
  // the quantity-weighted variance of the prices. It is zero exactly when
  // every produced row carries the same price, and then no redistribution can
  // move the margin at all.
  const varianta = directie.reduce((sum, d, i) => sum + d * preturi[i], 0);
  if (varianta <= 0) return undefined;

  // How far the step may go before the first shrinking row hits zero. Beyond
  // this a product would have to yield a negative quantity.
  const sens = delta > 0 ? 1 : -1;
  let limita = Infinity;
  directie.forEach((d, i) => {
    if (sens * d < 0) limita = Math.min(limita, cantitati[i] / Math.abs(d));
  });

  const marjaCurenta = marjaProfit(totals(rows).valoareCuTva, intrareCuTva);
  if (marjaCurenta === undefined) return undefined;
  const tinta = round2(marjaCurenta + delta);

  // The step that closes the gap exactly, before rounding: the value the
  // output must gain, divided by what one step is worth.
  let pas = ((delta / 100) * intrareCuTva) / varianta;

  // Rounding to whole bani of a kilogram can swallow a step this small
  // entirely, leaving the margin where it was. Start from a step big enough
  // to move at least one row by one increment.
  const directieMaxima = Math.max(...directie.map((d) => Math.abs(d)));
  const pasMinim = PAS_CANTITATE / directieMaxima;
  if (Math.abs(pas) < pasMinim) pas = sens * pasMinim;

  for (let incercare = 0; incercare < MAX_INCERCARI; incercare++) {
    const laLimita = Math.abs(pas) >= limita;
    const propunere = redistribuie(cantitati, directie, laLimita ? sens * limita : pas);
    const marjaNoua = marjaProfit(
      totals(rows.map((row, i) => ({ ...row, cantitate: propunere[i] }))).valoareCuTva,
      intrareCuTva,
    );
    if (marjaNoua !== undefined && atinge(marjaNoua, tinta, delta)) return propunere;
    // The step has run out of room and still falls short: no redistribution
    // gets there, so leave the table as the user left it.
    if (laLimita) return undefined;
    pas *= CRESTERE;
  }
  return undefined;
}

/** Whether a new margin has moved far enough in the requested direction. */
function atinge(marjaNoua: number, tinta: number, delta: number): boolean {
  const EPSILON = 1e-9;
  return delta > 0 ? marjaNoua >= tinta - EPSILON : marjaNoua <= tinta + EPSILON;
}

/**
 * Applies one step along `directie`, rounded to whole bani of a kilogram.
 *
 * Rounding each row on its own would let the table's total wander by a few
 * bani on every click, so the bani that rounding leaves over are handed to the
 * rows with the largest fractions until the total matches what it was — the
 * largest-remainder method. Rows that produced nothing keep their zero: they
 * cannot absorb a leftover ban, because meat that never came off the carcass
 * cannot be booked against them.
 */
function redistribuie(cantitati: number[], directie: number[], pas: number): number[] {
  const bani = cantitati.map((q, i) => Math.max(0, q + pas * directie[i]) / PAS_CANTITATE);
  const intregi = bani.map((b) => Math.floor(b));
  const tinta = Math.round(cantitati.reduce((sum, q) => sum + q, 0) / PAS_CANTITATE);

  let rest = tinta - intregi.reduce((sum, b) => sum + b, 0);

  // Candidates ordered by how much of a ban each row had shaved off it, so
  // the leftovers go where they are least arbitrary.
  const candidati = bani
    .map((b, i) => ({ i, fractie: b - intregi[i] }))
    .filter(({ i }) => cantitati[i] > 0)
    .sort((a, b) => b.fractie - a.fractie);

  for (let n = 0; rest > 0 && n < candidati.length; n++) {
    intregi[candidati[n].i] += 1;
    rest -= 1;
  }
  for (let n = candidati.length - 1; rest < 0 && n >= 0; n--) {
    if (intregi[candidati[n].i] > 0) {
      intregi[candidati[n].i] -= 1;
      rest += 1;
    }
  }

  return intregi.map((b) => round2(b * PAS_CANTITATE));
}

/**
 * The "ce iese" quantities a carcass of `totalIntrare` yields, given each
 * product's share of the input in percent (see model.Product.ProcentDinIntrare).
 *
 * The ratios are stored to three decimals and the quantities are kept to whole
 * bani of a kilogram, so rounding each row on its own would leave the table a
 * ban or two short of the carcass it came from. The leftovers are handed to the
 * rows with the largest fractions until the total matches — the same
 * largest-remainder method redistribuie uses — with one difference: the target
 * is what the ratios themselves add up to, not the carcass weight. A column
 * that only accounts for 75% of the input yields three quarters of a carcass;
 * scaling it up to the whole would invent meat that was never cut.
 *
 * A product with a zero ratio stays at zero and never absorbs a leftover ban:
 * a product the butchering does not produce cannot be booked a quantity.
 *
 * Like marjaProfit and ajusteazaMarja this has no counterpart in internal/calc
 * — it fills the form, and only the quantities it produces are ever saved.
 */
export function cantitatiDinProcente(procente: number[], totalIntrare: number): number[] {
  if (totalIntrare <= 0) return procente.map(() => 0);

  // Each row's exact yield, counted in whole bani of a kilogram so the
  // leftovers can be handed out as indivisible units.
  const bani = procente.map((procent) => (Math.max(0, procent) / 100 / PAS_CANTITATE) * totalIntrare);
  const intregi = bani.map((b) => Math.floor(b));
  const tinta = Math.round(bani.reduce((sum, b) => sum + b, 0));

  let rest = tinta - intregi.reduce((sum, b) => sum + b, 0);
  const candidati = bani
    .map((b, i) => ({ i, fractie: b - intregi[i] }))
    .filter(({ i }) => procente[i] > 0)
    .sort((a, b) => b.fractie - a.fractie);
  for (let n = 0; rest > 0 && n < candidati.length; n++) {
    intregi[candidati[n].i] += 1;
    rest -= 1;
  }

  return intregi.map((b) => round2(b * PAS_CANTITATE));
}

/**
 * The carcass ratios a column of "ce iese" quantities implies, in percent —
 * the inverse of cantitatiDinProcente, used to store a split the user arrived
 * at by nudging the margin.
 *
 * The quantities are read as a split of the whole carcass: the column is
 * scaled to account for exactly 100%, whatever went in. A document whose
 * output does not weigh what its input did still yields a usable column that
 * way, and it is the split between products the nudging moved, never the
 * carcass weight — ajusteazaMarja conserves the total.
 *
 * Rounded to three decimals, the precision Setări stores and shows, with the
 * leftover thousandths handed to the largest remainders so the column lands on
 * exactly 100 instead of 99.999 — Setări refuses to store anything else. A
 * zero quantity stays at zero and never absorbs a leftover: a product the
 * butchering did not produce cannot be booked a share.
 *
 * Returns undefined when nothing has come out at all, the one column with no
 * split to describe.
 *
 * Like marjaProfit, ajusteazaMarja and cantitatiDinProcente this is a working
 * aid with no counterpart in internal/calc.
 */
export function procenteDinCantitati(cantitati: number[]): number[] | undefined {
  const total = cantitati.reduce((sum, q) => sum + Math.max(0, q), 0);
  if (total <= 0) return undefined;

  // Counted in whole thousandths of a percent, so the leftovers can be handed
  // out as indivisible units the way cantitatiDinProcente hands out bani.
  const miimi = cantitati.map((q) => (Math.max(0, q) / total) * 100000);
  const intregi = miimi.map((m) => Math.floor(m));

  let rest = 100000 - intregi.reduce((sum, m) => sum + m, 0);
  const candidati = miimi
    .map((m, i) => ({ i, fractie: m - intregi[i] }))
    .filter(({ i }) => cantitati[i] > 0)
    .sort((a, b) => b.fractie - a.fractie);
  for (let n = 0; rest > 0 && n < candidati.length; n++) {
    intregi[candidati[n].i] += 1;
    rest -= 1;
  }

  return intregi.map((m) => round3(m / 1000));
}
