import { describe, expect, test } from 'bun:test';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import i18n from '@/i18n/index';
import { UsageDbCardContent } from '@/features/usage/components/UsageDbCard';
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

const baseProps = {
  status: null,
  error: null,
  loading: false,
  maintenance: null,
  maintenanceError: null,
  maintenanceRunning: null,
  onRefresh: () => undefined,
  onMaintenance: () => undefined,
};

const render = (props: Partial<Parameters<typeof UsageDbCardContent>[0]>) =>
  renderToStaticMarkup(createElement(UsageDbCardContent, { ...baseProps, ...props }));

describe('usage database card', () => {
  test('shows the healthy database values and maintenance actions', async () => {
    await i18n.changeLanguage('en');
    const markup = render({ state: 'ok', status: status() });

    expect(markup).toContain('data-state="ok"');
    expect(markup).toContain('/tmp/auths/usage.db');
    expect(markup).toContain('1.00 KB');
    expect(markup).toContain('90 days');
    expect(markup).toContain('2 / 4096');
    expect(markup).toContain('Prune now');
    expect(markup).toContain('Checkpoint');
    expect(markup).toContain('Vacuum');
    expect(markup).toContain('Never');
    expect(markup).not.toMatch(/usage_db\.[a-z_]+/);
  });

  test('a dropped record or write error turns the card into a warning', async () => {
    await i18n.changeLanguage('en');
    const markup = render({
      state: 'warning',
      status: status({ dropped: 3, write_errors: 1, last_error: 'disk full' }),
    });
    expect(markup).toContain('data-state="warning"');
    expect(markup).toContain('Attention: 3 dropped, 1 write errors.');
    expect(markup).toContain('disk full');
  });

  test('the disabled state explains the config key and hides the actions', async () => {
    await i18n.changeLanguage('en');
    const markup = render({ state: 'disabled', status: status({ enabled: false }) });
    expect(markup).toContain('data-state="disabled"');
    expect(markup).toContain('Usage database is disabled');
    expect(markup).toContain('observability.usage-db.enabled');
    expect(markup).not.toContain('Prune now');
  });

  test('the error state offers a retry with the message', async () => {
    await i18n.changeLanguage('en');
    const markup = render({ state: 'error', error: 'request failed' });
    expect(markup).toContain('data-state="error"');
    expect(markup).toContain('Could not load the usage database status.');
    expect(markup).toContain('request failed');
    expect(markup).toContain('Retry');
  });

  test('maintenance results name the action and its counts', async () => {
    await i18n.changeLanguage('en');
    const markup = render({
      state: 'ok',
      status: status(),
      maintenance: { action: 'prune', deleted: 1200, duration_ms: 340 },
    });
    expect(markup).toContain('Pruned 1200 events in 340 ms.');

    const checkpoint = render({
      state: 'ok',
      status: status(),
      maintenance: {
        action: 'checkpoint',
        busy: 0,
        log_frames: 10,
        checkpointed: 8,
        duration_ms: 12,
      },
    });
    expect(checkpoint).toContain('Checkpointed 8 of 10 frames (busy 0) in 12 ms.');

    const failed = render({
      state: 'ok',
      status: status(),
      maintenanceError: 'disk full',
    });
    expect(failed).toContain('Maintenance failed: disk full');
  });
});
