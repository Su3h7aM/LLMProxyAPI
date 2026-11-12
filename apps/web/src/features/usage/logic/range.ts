/**
 * Usage range presets and the browser time zone.
 *
 * The API takes a half-open range in epoch milliseconds. Presets end at the
 * moment of the request, so a poll always covers the latest window.
 */

export const USAGE_RANGE_PRESETS = ['1h', '24h', '7d', '30d', '90d', 'custom'] as const;

export type UsageRangePreset = (typeof USAGE_RANGE_PRESETS)[number];

export const USAGE_RANGE_MS: Record<Exclude<UsageRangePreset, 'custom'>, number> = {
  '1h': 60 * 60 * 1000,
  '24h': 24 * 60 * 60 * 1000,
  '7d': 7 * 24 * 60 * 60 * 1000,
  '30d': 30 * 24 * 60 * 60 * 1000,
  '90d': 90 * 24 * 60 * 60 * 1000,
};

/** The API rejects a range longer than 366 days. */
export const USAGE_RANGE_MAX_MS = 366 * 24 * 60 * 60 * 1000;

export const isUsageRangePreset = (value: unknown): value is UsageRangePreset =>
  typeof value === 'string' && (USAGE_RANGE_PRESETS as readonly string[]).includes(value);

export interface UsageRangeInput {
  preset: UsageRangePreset;
  customFromMs?: number | null;
  customToMs?: number | null;
}

export interface UsageRangeBounds {
  from_ms: number;
  to_ms: number;
}

/** Bounds for a preset, or null for an unusable custom range. */
export function resolveUsageRange(
  input: UsageRangeInput,
  nowMs: number
): UsageRangeBounds | null {
  if (!Number.isFinite(nowMs)) return null;
  if (input.preset === 'custom') {
    const from = input.customFromMs;
    const to = input.customToMs;
    if (from === null || from === undefined || to === null || to === undefined) return null;
    if (!Number.isFinite(from) || !Number.isFinite(to)) return null;
    if (to <= from || to - from > USAGE_RANGE_MAX_MS) return null;
    return { from_ms: from, to_ms: to };
  }
  const span = USAGE_RANGE_MS[input.preset];
  return { from_ms: nowMs - span, to_ms: nowMs };
}

/** The browser IANA zone, or UTC when the runtime does not report one. */
export function getBrowserTimeZone(): string {
  try {
    const zone = new Intl.DateTimeFormat().resolvedOptions().timeZone;
    return zone && zone.trim() ? zone.trim() : 'UTC';
  } catch {
    return 'UTC';
  }
}
