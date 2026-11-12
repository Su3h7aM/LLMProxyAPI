import { describe, expect, test } from 'bun:test';
import { computeUsageDelta, computeUsageRateDelta } from '@/features/usage/logic/delta';
import { hasUnpricedRequests, isCostEstimated } from '@/features/usage/logic/flags';
import {
  formatDurationMs,
  formatMicroUsd,
  formatMicroUsdExact,
  formatPercentagePoints,
  formatTokenCount,
  formatUsageChangePercent,
  formatUsageRate,
} from '@/features/usage/logic/format';

describe('micro-USD display', () => {
  test('uses the locale currency format', () => {
    expect(formatMicroUsd(0, 'en')).toBe('$0.00');
    expect(formatMicroUsd(1_250_000, 'en')).toBe('$1.25');
    expect(formatMicroUsd(1_250_000, 'ru')).toBe('1,25\u00a0$');
  });

  test('never shows a non-zero cost as zero', () => {
    expect(formatMicroUsd(1, 'en')).toBe('<$0.01');
    expect(formatMicroUsd(9_999, 'en')).toBe('<$0.01');
    expect(formatMicroUsd(10_000, 'en')).toBe('$0.01');
    expect(formatMicroUsd(1, 'ru')).toBe('<0,01\u00a0$');
  });

  test('drops cents for large totals but keeps the exact value available', () => {
    expect(formatMicroUsd(1_234_567_000, 'en')).toBe('$1,235');
    expect(formatMicroUsdExact(1_234_567, 'en')).toBe('$1.234567');
    expect(formatMicroUsdExact(1, 'en')).toBe('$0.000001');
  });

  test('an unusable value is a dash, not zero', () => {
    expect(formatMicroUsd(Number.NaN)).toBe('—');
    expect(formatMicroUsdExact(Number.POSITIVE_INFINITY)).toBe('—');
  });
});

describe('tokens, duration, and rates', () => {
  test('token totals use compact localized notation', () => {
    expect(formatTokenCount(0, 'en')).toBe('0');
    expect(formatTokenCount(999, 'en')).toBe('999');
    expect(formatTokenCount(1_234_567, 'en')).toBe('1.2M');
  });

  test('durations shrink their precision as they grow', () => {
    expect(formatDurationMs(850)).toBe('850 ms');
    expect(formatDurationMs(1_250)).toBe('1.3 s');
    expect(formatDurationMs(64_000)).toBe('1m 4s');
    expect(formatDurationMs(3_600_000)).toBe('1h');
    expect(formatDurationMs(90_000_000)).toBe('1d 1h');
    expect(formatDurationMs(-1)).toBe('—');
  });

  test('a 0-to-1 rate renders as a percentage', () => {
    expect(formatUsageRate(0.025, 'en')).toBe('2.5%');
    expect(formatUsageRate(0, 'en')).toBe('0%');
    expect(formatUsageRate(0.025, 'ru')).toBe('2,5\u00a0%');
  });

  test('delta percent and percentage points are formatted without a sign', () => {
    expect(formatUsageChangePercent(12.4, 'en')).toBe('12%');
    expect(formatUsageChangePercent(-12.4, 'en')).toBe('12%');
    expect(formatPercentagePoints(-1.5, 'en')).toBe('1.5 pp');
  });
});

describe('period delta', () => {
  test('returns the absolute and relative change with a direction', () => {
    expect(computeUsageDelta(120, 100)).toEqual({ value: 20, percent: 20, direction: 'up' });
    expect(computeUsageDelta(80, 100)).toEqual({ value: -20, percent: -20, direction: 'down' });
    expect(computeUsageDelta(100, 100)).toEqual({ value: 0, percent: 0, direction: 'flat' });
  });

  test('has no data when the previous period is empty', () => {
    expect(computeUsageDelta(10, 0)).toBeNull();
    expect(computeUsageDelta(0, 0)).toBeNull();
    expect(computeUsageDelta(Number.NaN, 10)).toBeNull();
  });

  test('a rate delta is in percentage points and needs previous requests', () => {
    expect(computeUsageRateDelta(0.025, 0.01, 40)).toEqual({
      value: 1.5,
      percent: null,
      direction: 'up',
    });
    expect(computeUsageRateDelta(0.01, 0.025, 40)).toMatchObject({
      value: -1.5,
      direction: 'down',
    });
    expect(computeUsageRateDelta(0.01, 0, 0)).toBeNull();
  });
});

describe('cost marks', () => {
  test('estimate and unpriced marks need a positive count', () => {
    expect(isCostEstimated({ estimated_cost_micro_usd: 1 })).toBe(true);
    expect(isCostEstimated({ estimated_cost_micro_usd: 0 })).toBe(false);
    expect(hasUnpricedRequests({ unpriced_requests: 4 })).toBe(true);
    expect(hasUnpricedRequests({ unpriced_requests: 0 })).toBe(false);
  });
});
