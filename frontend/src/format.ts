/** Parses user input that may use either a dot or a comma decimal separator. */
export function parseNumber(input: string): number {
  const normalised = input.trim().replace(',', '.');
  if (normalised === '') return 0;
  const value = Number(normalised);
  return Number.isFinite(value) ? value : 0;
}

/** Renders a number with a fixed number of decimals (2 by default). */
export function formatNumber(value: number, decimals = 2): string {
  return value.toFixed(decimals);
}

/** Renders an ISO date (YYYY-MM-DD) the way the paper form writes it. */
export function formatDateRO(iso: string): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso);
  if (!match) return iso;
  const [, year, month, day] = match;
  return `${day}.${month}.${year}`;
}
