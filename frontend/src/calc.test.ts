import { describe, expect, it } from 'vitest';
import {
  ajusteazaMarja,
  diferenta,
  incarcaDescarca,
  marjaProfit,
  pretCuTvaDin,
  pretFaraTvaDin,
  round2,
  totals,
  valoare,
} from './calc';
import type { RowLike } from './calc';

describe('round2', () => {
  it('rounds half away from zero', () => {
    expect(round2(2.346)).toBe(2.35);
    expect(round2(-2.346)).toBe(-2.35);
    expect(round2(2.344999)).toBe(2.34);
  });

  it('rounds a decimal tie down, because 1.005 is stored as 1.00499999999999989', () => {
    expect(round2(1.005)).toBe(1.0);
  });
});

describe('valoare', () => {
  it('matches the filled paper form', () => {
    expect(valoare(15, 21.9)).toBe(328.5);
    expect(valoare(8.5, 33.9)).toBe(288.15);
    expect(valoare(10.5, 18.5)).toBe(194.25);
  });
});

describe('totals', () => {
  it('sums each column', () => {
    const got = totals([
      { cantitate: 15, pretFaraTva: 16.5, pretCuTva: 21.9 },
      { cantitate: 8.5, pretFaraTva: 0, pretCuTva: 33.9 },
    ]);
    expect(got.cantitate).toBe(23.5);
    expect(got.valoareFaraTva).toBe(247.5);
    expect(got.valoareCuTva).toBe(616.65);
  });

  it('returns zeroes for an empty table', () => {
    expect(totals([])).toEqual({ cantitate: 0, valoareFaraTva: 0, valoareCuTva: 0 });
  });
});

describe('diferenta', () => {
  it('labels a surplus plus and a shortfall minus', () => {
    expect(diferenta(2300.67, 1000)).toEqual({ tip: 'plus', valoare: 1300.67 });
    expect(diferenta(1000, 2300.67)).toEqual({ tip: 'minus', valoare: 1300.67 });
    expect(diferenta(500, 500)).toEqual({ tip: '', valoare: 0 });
  });
});

describe('incarcaDescarca', () => {
  it('uses the bottom-right block wording', () => {
    expect(incarcaDescarca(2300.67, 1000)).toEqual({ tip: 'incarca', valoare: 1300.67 });
    expect(incarcaDescarca(1000, 2300.67)).toEqual({ tip: 'descarca', valoare: 1300.67 });
    expect(incarcaDescarca(500, 500)).toEqual({ tip: '', valoare: 0 });
  });
});

describe('marjaProfit', () => {
  it('is the surplus as a percentage of what went in', () => {
    expect(marjaProfit(21311, 18000)).toBe(18.39);
    expect(marjaProfit(1200, 1000)).toBe(20);
  });

  it('is negative when the butchering lost value', () => {
    expect(marjaProfit(900, 1000)).toBe(-10);
  });

  it('is zero when the two tables balance', () => {
    expect(marjaProfit(1000, 1000)).toBe(0);
  });

  it('is undefined rather than infinite when nothing went in', () => {
    expect(marjaProfit(21311, 0)).toBeUndefined();
    expect(marjaProfit(0, 0)).toBeUndefined();
  });
});

describe('pretCuTvaDin', () => {
  it('adds the rate to a price without TVA', () => {
    expect(pretCuTvaDin(100, 11)).toBe(111);
  });

  it('rounds to two decimals', () => {
    expect(pretCuTvaDin(19.73, 11)).toBe(21.9);
  });

  it('leaves a price unchanged at a zero rate', () => {
    expect(pretCuTvaDin(21.9, 0)).toBe(21.9);
  });

  it('converts a zero price to zero', () => {
    expect(pretCuTvaDin(0, 11)).toBe(0);
  });

  it('gives up on a rate of -100% or lower, which has no inverse', () => {
    expect(pretCuTvaDin(21.9, -100)).toBeUndefined();
    expect(pretCuTvaDin(21.9, -150)).toBeUndefined();
  });
});

describe('pretFaraTvaDin', () => {
  it('strips the rate from a price with TVA', () => {
    expect(pretFaraTvaDin(111, 11)).toBe(100);
  });

  it('rounds to two decimals', () => {
    expect(pretFaraTvaDin(21.9, 11)).toBe(19.73);
  });

  it('leaves a price unchanged at a zero rate', () => {
    expect(pretFaraTvaDin(21.9, 0)).toBe(21.9);
  });

  it('converts a zero price to zero', () => {
    expect(pretFaraTvaDin(0, 11)).toBe(0);
  });

  it('gives up on a rate of -100% or lower, which would divide by zero', () => {
    expect(pretFaraTvaDin(21.9, -100)).toBeUndefined();
    expect(pretFaraTvaDin(21.9, -150)).toBeUndefined();
  });
});

describe('ajusteazaMarja', () => {
  // A butchering shaped like the real form: a few products at different
  // prices, all of them produced, against a fixed cost of what went in.
  const rows = (): RowLike[] => [
    { cantitate: 30, pretFaraTva: 22.52, pretCuTva: 25 },
    { cantitate: 20, pretFaraTva: 13.51, pretCuTva: 15 },
    { cantitate: 10, pretFaraTva: 9.01, pretCuTva: 10 },
  ];
  const INTRARE = 1000;

  /** The margin the form would show for these quantities. */
  const marjaOf = (base: RowLike[], cantitati: number[]): number | undefined =>
    marjaProfit(
      totals(base.map((row, i) => ({ ...row, cantitate: cantitati[i] }))).valoareCuTva,
      INTRARE,
    );

  const totalOf = (cantitati: number[]): number =>
    round2(cantitati.reduce((sum, q) => sum + q, 0));

  it('raises the margin by at least 0.25 of a percent', () => {
    const base = rows();
    const before = marjaOf(base, [30, 20, 10])!;
    const after = ajusteazaMarja(base, INTRARE, 0.25)!;
    expect(after).toBeDefined();
    expect(marjaOf(base, after)!).toBeGreaterThanOrEqual(before + 0.25);
  });

  it('lowers the margin by at least 0.25 of a percent', () => {
    const base = rows();
    const before = marjaOf(base, [30, 20, 10])!;
    const after = ajusteazaMarja(base, INTRARE, -0.25)!;
    expect(marjaOf(base, after)!).toBeLessThanOrEqual(before - 0.25);
  });

  it('does not overshoot: one step moves the margin far less than a whole percent', () => {
    const base = rows();
    const before = marjaOf(base, [30, 20, 10])!;
    const after = ajusteazaMarja(base, INTRARE, 0.25)!;
    expect(marjaOf(base, after)!).toBeLessThan(before + 1);
  });

  it('keeps the total quantity unchanged — meat is only redistributed', () => {
    const after = ajusteazaMarja(rows(), INTRARE, 0.25)!;
    expect(totalOf(after)).toBe(60);
  });

  it('keeps the total unchanged over many steps, with no drift', () => {
    const base = rows();
    let cantitati = [30, 20, 10];
    for (let i = 0; i < 8; i++) {
      const next = ajusteazaMarja(
        base.map((row, j) => ({ ...row, cantitate: cantitati[j] })),
        INTRARE,
        0.25,
      );
      if (next === undefined) break;
      cantitati = next;
      expect(totalOf(cantitati)).toBe(60);
    }
  });

  it('moves every product a little rather than emptying the cheapest one', () => {
    const base = rows();
    const after = ajusteazaMarja(base, INTRARE, 0.25)!;
    // Above the weighted average price (20 lei) goes up, below it goes down.
    expect(after[0]).toBeGreaterThan(30);
    expect(after[1]).toBeLessThan(20);
    expect(after[2]).toBeLessThan(10);
    // And each change stays small — no wholesale transfer between two rows.
    after.forEach((q, i) => {
      expect(Math.abs(q - base[i].cantitate)).toBeLessThan(base[i].cantitate * 0.1);
    });
  });

  it('rounds quantities to two decimals', () => {
    const after = ajusteazaMarja(rows(), INTRARE, 0.25)!;
    after.forEach((q) => expect(round2(q)).toBe(q));
  });

  it('leaves a product that was not produced at zero', () => {
    const base: RowLike[] = [
      ...rows(),
      { cantitate: 0, pretFaraTva: 45.05, pretCuTva: 50 },
    ];
    const after = ajusteazaMarja(base, INTRARE, 0.25)!;
    expect(after[3]).toBe(0);
  });

  it('never produces a negative quantity', () => {
    const base = rows();
    let cantitati = [30, 20, 10];
    for (let i = 0; i < 200; i++) {
      const next = ajusteazaMarja(
        base.map((row, j) => ({ ...row, cantitate: cantitati[j] })),
        INTRARE,
        -0.25,
      );
      if (next === undefined) break;
      cantitati = next;
      cantitati.forEach((q) => expect(q).toBeGreaterThanOrEqual(0));
    }
  });

  it('gives up rather than exhausting a product to reach the target', () => {
    // Only 0.01 kg of the cheap product is left to move, nowhere near the
    // 0.25 percent a step must be worth.
    const base: RowLike[] = [
      { cantitate: 40, pretFaraTva: 22.52, pretCuTva: 25 },
      { cantitate: 0.01, pretFaraTva: 0.9, pretCuTva: 1 },
    ];
    expect(ajusteazaMarja(base, INTRARE, 0.25)).toBeUndefined();
  });

  it('gives up when every product carries the same price, leaving no mix to shift', () => {
    const base: RowLike[] = [
      { cantitate: 30, pretFaraTva: 22.52, pretCuTva: 25 },
      { cantitate: 20, pretFaraTva: 22.52, pretCuTva: 25 },
    ];
    expect(ajusteazaMarja(base, INTRARE, 0.25)).toBeUndefined();
  });

  it('gives up when there is a single product, since one row is its own average', () => {
    const base: RowLike[] = [{ cantitate: 30, pretFaraTva: 22.52, pretCuTva: 25 }];
    expect(ajusteazaMarja(base, INTRARE, 0.25)).toBeUndefined();
  });

  it('gives up when nothing went in, since the margin itself is undefined', () => {
    expect(ajusteazaMarja(rows(), 0, 0.25)).toBeUndefined();
  });

  it('gives up when nothing has come out yet', () => {
    const base: RowLike[] = [
      { cantitate: 0, pretFaraTva: 22.52, pretCuTva: 25 },
      { cantitate: 0, pretFaraTva: 13.51, pretCuTva: 15 },
    ];
    expect(ajusteazaMarja(base, INTRARE, 0.25)).toBeUndefined();
  });

  it('gives up on an empty table', () => {
    expect(ajusteazaMarja([], INTRARE, 0.25)).toBeUndefined();
  });
});
