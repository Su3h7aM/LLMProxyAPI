import { describe, expect, test } from 'bun:test';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { I18nextProvider } from 'react-i18next';
import { createTestI18n, type TestLanguage } from './helpers/locales';
import { buildUsageSummaryCards } from '@/features/usage/logic/summaryCards';
import { buildUsageTimeline } from '@/features/usage/logic/timeline';
import { UsageSummaryCards } from '@/features/usage/components/UsageSummaryCards';
import type { UsageCounters, UsageSummaryResponse, UsageTimeseriesResponse } from '@/types/usageStats';

const LOCALES: TestLanguage[] = ['en', 'zh-CN', 'zh-TW', 'ru'];
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
  estimated_cost_micro_usd: 200_000,
  unpriced_requests: 4,
  latency_avg_ms: 840,
  latency_max_ms: 9_100,
  ttft_avg_ms: 310,
  ...overrides,
});

const summary = (overrides: Partial<UsageSummaryResponse> = {}): UsageSummaryResponse => ({
  range: { from_ms: NOW - 86_400_000, to_ms: NOW },
  totals: counters(),
  previous: counters({
    requests: 100,
    total_tokens: 150_000,
    cost_micro_usd: 1_000_000,
    failure_rate: 0.01,
    latency_avg_ms: 700,
  }),
  latency: { p50_ms: 600, p95_ms: 2_100, source: 'events' },
  first_event_ms: NOW - 86_400_000,
  ...overrides,
});

const t = (key: string) => key;

describe('usage summary cards', () => {
  test('six cards in reading order with deltas and marks', () => {
    const cards = buildUsageSummaryCards(summary(), 'en', t);
    expect(cards.map((card) => card.key)).toEqual([
      'requests',
      'tokens',
      'cost',
      'failures',
      'latency_avg',
      'latency_p95',
    ]);
    expect(cards[0].value).toBe('120');
    expect(cards[0].delta?.direction).toBe('up');
    expect(cards[2].estimate).toBe(true);
    expect(cards[2].unpriced).toBe(4);
    expect(cards[3].deltaUnit).toBe('points');
    expect(cards[4].delta?.direction).toBe('up');
    expect(cards[5].showDelta).toBe(false);
    expect(cards[1].detail).toContain('usage_page.tokens_input');
  });

  test('hourly percentiles render n/a with the hourly hint', () => {
    const cards = buildUsageSummaryCards(
      summary({ latency: { p50_ms: null, p95_ms: null, source: 'hourly' } }),
      'en',
      t
    );
    const p95 = cards.find((card) => card.key === 'latency_p95');
    expect(p95?.value).toBe('usage_page.latency_unavailable');
    expect(p95?.detail).toBe('usage_page.latency_hourly_hint');
  });

  test('empty previous period leaves deltas empty', () => {
    const cards = buildUsageSummaryCards(
      summary({ previous: counters({ requests: 0 }) }),
      'en',
      t
    );
    expect(cards[0].delta).toBeNull();
    expect(cards[3].delta).toBeNull();
  });
});

const point = (ts: number, requests: number, cost = requests * 1000) => ({
  ts_ms: ts,
  counters: counters({ requests, cost_micro_usd: cost, total_tokens: requests * 10 }),
});

const series = (overrides: Partial<UsageTimeseriesResponse> = {}): UsageTimeseriesResponse => ({
  bucket: 'hour',
  tz: 'UTC',
  points: [point(NOW - 3_600_000, 20), point(NOW, 50)],
  ...overrides,
});

describe('usage timeline model', () => {
  test('ungrouped points map the metric to one total series', () => {
    const model = buildUsageTimeline(series(), 'requests', 'Total');
    expect(model.series).toHaveLength(1);
    expect(model.series[0].id).toBe('total');
    expect(model.data.map((row) => row.values['total'])).toEqual([20, 50]);
    const cost = buildUsageTimeline(series(), 'cost', 'Total');
    expect(cost.data.map((row) => row.values['total'])).toEqual([20_000, 50_000]);
    const tokens = buildUsageTimeline(series(), 'tokens', 'Total');
    expect(tokens.data.map((row) => row.values['total'])).toEqual([200, 500]);
  });

  test('groups merge on timestamps, fill gaps with zero, and keep colors', () => {
    const model = buildUsageTimeline(
      series({
        points: [],
        groups: [
          { key: 'a', label: 'a', points: [point(NOW - 3_600_000, 20)] },
          {
            key: 'other',
            label: 'other',
            points: [point(NOW - 3_600_000, 1), point(NOW, 5)],
          },
        ],
      }),
      'requests',
      'Total'
    );
    expect(model.series.map((item) => item.id)).toEqual(['a', 'other']);
    expect(model.series[0].color).toBe('var(--chart-1)');
    expect(model.data).toHaveLength(2);
    expect(model.data[0].values).toEqual({ a: 20, other: 1 });
    expect(model.data[1].values).toEqual({ a: 0, other: 5 });
  });
});

describe('usage summary cards render', () => {
  const render = async (data: UsageSummaryResponse, language: TestLanguage = 'en') => {
    const i18n = await createTestI18n(language);
    return renderToStaticMarkup(
      createElement(I18nextProvider, { i18n }, createElement(UsageSummaryCards, { summary: data }))
    );
  };

  test('cards show values, deltas, and marks', async () => {
    const markup = await render(summary());
    for (const label of ['Requests', 'Tokens', 'Cost', 'Failure rate', 'Average latency', 'P95 latency']) {
      expect(markup).toContain(label);
    }
    expect(markup).toContain('Estimate');
    expect(markup).toContain('unpriced');
    expect(markup).toContain('↑');
  });

  test('every locale renders without raw keys', async () => {
    for (const language of LOCALES) {
      const markup = await render(summary(), language);
      expect(markup).not.toContain('usage_page.');
    }
  });
});
