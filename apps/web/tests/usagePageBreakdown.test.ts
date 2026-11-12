import { describe, expect, test } from 'bun:test';
import { createElement, type ReactElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { I18nextProvider } from 'react-i18next';
import { createTestI18n, type TestLanguage } from './helpers/locales';
import {
  BREAKDOWN_PAGE_LIMIT,
  breakdownFilterPatch,
  breakdownQueryFor,
  fallbackKeyLabel,
  saveUsageKeyLabel,
} from '@/features/usage/logic/breakdown';
import { UsageBreakdownTable } from '@/features/usage/components/UsageBreakdown';
import { DEFAULT_USAGE_FILTERS } from '@/stores/useUsageFilterStore';
import type {
  UsageBreakdownItem,
  UsageCounters,
  UsageQueryParams,
} from '@/types/usageStats';

const LOCALES: TestLanguage[] = ['en', 'zh-CN', 'zh-TW', 'ru'];

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

const item = (overrides: Partial<UsageBreakdownItem> = {}): UsageBreakdownItem => ({
  key: 'claude-sonnet-5-5',
  label: 'claude-sonnet-5-5',
  share: 0.42,
  counters: counters(),
  ...overrides,
});

const base: UsageQueryParams = { from_ms: 1, to_ms: 2, tz: 'UTC' };
const noop = () => {};

describe('breakdown query', () => {
  test('maps the tab to the contract group with sort, order, and paging', () => {
    expect(breakdownQueryFor(base, 'api_keys', 'cost', 'asc', 20)).toEqual({
      ...base,
      group_by: 'api_key',
      sort: 'cost',
      order: 'asc',
      limit: BREAKDOWN_PAGE_LIMIT,
      offset: 20,
    });
    expect(breakdownQueryFor(base, 'status', 'requests', 'desc', 0).group_by).toBe(
      'status_class'
    );
  });
});

describe('breakdown filter patch', () => {
  test('row clicks add the group without duplicating it', () => {
    expect(breakdownFilterPatch('models', 'm', DEFAULT_USAGE_FILTERS)).toEqual({ model: ['m'] });
    expect(
      breakdownFilterPatch('models', 'm', { ...DEFAULT_USAGE_FILTERS, model: ['m'] })
    ).toEqual({ model: ['m'] });
    expect(breakdownFilterPatch('providers', 'p', DEFAULT_USAGE_FILTERS)).toEqual({
      provider: ['p'],
    });
    expect(breakdownFilterPatch('credentials', 'c', DEFAULT_USAGE_FILTERS)).toEqual({
      credential: ['c'],
    });
    expect(
      breakdownFilterPatch('api_keys', 'k', DEFAULT_USAGE_FILTERS)
    ).toEqual({ apiKey: ['k'] });
  });

  test('status classes fold onto ok and failed', () => {
    expect(breakdownFilterPatch('status', '2xx', DEFAULT_USAGE_FILTERS)).toEqual({
      status: 'ok',
    });
    expect(breakdownFilterPatch('status', '5xx', DEFAULT_USAGE_FILTERS)).toEqual({
      status: 'failed',
    });
  });
});

describe('key label save', () => {
  test('an empty label is a clear, never a raw id', () => {
    expect(fallbackKeyLabel('9f2c4a0e11d3b7aa')).toBe('key-9f2c4a');
  });

  test('applies the draft optimistically and keeps it on success', async () => {
    const writes: Array<[string, string]> = [];
    const seen: string[] = [];
    const saved = await saveUsageKeyLabel(
      {
        read: () => undefined,
        write: (key, label) => void writes.push([key, label]),
        putKeyLabel: async (id, label) => void seen.push(`${id}:${label}`),
      },
      { key: 'abc123', label: 'old' },
      '  new  '
    );
    expect(saved).toBe(true);
    expect(writes).toEqual([['abc123', 'new']]);
    expect(seen).toEqual(['abc123:new']);
  });

  test('a cleared label shows the short fallback while the server clears it', async () => {
    const writes: string[] = [];
    const saved = await saveUsageKeyLabel(
      {
        read: () => 'old',
        write: (_key, label) => void writes.push(label),
        putKeyLabel: async () => {},
      },
      { key: '9f2c4a0e11d3b7aa', label: 'old' },
      '   '
    );
    expect(saved).toBe(true);
    expect(writes).toEqual(['key-9f2c4a']);
  });

  test('a failure rolls back to the previous label', async () => {
    const writes: string[] = [];
    const saved = await saveUsageKeyLabel(
      {
        read: () => undefined,
        write: (_key, label) => void writes.push(label),
        putKeyLabel: async () => {
          throw new Error('nope');
        },
      },
      { key: 'abc123', label: 'old' },
      'new'
    );
    expect(saved).toBe(false);
    expect(writes).toEqual(['new', 'old']);
  });
});

describe('breakdown table render', () => {
  const render = async (element: ReactElement, language: TestLanguage = 'en') => {
    const i18n = await createTestI18n(language);
    return renderToStaticMarkup(createElement(I18nextProvider, { i18n }, element));
  };

  const table = (overrides: Partial<Parameters<typeof UsageBreakdownTable>[0]> = {}) =>
    createElement(UsageBreakdownTable, {
      tab: 'models',
      items: [item(), item({ key: 'b', label: 'b', share: 0.1 })],
      total: 12,
      offset: 0,
      overrides: {},
      editingId: null,
      draft: '',
      saving: false,
      onAddFilter: noop,
      onEdit: noop,
      onDraft: noop,
      onSave: noop,
      onCancel: noop,
      onPrev: noop,
      onNext: noop,
      ...overrides,
    });

  test('rows show every column with formatted values', async () => {
    const markup = await render(table());
    for (const label of [
      'Name',
      'Share',
      'Requests',
      'Tokens',
      'Cost',
      'Failure rate',
      'Avg latency',
    ]) {
      expect(markup).toContain(label);
    }
    expect(markup).toContain('claude-sonnet-5-5');
    expect(markup).toContain('42%');
    expect(markup).toContain('120');
    expect(markup).toContain('$1.25');
    expect(markup).toContain('Previous');
    expect(markup).toContain('Next');
    expect(markup).toContain('2 of 12');
  });

  test('paging buttons disable at the edges', async () => {
    const first = await render(table({ offset: 0, total: 2 }));
    expect(first).toContain('disabled=""');
    const last = await render(table({ offset: 10, total: 12 }));
    expect(last).toContain('disabled=""');
  });

  test('the API key tab shows labels with edit controls, never the raw id', async () => {
    const markup = await render(
      table({
        tab: 'api_keys',
        items: [item({ key: '9f2c4a0e11d3b7aa', label: 'my-harness' })],
        total: 1,
        overrides: { other: 'x' },
      })
    );
    expect(markup).toContain('my-harness');
    expect(markup).not.toContain('9f2c4a0e11d3b7aa');
  });

  test('an override replaces the server label', async () => {
    const markup = await render(
      table({
        tab: 'api_keys',
        items: [item({ key: 'abc', label: 'old' })],
        total: 1,
        overrides: { abc: 'renamed' },
      })
    );
    expect(markup).toContain('renamed');
    expect(markup).not.toContain('>old<');
  });

  test('edit mode shows the draft with save and cancel', async () => {
    const markup = await render(
      table({
        tab: 'api_keys',
        items: [item({ key: 'abc', label: 'old' })],
        total: 1,
        editingId: 'abc',
        draft: 'renamed',
      })
    );
    expect(markup).toContain('value="renamed"');
    expect(markup).toContain('aria-label="Save"');
    expect(markup).toContain('aria-label="Cancel"');
  });

  test('every locale renders without raw keys', async () => {
    for (const language of LOCALES) {
      const markup = await render(table(), language);
      expect(markup).not.toContain('usage_page.');
    }
  });
});
