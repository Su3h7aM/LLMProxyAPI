import { describe, expect, test } from 'bun:test';
import {
  DEFAULT_USAGE_FILTERS,
  usageFiltersToQuery,
} from '@/features/usage/logic/filters';
import {
  getBrowserTimeZone,
  resolveUsageRange,
  USAGE_RANGE_MAX_MS,
  USAGE_RANGE_MS,
} from '@/features/usage/logic/range';
import {
  nextUsageCursor,
  toUsageQueryString,
  usageQueryPath,
  withUsageCursor,
} from '@/features/usage/logic/query';

const NOW = Date.parse('2026-11-11T12:00:00.000Z');
const DAY = 24 * 60 * 60 * 1000;

describe('range presets', () => {
  test('each preset ends now and covers its span', () => {
    for (const [preset, span] of Object.entries(USAGE_RANGE_MS)) {
      expect(
        resolveUsageRange({ preset: preset as keyof typeof USAGE_RANGE_MS }, NOW)
      ).toEqual({ from_ms: NOW - span, to_ms: NOW });
    }
  });

  test('a custom range is half-open and keeps the caller bounds', () => {
    expect(
      resolveUsageRange({ preset: 'custom', customFromMs: NOW - DAY, customToMs: NOW }, NOW)
    ).toEqual({ from_ms: NOW - DAY, to_ms: NOW });
  });

  test('an unusable custom range is rejected instead of clamped', () => {
    expect(resolveUsageRange({ preset: 'custom' }, NOW)).toBeNull();
    expect(
      resolveUsageRange({ preset: 'custom', customFromMs: NOW, customToMs: NOW }, NOW)
    ).toBeNull();
    expect(
      resolveUsageRange({ preset: 'custom', customFromMs: NOW, customToMs: NOW - DAY }, NOW)
    ).toBeNull();
    expect(
      resolveUsageRange(
        {
          preset: 'custom',
          customFromMs: NOW - USAGE_RANGE_MAX_MS - 1,
          customToMs: NOW,
        },
        NOW
      )
    ).toBeNull();
  });

  test('the browser zone is an IANA name, or UTC', () => {
    const zone = getBrowserTimeZone();
    expect(zone.length).toBeGreaterThan(0);
    expect(zone).toBe(new Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC');
  });
});

describe('filter to query mapping', () => {
  test('maps shared filters, dropping the all choices and invalid custom ranges', () => {
    const query = usageFiltersToQuery(
      {
        ...DEFAULT_USAGE_FILTERS,
        rangePreset: '24h',
        provider: ['claude'],
        model: ['sonnet'],
        credential: ['auth-1'],
        apiKey: ['9f2c4a0e11d3b7aa'],
        status: 'failed',
        stream: 'false',
      },
      NOW,
      'UTC'
    );
    expect(query).toEqual({
      from_ms: NOW - DAY,
      to_ms: NOW,
      tz: 'UTC',
      provider: ['claude'],
      model: ['sonnet'],
      auth_id: ['auth-1'],
      api_key_id: ['9f2c4a0e11d3b7aa'],
      status: 'failed',
      stream: false,
    });
    expect(
      usageFiltersToQuery(
        { ...DEFAULT_USAGE_FILTERS, rangePreset: 'custom' },
        NOW,
        'UTC'
      )
    ).toBeNull();
  });
});

describe('query string', () => {
  test('repeats multi-value parameters and keeps a fixed key order', () => {
    expect(
      toUsageQueryString({
        to_ms: 2,
        from_ms: 1,
        provider: ['b', 'a', 'a', ' '],
        model: ['m'],
        status: 'ok',
      })
    ).toBe('from_ms=1&to_ms=2&provider=b&provider=a&model=m&status=ok');
  });

  test('omits empty values and encodes reserved characters', () => {
    expect(
      toUsageQueryString({
        tz: 'Europe/Berlin',
        q: '  ',
        provider: [],
        stream: false,
        limit: 0,
      })
    ).toBe('tz=Europe%2FBerlin&stream=false&limit=0');
  });

  test('builds a full route path with and without a query', () => {
    expect(usageQueryPath('/summary', { from_ms: 1 })).toBe(
      '/observability/usage/summary?from_ms=1'
    );
    expect(usageQueryPath('/events')).toBe('/observability/usage/events');
  });

  test('cursor helpers stop at the end and keep the rest of the query', () => {
    expect(nextUsageCursor({ next_cursor: ' next ' })).toBe('next');
    expect(nextUsageCursor({ next_cursor: '  ' })).toBeNull();
    expect(withUsageCursor({ limit: 50, cursor: 'old' }, 'next')).toEqual({
      limit: 50,
      cursor: 'next',
    });
    expect(withUsageCursor({ limit: 50, cursor: 'old' }, null)).toEqual({ limit: 50 });
  });
});
