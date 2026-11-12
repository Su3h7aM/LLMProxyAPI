import { describe, expect, test } from 'bun:test';
import {
  buildUsageCards,
  dashboardUsageBucket,
  dashboardUsageQuery,
  topUsageModels,
  usageSparkPoints,
} from '@/features/dashboard/logic/usage';
import type { UsageCounters, UsageSummaryResponse } from '@/types/usageStats';

const NOW = Date.parse('2026-11-11T12:00:00.000Z');

const counters = (overrides: Partial<UsageCounters> = {}): UsageCounters => ({
  requests: 120,
  failures: 3,
  failure_rate: 0.025,
  input_tokens: 100_000,
  output_tokens: 40_000,
  reasoning_tokens: 5_000,
  cache_read_tokens: 30_000,
  cache_write_tokens: 2_000,
  total_tokens: 177_000,
  cost_micro_usd: 1_250_000,
  estimated_cost_micro_usd: 0,
  unpriced_requests: 0,
  latency_avg_ms: 840,
  latency_max_ms: 9_100,
  ttft_avg_ms: 310,
  ...overrides,
});

const summary = (totals: UsageCounters, previous: UsageCounters): UsageSummaryResponse => ({
  range: { from_ms: NOW - 86_400_000, to_ms: NOW },
  totals,
  previous,
  latency: { p50_ms: 600, p95_ms: 2_100, source: 'events' },
  first_event_ms: NOW - 86_400_000,
});

describe('usage cards', () => {
  test('four cards in reading order with the previous-period delta', () => {
    const cards = buildUsageCards(
      summary(
        counters(),
        counters({
          requests: 100,
          total_tokens: 150_000,
          cost_micro_usd: 1_000_000,
          failure_rate: 0.01,
        })
      ),
      'en'
    );
    expect(cards.map((card) => card.key)).toEqual(['requests', 'tokens', 'cost', 'failures']);
    expect(cards[0]).toMatchObject({ value: '120', deltaUnit: 'percent' });
    expect(cards[0].delta).toEqual({ value: 20, percent: 20, direction: 'up' });
    expect(cards[1]).toMatchObject({ value: '177K', title: '177,000' });
    expect(cards[2]).toMatchObject({ value: '$1.25', title: '$1.25' });
    expect(cards[3]).toMatchObject({ value: '2.5%', deltaUnit: 'points' });
    expect(cards[3].delta).toEqual({ value: 1.5, percent: null, direction: 'up' });
  });

  test('no previous period means no delta for counts and rates', () => {
    const cards = buildUsageCards(
      summary(
        counters(),
        counters({
          requests: 0,
          failures: 0,
          failure_rate: 0,
          total_tokens: 0,
          cost_micro_usd: 0,
        })
      ),
      'en'
    );
    expect(cards.every((card) => card.delta === null)).toBe(true);
  });

  test('the cost card carries the estimate and unpriced marks', () => {
    const marked = buildUsageCards(
      summary(
        counters({ estimated_cost_micro_usd: 200_000, unpriced_requests: 4 }),
        counters()
      ),
      'en'
    );
    expect(marked[2]).toMatchObject({ estimate: true, unpriced: 4 });
    const clean = buildUsageCards(summary(counters(), counters()), 'en');
    expect(clean[2]).toMatchObject({ estimate: false, unpriced: 0 });
  });
});

describe('section series and models', () => {
  test('the sparkline reads request totals in time order', () => {
    const points = [
      { ts_ms: 1, counters: counters({ requests: 2 }) },
      { ts_ms: 2, counters: counters({ requests: 5 }) },
    ];
    expect(
      usageSparkPoints({ bucket: 'hour', tz: 'UTC', points })
    ).toEqual([2, 5]);
    expect(usageSparkPoints(null)).toEqual([]);
  });

  test('top models keep the API order and the requested limit', () => {
    const items = [
      { key: 'a', label: 'a', share: 0.5, counters: counters() },
      { key: 'b', label: 'b', share: 0.3, counters: counters() },
      { key: 'c', label: 'c', share: 0.2, counters: counters() },
    ];
    expect(
      topUsageModels({ group_by: 'model', total: 3, items }, 2).map((item) => item.key)
    ).toEqual(['a', 'b']);
    expect(topUsageModels(null)).toEqual([]);
  });

  test('one day uses hour buckets; the longer dashboard ranges use days', () => {
    expect(dashboardUsageBucket('24h')).toBe('hour');
    expect(dashboardUsageBucket('7d')).toBe('day');
    expect(dashboardUsageBucket('30d')).toBe('day');
    expect(dashboardUsageQuery('24h', NOW, 'UTC')).toEqual({
      from_ms: NOW - 86_400_000,
      to_ms: NOW,
      tz: 'UTC',
    });
  });
});
