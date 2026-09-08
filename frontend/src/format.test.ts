import { describe, expect, it } from 'vitest';
import { formatDateRO, formatNumber, parseDateRO, parseNumber } from './format';

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
  it('renders an ISO date as dd/mm/yyyy', () => {
    expect(formatDateRO('2026-09-03')).toBe('03/09/2026');
  });

  it('passes through anything that is not an ISO date', () => {
    expect(formatDateRO('')).toBe('');
    expect(formatDateRO('nope')).toBe('nope');
  });
});

describe('parseDateRO', () => {
  it('reads a dd/mm/yyyy date back as ISO', () => {
    expect(parseDateRO('03/09/2026')).toBe('2026-09-03');
  });

  it('accepts a dot or a dash separator and missing leading zeros', () => {
    expect(parseDateRO('03.09.2026')).toBe('2026-09-03');
    expect(parseDateRO('03-09-2026')).toBe('2026-09-03');
    expect(parseDateRO('3/9/2026')).toBe('2026-09-03');
    expect(parseDateRO('  3/9/2026  ')).toBe('2026-09-03');
  });

  it('accepts 29 February in a leap year', () => {
    expect(parseDateRO('29/02/2024')).toBe('2024-02-29');
  });

  it('rejects dates that do not exist', () => {
    expect(parseDateRO('31/02/2026')).toBeUndefined();
    expect(parseDateRO('29/02/2026')).toBeUndefined();
    expect(parseDateRO('01/13/2026')).toBeUndefined();
    expect(parseDateRO('00/09/2026')).toBeUndefined();
  });

  it('rejects blank and half-typed input', () => {
    expect(parseDateRO('')).toBeUndefined();
    expect(parseDateRO('03/09')).toBeUndefined();
    expect(parseDateRO('2026-09-03')).toBeUndefined();
    expect(parseDateRO('nope')).toBeUndefined();
  });

  it('round-trips with formatDateRO', () => {
    expect(formatDateRO(parseDateRO('03/09/2026')!)).toBe('03/09/2026');
  });
});
