import { describe, expect, test } from 'bun:test';
import {
  hasUsageDbWarnings,
  usageDbCardState,
} from '@/features/usage/logic/usageDb';
import type { UsageDbStatus } from '@/types/usageStats';

const status = (overrides: Partial<UsageDbStatus> = {}): UsageDbStatus => ({
  enabled: true,
  path: '/tmp/auths/usage.db',
  size_bytes: 1024,
  schema_version: 2,
  events: 120,
  oldest_ts_ms: 1_700_000_000_000,
  newest_ts_ms: 1_700_000_100_000,
  buffer_depth: 2,
  buffer_capacity: 4096,
  dropped: 0,
  write_errors: 0,
  last_error: '',
  retention_days: 90,
  last_prune_ts_ms: 0,
  ...overrides,
});

describe('usage database card state', () => {
  test('disabled wins over data and errors', () => {
    expect(
      usageDbCardState({ status: status({ enabled: false }), error: null, loading: false, disabled: false })
    ).toBe('disabled');
    expect(
      usageDbCardState({ status: status(), error: null, loading: false, disabled: true })
    ).toBe('disabled');
  });

  test('a dropped record or a write error is a warning', () => {
    expect(
      usageDbCardState({ status: status({ dropped: 1 }), error: null, loading: false, disabled: false })
    ).toBe('warning');
    expect(
      usageDbCardState({
        status: status({ write_errors: 2 }),
        error: null,
        loading: false,
        disabled: false,
      })
    ).toBe('warning');
    expect(hasUsageDbWarnings(status({ last_error: 'disk full' }))).toBe(false);
  });

  test('healthy, loading, and error states', () => {
    expect(
      usageDbCardState({ status: status(), error: null, loading: false, disabled: false })
    ).toBe('ok');
    expect(
      usageDbCardState({ status: null, error: null, loading: true, disabled: false })
    ).toBe('loading');
    expect(
      usageDbCardState({
        status: null,
        error: new Error('boom'),
        loading: false,
        disabled: false,
      })
    ).toBe('error');
  });
});
