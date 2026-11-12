/**
 * Display helpers for usage counters.
 *
 * Cost arrives as integer micro-USD. The panel formats it with the active
 * locale, keeps the full value in a title, and never rounds a non-zero cost
 * down to a misleading `$0.00`.
 */

const MICRO_USD_PER_USD = 1_000_000;
/** $0.01 expressed in micro-USD. */
export const TINY_COST_MICRO_USD = 10_000;

const isFiniteNumber = (value: number): boolean => Number.isFinite(value);

const currencyFormat = (locale: string, maximumFractionDigits: number): Intl.NumberFormat =>
  new Intl.NumberFormat(locale, {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: Math.min(2, maximumFractionDigits),
    maximumFractionDigits,
  });

/**
 * Cost for a card or a table cell.
 *
 * A non-zero cost below one cent shows as `<` plus the locale's one-cent
 * currency string, so a real cost never looks like zero.
 */
export function formatMicroUsd(microUsd: number, locale = 'en'): string {
  if (!isFiniteNumber(microUsd)) return '—';
  if (microUsd > 0 && microUsd < TINY_COST_MICRO_USD) {
    return `<${currencyFormat(locale, 2).format(TINY_COST_MICRO_USD / MICRO_USD_PER_USD)}`;
  }
  const usd = microUsd / MICRO_USD_PER_USD;
  return currencyFormat(locale, usd >= 1000 ? 0 : 2).format(usd);
}

/** Full-precision cost for a `title`, so the rounded display stays checkable. */
export function formatMicroUsdExact(microUsd: number, locale = 'en'): string {
  if (!isFiniteNumber(microUsd)) return '—';
  return currencyFormat(locale, 6).format(microUsd / MICRO_USD_PER_USD);
}

/** Token counts in compact notation: 1.2K, 3.4M. */
export function formatTokenCount(tokens: number, locale = 'en'): string {
  if (!isFiniteNumber(tokens)) return '0';
  return new Intl.NumberFormat(locale, {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(tokens);
}

/** Duration in ms as a short human string: "850 ms", "1.2 s", "2m 5s", "3h 4m", "2d 3h". */
export function formatDurationMs(ms: number): string {
  if (!isFiniteNumber(ms) || ms < 0) return '—';
  if (ms < 1000) return `${Math.round(ms)} ms`;
  if (ms < 60_000) {
    const seconds = ms / 1000;
    return `${seconds.toFixed(seconds < 10 ? 1 : 0)} s`;
  }
  const totalSeconds = Math.round(ms / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  if (minutes < 60) {
    const seconds = totalSeconds % 60;
    return seconds === 0 ? `${minutes}m` : `${minutes}m ${seconds}s`;
  }
  const hours = Math.floor(minutes / 60);
  if (hours < 24) {
    const restMinutes = minutes % 60;
    return restMinutes === 0 ? `${hours}h` : `${hours}h ${restMinutes}m`;
  }
  const days = Math.floor(hours / 24);
  const restHours = hours % 24;
  return restHours === 0 ? `${days}d` : `${days}d ${restHours}h`;
}

/** A 0-to-1 rate as a localized percentage: 0.025 -> "2.5%". */
export function formatUsageRate(rate: number, locale = 'en'): string {
  if (!isFiniteNumber(rate)) return '—';
  return new Intl.NumberFormat(locale, {
    style: 'percent',
    maximumFractionDigits: 1,
  }).format(rate);
}

/** Absolute relative change as a percentage, for a delta line: 12.4 -> "12%". */
export function formatUsageChangePercent(percent: number, locale = 'en'): string {
  if (!isFiniteNumber(percent)) return '—';
  return new Intl.NumberFormat(locale, {
    style: 'percent',
    maximumFractionDigits: 0,
  }).format(Math.abs(percent) / 100);
}

/** A difference between two rates, in percentage points. */
export function formatPercentagePoints(points: number, locale = 'en'): string {
  if (!isFiniteNumber(points)) return '—';
  const formatted = new Intl.NumberFormat(locale, {
    maximumFractionDigits: 1,
  }).format(Math.abs(points));
  return `${formatted} pp`;
}
