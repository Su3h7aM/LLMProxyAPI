import { describe, expect, test } from 'bun:test';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import i18n from '@/i18n/index';
import { PriceSyncPanel } from '@/features/usage/components/PriceSyncPanel';
import { PricesTable } from '@/features/usage/components/PricesTable';
import { UnmatchedPrices } from '@/features/usage/components/UnmatchedPrices';
import { CorrectionPanel } from '@/features/usage/components/CorrectionPanel';
import { PriceUnpricedPanel } from '@/features/usage/components/PriceUnpricedPanel';
import { PriceHistoryTable } from '@/features/usage/components/PriceHistorySheet';
import type { UsagePriceItem, UsagePriceStatusResponse } from '@/types/usageStats';

const NOW = Date.parse('2026-11-12T12:00:00.000Z');

const price = (overrides: Partial<UsagePriceItem> = {}): UsagePriceItem => ({
  id: 1,
  provider: 'claude',
  model: 'claude-sonnet-5-5',
  valid_from_ms: 0,
  valid_to_ms: null,
  input: 3_000_000,
  output: 15_000_000,
  cache_read: 75_000,
  cache_write: 3_750_000,
  reasoning: null,
  source: 'models.dev',
  synced_at_ms: NOW,
  note: '',
  tiers: [],
  ...overrides,
});

const status = (overrides: Partial<UsagePriceStatusResponse> = {}): UsagePriceStatusResponse => ({
  last_sync_ms: NOW,
  last_attempt_ms: NOW,
  last_error: '',
  counts_by_source: { 'models.dev': 12, manual: 2 },
  unmatched: [],
  ...overrides,
});

const render = (node: Parameters<typeof renderToStaticMarkup>[0]) =>
  renderToStaticMarkup(node);

describe('prices table', () => {
  test('renders rates, sources, and the actions of each row', async () => {
    await i18n.changeLanguage('en');
    const markup = render(
      createElement(PricesTable, {
        prices: [price(), price({ id: 2, model: 'claude-opus', source: 'manual' })],
        total: 120,
        page: 2,
        pages: 3,
        loading: false,
        filters: { provider: '', source: '', q: '' },
        filtered: false,
        onFiltersChange: () => undefined,
        onPageChange: () => undefined,
        onEdit: () => undefined,
        onHistory: () => undefined,
        onRemoveManual: () => undefined,
      })
    );

    expect(markup).toContain('claude-sonnet-5-5');
    expect(markup).toContain('$3.00');
    expect(markup).toContain('$0.075');
    expect(markup).toContain('models.dev');
    expect(markup).toContain('manual');
    expect(markup).toContain('120 current prices');
    expect(markup).toContain('Page 2 of 3');
    expect(markup).toContain('Edit');
    expect(markup).toContain('History');
    expect(markup).toContain('Remove manual');
    expect(markup).not.toMatch(/usage_prices\.[a-z_]+/);
  });

  test('only a manual row offers the remove action', async () => {
    await i18n.changeLanguage('en');
    const markup = render(
      createElement(PricesTable, {
        prices: [price()],
        total: 1,
        page: 1,
        pages: 1,
        loading: false,
        filters: { provider: '', source: '', q: '' },
        filtered: false,
        onFiltersChange: () => undefined,
        onPageChange: () => undefined,
        onEdit: () => undefined,
        onHistory: () => undefined,
        onRemoveManual: () => undefined,
      })
    );
    expect(markup).not.toContain('Remove manual');
  });

  test('a filtered empty list explains the filters, not an empty book', async () => {
    await i18n.changeLanguage('en');
    const markup = render(
      createElement(PricesTable, {
        prices: [],
        total: 0,
        page: 1,
        pages: 1,
        loading: false,
        filters: { provider: 'claude', source: '', q: '' },
        filtered: true,
        onFiltersChange: () => undefined,
        onPageChange: () => undefined,
        onEdit: () => undefined,
        onHistory: () => undefined,
        onRemoveManual: () => undefined,
      })
    );
    expect(markup).toContain('No prices match these filters');
  });
});

describe('price sync panel', () => {
  test('shows the last sync, source counts, and the read-only config', async () => {
    await i18n.changeLanguage('en');
    const markup = render(
      createElement(PriceSyncPanel, {
        status: status(),
        config: { enabled: true, url: 'https://models.dev/api.json', intervalHours: 24 },
        result: null,
        error: null,
        statusError: null,
        syncing: false,
        onSync: () => undefined,
      })
    );
    expect(markup).toContain('Last sync');
    expect(markup).toContain('models.dev: 12');
    expect(markup).toContain('manual: 2');
    expect(markup).toContain('Every 24 hours');
    expect(markup).toContain('Read-only here');
    expect(markup).toContain('Sync now');
  });

  test('renders the sync result counts and a failure message', async () => {
    await i18n.changeLanguage('en');
    const markup = render(
      createElement(PriceSyncPanel, {
        status: status({ last_error: 'fetch timeout' }),
        config: null,
        result: {
          added: 1,
          updated: 2,
          unchanged: 3,
          skipped_overrides: 4,
          error: '',
          duration_ms: 3,
        },
        error: 'network down',
        statusError: 'status failed',
        syncing: false,
        onSync: () => undefined,
      })
    );
    expect(markup).toContain('fetch timeout');
    expect(markup).toContain('Added 1, updated 2, unchanged 3, skipped overrides 4');
    expect(markup).toContain('Finished in 3 ms');
    expect(markup).toContain('Sync failed: network down');
    expect(markup).toContain('status failed');
  });
});

describe('unmatched models', () => {
  test('lists every model and its event count with an add action', async () => {
    await i18n.changeLanguage('en');
    const markup = render(
      createElement(UnmatchedPrices, {
        models: [
          { provider: 'codex', model: 'gpt-5u', events: 1 },
          { provider: 'kimi', model: 'kimi-k3-256k', events: 2 },
        ],
        onAdd: () => undefined,
      })
    );
    expect(markup).toContain('Models with no price');
    expect(markup).toContain('gpt-5u');
    expect(markup).toContain('1 event');
    expect(markup).toContain('2 events');
    expect(markup).toContain('Add price');
  });

  test('an empty list says every model has a price', async () => {
    await i18n.changeLanguage('en');
    const markup = render(
      createElement(UnmatchedPrices, { models: [], onAdd: () => undefined })
    );
    expect(markup).toContain('Every recorded model has a price.');
  });
});

describe('price history table', () => {
  test('shows every version with its valid range and the current marker', async () => {
    await i18n.changeLanguage('en');
    const markup = render(
      createElement(PriceHistoryTable, {
        locale: 'en',
        versions: [
          price({ id: 2, source: 'manual', valid_from_ms: NOW }),
          price({
            id: 1,
            valid_from_ms: 0,
            valid_to_ms: NOW,
            note: 'first sync',
          }),
        ],
      })
    );
    expect(markup).toContain('From');
    expect(markup).toContain('Current');
    expect(markup).toContain('first sync');
    expect(markup).toContain('$3.00');
    expect(markup.indexOf('manual')).toBeLessThan(markup.indexOf('models.dev'));
  });
});

describe('price maintenance panels', () => {
  test('the correction panel warns about frozen history and the typed word', async () => {
    await i18n.changeLanguage('en');
    const markup = render(
      createElement(CorrectionPanel, {
        result: null,
        error: null,
        running: false,
        onRun: () => undefined,
      })
    );
    expect(markup).toContain('History is frozen');
    expect(markup).toContain('Type correct to confirm');
    expect(markup).toContain('Correct costs');
    expect(markup).not.toMatch(/usage_prices\.[a-z_]+/);
  });

  test('the unpriced panel shows its counts', async () => {
    await i18n.changeLanguage('en');
    const markup = render(
      createElement(PriceUnpricedPanel, {
        result: { priced: 12, skipped_no_price: 2, duration_ms: 9 },
        error: null,
        running: false,
        onRun: () => undefined,
      })
    );
    expect(markup).toContain('Priced 12 events. 2 have no matching price.');
    expect(markup).toContain('Price unpriced events');
  });
});
