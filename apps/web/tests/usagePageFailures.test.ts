import { describe, expect, test } from 'bun:test';
import { createElement, type ReactElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { I18nextProvider } from 'react-i18next';
import { createTestI18n, type TestLanguage } from './helpers/locales';
import { buildHeatmapCells, heatmapCellLabel } from '@/features/usage/logic/heatmap';
import { UsageFailuresView } from '@/features/usage/components/UsageFailures';
import { UsageEventDrawer } from '@/features/usage/components/UsageEventDrawer';
import { UsageHeatmap } from '@/features/usage/components/UsageHeatmap';
import type {
  UsageFailureEvent,
  UsageFailureGroup,
  UsageEventListItem,
} from '@/types/usageStats';

const LOCALES: TestLanguage[] = ['en', 'zh-CN', 'zh-TW', 'ru'];
const noop = () => {};

const listItem = (overrides: Partial<UsageEventListItem> = {}): UsageEventListItem => ({
  id: 7,
  request_id: '0b1c2d3e',
  trace_id: 'a1b2c3d4',
  ts_ms: Date.parse('2026-11-11T10:00:00.000Z'),
  provider: 'claude',
  model: 'claude-sonnet-5-5',
  alias: '',
  executor: '',
  auth_id: 'claude-main.json',
  auth_type: 'oauth',
  account_label: 'ab***@example.com',
  api_key_id: '',
  api_key_label: '',
  stream: true,
  status_class: '5xx',
  failed: true,
  fail_code: 'rate_limit',
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

const group = (overrides: Partial<UsageFailureGroup> = {}): UsageFailureGroup => ({
  fail_code: 'rate_limit',
  status_class: '5xx',
  provider: 'claude',
  auth_id: 'claude-main.json',
  auth_label: 'ab***@example.com',
  model: 'claude-sonnet-5-5',
  count: 17,
  first_ms: Date.parse('2026-11-11T10:00:00.000Z'),
  last_ms: Date.parse('2026-11-11T11:00:00.000Z'),
  ...overrides,
});

const recent = (overrides: Partial<UsageFailureEvent> = {}): UsageFailureEvent => ({
  ...listItem(),
  fail_summary: 'Upstream rate limit',
  ...overrides,
});

const render = async (element: ReactElement, language: TestLanguage = 'en') => {
  const i18n = await createTestI18n(language);
  return renderToStaticMarkup(createElement(I18nextProvider, { i18n }, element));
};

describe('heatmap cells', () => {
  test('scales values against the peak and stays zero-safe', () => {
    const cells = buildHeatmapCells([
      [0, 10],
      [5, 0],
    ]);
    expect(cells).toHaveLength(4);
    expect(cells[1]).toEqual({ day: 0, hour: 1, value: 10, intensity: 1 });
    expect(cells[2].intensity).toBe(0.5);
    expect(cells[0].intensity).toBe(0);
    const flat = buildHeatmapCells([
      [0, 0],
      [0, 0],
    ]);
    expect(flat.every((cell) => cell.intensity === 0)).toBe(true);
  });

  test('labels name the weekday, hour, and value', () => {
    expect(heatmapCellLabel('Monday', 14, '123')).toBe('Monday, 14:00, 123');
  });
});

describe('failures view render', () => {
  const view = (overrides: Partial<Parameters<typeof UsageFailuresView>[0]> = {}) =>
    createElement(UsageFailuresView, {
      groups: [group()],
      recent: [recent()],
      onApplyGroup: noop,
      onOpenEvent: noop,
      ...overrides,
    });

  test('group rows name every column', async () => {
    const markup = await render(view());
    for (const label of [
      'Code',
      'Class',
      'Provider',
      'Credential',
      'Model',
      'Count',
      'First seen',
      'Last seen',
    ]) {
      expect(markup).toContain(label);
    }
    expect(markup).toContain('rate_limit');
    expect(markup).toContain('claude-sonnet-5-5');
    expect(markup).toContain('ab***@example.com');
    expect(markup).toContain('17');
  });

  test('recent rows show the failure with an open control', async () => {
    const markup = await render(view());
    expect(markup).toContain('Recent failures');
    expect(markup).toContain('Upstream rate limit');
    expect(markup).toContain('Open event');
  });

  test('empty groups and recent lists stay distinct', async () => {
    const markup = await render(view({ groups: [], recent: [] }));
    expect(markup).toContain('No failures in this range.');
    expect(markup).toContain('No recent failures.');
  });

  test('every locale renders without raw keys', async () => {
    for (const language of LOCALES) {
      const markup = await render(view(), language);
      expect(markup).not.toContain('usage_page.');
    }
  });
});

describe('event drawer render', () => {
  test('an open drawer names the detail without data', async () => {
    const markup = await render(createElement(UsageEventDrawer, { eventId: 7, onClose: noop }));
    expect(markup).toContain('Event detail');
  });

  test('a closed drawer renders nothing', async () => {
    const markup = await render(
      createElement(UsageEventDrawer, { eventId: null, onClose: noop })
    );
    expect(markup).not.toContain('Event detail');
  });
});

describe('heatmap render', () => {
  test('without data it shows the section with an empty state', async () => {
    const markup = await render(
      createElement(UsageHeatmap, {
        query: { from_ms: 1, to_ms: 2, tz: 'UTC' },
        queryKey: 'test',
        enabled: false,
        tz: 'UTC',
      })
    );
    expect(markup).toContain('Weekly pattern');
    expect(markup).toContain('Requests');
  });
});
