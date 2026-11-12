import { describe, expect, test } from 'bun:test';
import { createElement, type ReactElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { I18nextProvider } from 'react-i18next';
import { createTestI18n, type TestLanguage } from './helpers/locales';
import { eventsExportQuery, eventsResetKey, mergeEventsPages } from '@/features/usage/logic/events';
import { UsageEventsTable } from '@/features/usage/components/UsageEvents';
import { EventDetailBody, EventPrice } from '@/features/usage/components/UsageEventDrawer';
import type { UsageEventDetail, UsageEventListItem } from '@/types/usageStats';

const LOCALES: TestLanguage[] = ['en', 'zh-CN', 'zh-TW', 'ru'];
const noop = () => {};

const event = (overrides: Partial<UsageEventListItem> = {}): UsageEventListItem => ({
  id: 7,
  request_id: '0b1c2d3e4f5a',
  trace_id: 'a1b2c3d4',
  ts_ms: Date.parse('2026-11-11T10:00:00.000Z'),
  provider: 'claude',
  model: 'claude-sonnet-5-5',
  alias: '',
  executor: '',
  auth_id: 'claude-main.json',
  auth_type: 'oauth',
  account_label: 'ab***@example.com',
  api_key_id: '9f2c4a0e11d3b7aa',
  api_key_label: '',
  stream: true,
  status_class: '2xx',
  failed: false,
  fail_code: '',
  latency_ms: 1280,
  ttft_ms: 340,
  input_tokens: 1200,
  output_tokens: 480,
  reasoning_tokens: 0,
  cache_read_tokens: 800,
  cache_write_tokens: 0,
  total_tokens: 2480,
  service_tier: '',
  reasoning_effort: '',
  cost_micro_usd: 12500,
  cost_source: 'price-book',
  ...overrides,
});

const render = async (element: ReactElement, language: TestLanguage = 'en') => {
  const i18n = await createTestI18n(language);
  return renderToStaticMarkup(createElement(I18nextProvider, { i18n }, element));
};

const table = (overrides: Partial<Parameters<typeof UsageEventsTable>[0]> = {}) =>
  createElement(UsageEventsTable, {
    items: [event(), event({ id: 8, failed: true, status_class: '5xx', cost_source: 'price-book-late' })],
    totalText: '2 of 5',
    hasMore: true,
    loadingMore: false,
    copiedId: null,
    onLoadMore: noop,
    onOpenEvent: noop,
    onCopyRequest: noop,
    ...overrides,
  });

describe('events helpers', () => {
  test('the reset key drops the moving bounds but keeps the filters and search', () => {
    const query = { from_ms: 1, to_ms: 2, tz: 'UTC', provider: ['a'] };
    expect(eventsResetKey(query, 'q')).toBe(eventsResetKey({ ...query, from_ms: 9, to_ms: 10 }, 'q'));
    expect(eventsResetKey(query, 'q')).not.toBe(eventsResetKey(query, 'other'));
    expect(eventsResetKey(query, 'q')).not.toBe(eventsResetKey({ ...query, provider: ['b'] }, 'q'));
  });

  test('pages append by cursor and replace on refetch', () => {
    const first = { items: [event()], next_cursor: 'c1', total_estimate: 3, total_exact: true };
    const second = { items: [event({ id: 8 })], next_cursor: '', total_estimate: 3, total_exact: true };
    const one = mergeEventsPages([], 'q|first', first);
    expect(one).toHaveLength(1);
    expect(one[0].next).toBe('c1');
    const two = mergeEventsPages(one, 'q|c1', second);
    expect(two).toHaveLength(2);
    expect(two.flatMap((page) => page.items.map((item) => item.id))).toEqual([7, 8]);
    const refetched = mergeEventsPages(two, 'q|c1', {
      ...second,
      items: [event({ id: 9 })],
    });
    expect(refetched).toHaveLength(2);
    expect(refetched[1].items.map((item) => item.id)).toEqual([9]);
    const emptyCursor = mergeEventsPages([], 'q|first', { ...first, next_cursor: '  ' });
    expect(emptyCursor[0].next).toBeNull();
  });

  test('the export query carries the filters, search, and format', () => {
    expect(eventsExportQuery({ from_ms: 1, to_ms: 2, tz: 'UTC' }, 'hi', 'csv')).toEqual({
      from_ms: 1,
      to_ms: 2,
      tz: 'UTC',
      q: 'hi',
      format: 'csv',
    });
    expect(eventsExportQuery({ from_ms: 1 }, '', 'jsonl')).toEqual({
      from_ms: 1,
      format: 'jsonl',
    });
  });
});

describe('event detail render', () => {
  const detail = (overrides: Partial<UsageEventDetail> = {}): UsageEventDetail => ({
    ...event(),
    fail_summary: 'Upstream rate limit',
    session_id: 's1',
    access_provider: 'claude',
    auth_index: '0',
    price_id: 3,
    price: {
      provider: 'claude',
      model: 'claude-sonnet-5-5',
      valid_from_ms: Date.parse('2026-11-01T00:00:00.000Z'),
      input: 3_000_000,
      output: 15_000_000,
      cache_read: 300_000,
      cache_write: 3_750_000,
      source: 'manual',
    },
    ...overrides,
  });

  test('shows all fields, the fail summary, and the per-million price', async () => {
    const markup = await render(createElement(EventDetailBody, { event: detail() }));
    expect(markup).toContain('0b1c2d3e4f5a');
    expect(markup).toContain('Upstream rate limit');
    expect(markup).toContain('ab***@example.com');
    const price = await render(createElement(EventPrice, { event: detail() }));
    expect(price).toContain('Price used');
    expect(price).toContain('$3.00 / 1M');
    expect(price).toContain('manual');
  });

  test('a missing price says so without raw numbers', async () => {
    const markup = await render(
      createElement(EventPrice, { event: detail({ price: null, price_id: null }) })
    );
    expect(markup).toContain('No price matched this event.');
  });

  test('every locale renders without raw keys', async () => {
    for (const language of LOCALES) {
      const markup = await render(createElement(EventDetailBody, { event: detail() }), language);
      expect(markup).not.toContain('usage_page.');
      const price = await render(createElement(EventPrice, { event: detail() }), language);
      expect(price).not.toContain('usage_page.');
    }
  });
});

describe('events table render', () => {
  test('rows name every column', async () => {
    const markup = await render(table());
    for (const label of [
      'Time',
      'Model',
      'Provider',
      'Credential',
      'Key',
      'Latency',
      'TTFT',
      'Tokens',
      'Cost',
      'Request',
    ]) {
      expect(markup).toContain(label);
    }
    expect(markup).toContain('claude-sonnet-5-5');
    expect(markup).toContain('0b1c2d3e4f5a');
    expect(markup).toContain('OK');
    expect(markup).toContain('Failed');
    expect(markup).toContain('Load more');
    expect(markup).toContain('2 of 5');
  });

  test('an unlabeled key shows the short fallback, never the raw id', async () => {
    const markup = await render(table());
    expect(markup).toContain('key-9f2c4a');
    expect(markup).not.toContain('9f2c4a0e11d3b7aa');
  });

  test('late pricing shows the estimate mark', async () => {
    const markup = await render(table());
    expect(markup).toContain('Estimate');
  });

  test('no raw API key appears anywhere in the output', async () => {
    const markup = await render(
      table({
        items: [
          event({ api_key_id: 'abcdef1234567890', api_key_label: '' }),
          event({ id: 9, api_key_id: 'abcdef1234567890', api_key_label: 'harness' }),
        ],
      })
    );
    expect(markup).not.toContain('abcdef1234567890');
    expect(markup).toContain('key-abcdef');
    expect(markup).toContain('harness');
  });

  test('the load-more control hides at the end', async () => {
    const markup = await render(table({ hasMore: false, totalText: '2 of 2' }));
    expect(markup).not.toContain('Load more');
    expect(markup).toContain('2 of 2');
  });

  test('every locale renders without raw keys', async () => {
    for (const language of LOCALES) {
      const markup = await render(table(), language);
      expect(markup).not.toContain('usage_page.');
    }
  });
});
