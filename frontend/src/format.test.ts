import { describe, expect, it } from 'vitest';
import { formatDateRO, formatNumber, parseNumber } from './format';

describe('parseNumber', () => {
  it('accepts a dot decimal separator', () => {
    expect(parseNumber('162.20')).toBe(162.2);
  });

  it('accepts a comma decimal separator', () => {
    expect(parseNumber('162,20')).toBe(162.2);
  });

  it('treats blank and unparseable input as zero', () => {
    expect(parseNumber('')).toBe(0);
    expect(parseNumber('   ')).toBe(0);
    expect(parseNumber('abc')).toBe(0);
  });

  it('trims surrounding whitespace', () => {
    expect(parseNumber(' 15 ')).toBe(15);
  });
});

describe('formatNumber', () => {
  it('renders two decimals by default', () => {
    expect(formatNumber(328.5)).toBe('328.50');
    expect(formatNumber(0)).toBe('0.00');
  });

  it('honours an explicit decimal count', () => {
    expect(formatNumber(21.9, 3)).toBe('21.900');
  });
});

describe('formatDateRO', () => {
  it('renders an ISO date as dd.mm.yyyy', () => {
    expect(formatDateRO('2026-09-03')).toBe('03.09.2026');
  });

  it('passes through anything that is not an ISO date', () => {
    expect(formatDateRO('')).toBe('');
    expect(formatDateRO('nope')).toBe('nope');
  });
});
