import { describe, expect, it } from 'vitest';
import {
  diferenta,
  incarcaDescarca,
  marjaProfit,
  pretCuTvaDin,
  pretFaraTvaDin,
  round2,
  totals,
  valoare,
} from './calc';

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
