import { describe, expect, test } from 'bun:test';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import i18n from '@/i18n/index';
import { UsagePanelContent } from '@/features/dashboard/components/UsagePanel';
import type { DashboardUsageData } from '@/features/dashboard/hooks/useDashboardUsage';
import type { UsageCounters } from '@/types/usageStats';
import { UsageDisabledError } from '@/services/api/usage';

const NOW = Date.parse('2026-11-11T12:00:00.000Z');
const LOCALES = ['en', 'zh-CN', 'zh-TW', 'ru'];

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

const data = (overrides: Partial<DashboardUsageData> = {}): DashboardUsageData => ({
  summary: {
    range: { from_ms: NOW - 86_400_000, to_ms: NOW },
    totals: counters(),
    previous: counters({
      requests: 100,
      total_tokens: 150_000,
      cost_micro_usd: 1_000_000,
      failure_rate: 0.01,
    }),
    latency: { p50_ms: 600, p95_ms: 2_100, source: 'events' },
    first_event_ms: NOW - 90 * 86_400_000,
  },
  series: {
    bucket: 'hour',
    tz: 'UTC',
    points: [
      { ts_ms: NOW - 7_200_000, counters: counters({ requests: 20 }) },
      { ts_ms: NOW - 3_600_000, counters: counters({ requests: 50 }) },
      { ts_ms: NOW, counters: counters({ requests: 120 }) },
    ],
  },
  breakdown: {
    group_by: 'model',
    total: 2,
    items: [
      {
        key: 'claude-sonnet-5-5',
        label: 'claude-sonnet-5-5',
        share: 0.4,
        counters: counters({ requests: 48, total_tokens: 80_000 }),
        meta: { provider: 'claude' },
      },
      {
        key: 'gpt-6-luna',
        label: 'gpt-6-luna',
        share: 0.25,
        counters: counters({ requests: 30, total_tokens: 40_000 }),
        meta: { provider: 'codex' },
      },
    ],
  },
  ...overrides,
});

const render = (props: Parameters<typeof UsagePanelContent>[0]) =>
  renderToStaticMarkup(createElement(UsagePanelContent, props));

const baseProps = {
  range: '24h' as const,
  onRangeChange: () => undefined,
  onRetry: () => undefined,
};

describe('dashboard usage section', () => {
  test('renders the four cards, the trend, and the top models from the API data', async () => {
    await i18n.changeLanguage('en');
    const markup = render({ ...baseProps, data: data(), error: null, loading: false });

    expect(markup).toContain('Usage');
    expect(markup).toContain('Requests');
    expect(markup).toContain('Tokens');
    expect(markup).toContain('Cost');
    expect(markup).toContain('Failure rate');
    expect(markup).toContain('$1.25');
    expect(markup).toContain('177K');
    expect(markup).toContain('2.5%');
    expect(markup).toContain('↑ 20% vs previous');
    expect(markup).toContain('claude-sonnet-5-5');
    expect(markup).toContain('gpt-6-luna');
    expect(markup).toContain('40%');
    expect(markup).not.toMatch(/dashboard\.[a-z_]+/);
  });

  test('marks late pricing as an estimate and counts unpriced requests', async () => {
    await i18n.changeLanguage('en');
    const markup = render({ ...baseProps, data: data(), error: null, loading: false });
    expect(markup).toContain('Estimate');
    expect(markup).toContain('4 unpriced');
    expect(markup).toContain('price that arrived after the request');
    expect(markup).toContain('cost is incomplete');
  });

  test('an empty period is an empty state, not a zero card grid', async () => {
    await i18n.changeLanguage('en');
    const empty = data();
    empty.summary.totals = counters({
      requests: 0,
      failures: 0,
      total_tokens: 0,
      cost_micro_usd: 0,
      estimated_cost_micro_usd: 0,
      unpriced_requests: 0,
    });
    const markup = render({ ...baseProps, data: empty, error: null, loading: false });
    expect(markup).toContain('No usage events yet');
    expect(markup).not.toContain('Failure rate');
  });

  test('a disabled usage database is a distinct state', async () => {
    await i18n.changeLanguage('en');
    const markup = render({
      ...baseProps,
      data: null,
      error: new UsageDisabledError(),
      loading: false,
    });
    expect(markup).toContain('Usage database is off');
    expect(markup).toContain('Turn on the stored usage database');
    expect(markup).not.toContain('Usage data could not be loaded.');
  });

  test('a failure offers the message and a retry', async () => {
    await i18n.changeLanguage('en');
    const markup = render({
      ...baseProps,
      data: data(),
      error: new Error('usage summary failed'),
      loading: false,
    });
    expect(markup).toContain('role="alert"');
    expect(markup).toContain('Usage data could not be loaded.');
    expect(markup).toContain('usage summary failed');
    expect(markup).toContain('Retry');
  });

  test('the first load says that it is loading', async () => {
    await i18n.changeLanguage('en');
    const markup = render({ ...baseProps, data: null, error: null, loading: true });
    expect(markup).toContain('Loading usage statistics');
  });

  test('every locale renders translated text and keeps the range buttons pressed state', async () => {
    for (const language of LOCALES) {
      await i18n.changeLanguage(language);
      const markup = render({ ...baseProps, data: data(), error: null, loading: false });
      expect(markup).not.toMatch(/dashboard\.[a-z_]+/);
      expect(markup).not.toContain('{{');
      expect(markup).toMatch(/aria-pressed="true"/);
      expect(markup).toMatch(/aria-pressed="false"/);
    }
  });
});
