/**
 * Change between a period and the previous period of the same length.
 *
 * `null` means "no data": the previous period is empty, so a relative
 * change has no meaning. A rate can still have a change in percentage
 * points; that path is `computeUsageRateDelta`.
 */

export type UsageDeltaDirection = 'up' | 'down' | 'flat';

export interface UsageDelta {
  /** Absolute change in the metric's own unit (count, or percentage points). */
  value: number;
  /** Relative change in percent; null for rates and when the previous value is zero. */
  percent: number | null;
  direction: UsageDeltaDirection;
}

const directionOf = (value: number): UsageDeltaDirection =>
  value > 0 ? 'up' : value < 0 ? 'down' : 'flat';

/** Change of a count between two periods. Null when the previous count is zero. */
export function computeUsageDelta(current: number, previous: number): UsageDelta | null {
  if (!Number.isFinite(current) || !Number.isFinite(previous) || previous <= 0) return null;
  const value = current - previous;
  return {
    value,
    percent: (value / previous) * 100,
    direction: directionOf(value),
  };
}

/**
 * Change of a 0-to-1 rate in percentage points.
 *
 * Null when the previous period has no requests: a zero rate from an empty
 * period is not evidence of a change.
 */
export function computeUsageRateDelta(
  currentRate: number,
  previousRate: number,
  previousRequests: number
): UsageDelta | null {
  if (
    !Number.isFinite(currentRate) ||
    !Number.isFinite(previousRate) ||
    !Number.isFinite(previousRequests) ||
    previousRequests <= 0
  ) {
    return null;
  }
  // Rates rarely represent exact binary fractions; keep the point value readable.
  const value = Math.round((currentRate - previousRate) * 100 * 1e6) / 1e6;
  return { value, percent: null, direction: directionOf(value) };
}
