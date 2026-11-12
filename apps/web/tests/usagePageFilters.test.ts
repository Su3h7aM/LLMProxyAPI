import { afterEach, describe, expect, test } from 'bun:test';
import { createElement, type ReactElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { I18nextProvider } from 'react-i18next';
import { createTestI18n, type TestLanguage } from './helpers/locales';
import { UsageFilterBar } from '@/features/usage/components/UsageFilterBar';
import { buildUsageFilterChips, fromDateTimeLocal, optionBaseQuery, resolveCustomDraft, toDateTimeLocal } from '@/features/usage/logic/filterBar';
import { UsagePageContent } from '@/features/usage/UsagePage';
import { DEFAULT_USAGE_FILTERS, useUsageFilterStore } from '@/stores/useUsageFilterStore';
import { UsageDisabledError } from '@/services/api/usage';
import type { UsageFilterValues } from '@/features/usage/logic/filters';

const LOCALES: TestLanguage[] = ['en', 'zh-CN', 'zh-TW', 'ru'];
const NOW = Date.parse('2026-11-11T12:00:00.000Z');

const filters = (overrides: Partial<UsageFilterValues> = {}): UsageFilterValues => ({
  ...DEFAULT_USAGE_FILTERS,
  ...overrides,
});

const render = async (element: ReactElement, language: TestLanguage = 'en') => {
  const i18n = await createTestI18n(language);
  return renderToStaticMarkup(createElement(I18nextProvider, { i18n }, element));
};

afterEach(() => {
  useUsageFilterStore.getState().reset();
});

describe('usage datetime-local helpers', () => {
  test('round-trips through the local input format', () => {
    const ms = new Date(2026, 4, 9, 8, 5, 0, 0).getTime();
    expect(fromDateTimeLocal(toDateTimeLocal(ms))).toBe(ms);
    expect(toDateTimeLocal(ms)).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/);
  });

  test('rejects malformed input', () => {
    expect(fromDateTimeLocal('')).toBeNull();
    expect(fromDateTimeLocal('2026-11-11')).toBeNull();
    expect(fromDateTimeLocal('not a date')).toBeNull();
    expect(toDateTimeLocal(Number.NaN)).toBe('');
  });
});

describe('usage option queries', () => {
  test('keep range, zone, status and stream but drop the dimension filters', () => {
    const query = optionBaseQuery(
      filters({
        rangePreset: '7d',
        provider: ['claude'],
        model: ['sonnet'],
        credential: ['a.json'],
        apiKey: ['id'],
        status: 'failed',
        stream: 'true',
      }),
      NOW,
      'Europe/Berlin'
    );
    expect(query).toEqual({
      from_ms: NOW - 7 * 24 * 60 * 60 * 1000,
      to_ms: NOW,
      tz: 'Europe/Berlin',
      status: 'failed',
      stream: true,
    });
  });

  test('is null for an unusable custom range', () => {
    expect(
      optionBaseQuery(
        filters({ rangePreset: 'custom', customFromMs: null, customToMs: NOW }),
        NOW,
        'UTC'
      )
    ).toBeNull();
  });
});

describe('usage filter bar', () => {
  test('renders presets, dimensions, status, stream and reset', async () => {
    const markup = await render(
      createElement(UsageFilterBar, { enabled: true, nowMs: NOW })
    );
    for (const label of ['1h', '24h', '7d', '30d', '90d', 'Custom']) {
      expect(markup).toContain(label);
    }
    for (const label of ['Provider', 'Model', 'Credential', 'API key', 'Status', 'Stream']) {
      expect(markup).toContain(label);
    }
    expect(markup).toContain('Reset');
  });

  test('shows active filters as removable chips', async () => {
    const state = useUsageFilterStore.getState();
    const chips = buildUsageFilterChips(
      filters({ provider: ['claude'], status: 'failed' }),
      (key: string) => key,
      state
    );
    expect(chips.map((chip) => chip.id)).toEqual(['provider:claude', 'status:failed']);
    expect(chips.map((chip) => chip.text)).toEqual(['claude', 'usage_page.status_failed']);
    chips[0].onRemove();
    expect(useUsageFilterStore.getState().provider).toEqual([]);
    chips[1].onRemove();
    expect(useUsageFilterStore.getState().status).toBe('all');
  });

  test('the custom draft resolves to bounds or null', () => {
    const from = toDateTimeLocal(NOW - 3_600_000);
    const to = toDateTimeLocal(NOW);
    const valid = resolveCustomDraft(from, to, filters(), NOW, 'UTC');
    expect(valid).toEqual({
      from_ms: NOW - 3_600_000,
      to_ms: NOW,
      tz: 'UTC',
      provider: [],
      model: [],
      auth_id: [],
      api_key_id: [],
    });
    expect(resolveCustomDraft('nope', to, filters(), NOW, 'UTC')).toBeNull();
    expect(resolveCustomDraft(to, from, filters(), NOW, 'UTC')).toBeNull();
  });
});

describe('usage page states', () => {
  test('loading shows skeletons', async () => {
    const markup = await render(
      createElement(UsagePageContent, {
        body: { data: null, error: null, loading: true },
        onRetry: () => {},
      })
    );
    expect(markup).toContain('role="status"');
    expect(markup).toContain('Loading usage statistics');
  });

  test('a 503 names the usage database and the config key', async () => {
    const markup = await render(
      createElement(UsagePageContent, {
        body: { data: null, error: new UsageDisabledError(), loading: false },
        onRetry: () => {},
      })
    );
    expect(markup).toContain('Usage database is off');
    expect(markup).toContain('<code>observability.usage-db.enabled</code>');
  });

  test('an error shows the message with retry', async () => {
    const markup = await render(
      createElement(UsagePageContent, {
        body: { data: null, error: new Error('boom'), loading: false },
        onRetry: () => {},
      })
    );
    expect(markup).toContain('role="alert"');
    expect(markup).toContain('boom');
    expect(markup).toContain('Retry');
  });

  test('an empty range shows the empty state', async () => {
    const markup = await render(
      createElement(UsagePageContent, {
        body: { data: { totals: { requests: 0 } }, error: null, loading: false },
        onRetry: () => {},
      })
    );
    expect(markup).toContain('No usage events yet');
  });

  test('every locale renders the states without raw keys', async () => {
    for (const language of LOCALES) {
      const markup = await render(
        createElement(UsagePageContent, {
          body: { data: null, error: new UsageDisabledError(), loading: false },
          onRetry: () => {},
        }),
        language
      );
      expect(markup).not.toContain('usage_page.');
    }
  });
});
