import { afterEach, beforeEach, describe, expect, spyOn, test } from 'bun:test';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { I18nextProvider } from 'react-i18next';
import type { Mock } from 'bun:test';
import { apiClient } from '@/services/api/client';
import { authFilesApi } from '@/services/api/authFiles';
import { pluginsApi } from '@/services/api/plugins';
import { useAuthStore } from '@/stores/useAuthStore';
import { useQuotaStore } from '@/stores/useQuotaStore';
import {
  PLUGIN_CONFIG,
  normalizePluginQuota,
  normalizePluginQuotaSummary,
  resetPluginQuotaRouteCache,
  resolveQuotaPluginId,
} from '@/features/quota/providers/plugin/data';
import { PluginQuotaBody } from '@/features/quota/providers/plugin/PluginQuotaBody';
import { classifyQuotaFiles, resolveQuotaProviderType } from '@/features/quota/logic';
import { resolveAuthFileQuotaType } from '@/features/authFiles/logic';
import { getQuotaCacheKey } from '@/utils/quota/identity';
import { requestQuota } from '@/features/quota/hooks/quotaRequests';
import { selectQuotaWindow } from '@/features/quota/providers/observations';
import { buildTimelineLane } from '@/features/quota/quotaTimelineModel';
import { QUOTA_CLASS_KEYS, type QuotaClassMap } from '@/features/quota/types';
import type { PluginListEntry, PluginListResponse } from '@/types';
import { createTestI18n } from './helpers/locales';

const i18n = await createTestI18n();
await i18n.changeLanguage('en');
const t = i18n.t.bind(i18n);
const quotaPlugin = (overrides: Partial<PluginListEntry> = {}): PluginListEntry => ({
  id: 'kiro-quota',
  path: '/plugins/kiro-quota',
  configured: true,
  registered: true,
  enabled: true,
  effectiveEnabled: true,
  supportsOAuth: false,
  supportsQuota: true,
  quotaProvider: 'kiro',
  logo: '',
  configFields: [],
  menus: [],
  metadata: null,
  ...overrides,
});
const pluginList = (plugins: PluginListEntry[]): PluginListResponse => ({
  pluginsEnabled: true,
  pluginsDir: 'plugins',
  plugins,
});
let list: Mock<typeof pluginsApi.list>;
const file = {
  name: 'custom.json',
  provider: 'custom-provider',
  authIndex: 'fixture-index',
  supportsQuota: true,
  quotaProvider: 'quota-provider',
};
const classes = Object.fromEntries(
  QUOTA_CLASS_KEYS.map((key) => [key, key])
) as unknown as QuotaClassMap;
const originalStorage = useAuthStore.persist.getOptions().storage;
beforeEach(() => {
  resetPluginQuotaRouteCache();
  list = spyOn(pluginsApi, 'list').mockResolvedValue(pluginList([quotaPlugin()]));
  useAuthStore.persist.setOptions({
    storage: { getItem: () => null, setItem: () => {}, removeItem: () => {} },
  });
});
afterEach(() => {
  list.mockRestore();
  resetPluginQuotaRouteCache();
  useAuthStore.persist.setOptions({ storage: originalStorage });
  useQuotaStore.getState().clearQuotaCache();
});

function markup(data: ReturnType<typeof normalizePluginQuota>) {
  return renderToStaticMarkup(
    createElement(
      I18nextProvider,
      { i18n },
      createElement(PluginQuotaBody, { quota: PLUGIN_CONFIG.buildSuccessState(data), classes })
    )
  );
}

describe('generic plugin quota contract', () => {
  test('dispatches the v8 plugin quota route with normalized identity', async () => {
    const post = spyOn(apiClient, 'post').mockResolvedValue({
      summary: [
        { key: 'balance', label: 'Balance', value: 0, format: 'currency', currency: 'USD' },
      ],
    });
    try {
      const data = await PLUGIN_CONFIG.fetchQuota(file, t);
      expect(post).toHaveBeenCalledWith('/plugins/kiro-quota/quota', {
        auth_index: 'fixture-index',
      });
      expect(data.summary).toEqual([
        {
          key: 'balance',
          label: 'Balance',
          value: 0,
          format: 'currency',
          currency: 'USD',
          unit: undefined,
        },
      ]);
      expect(data.groups).toEqual([]);
    } finally {
      post.mockRestore();
    }
  });

  test('reports probe-only credentials as unsupported without a quota route', async () => {
    const post = spyOn(apiClient, 'post');
    try {
      await expect(
        PLUGIN_CONFIG.fetchQuota({ ...file, quotaProvider: undefined }, t)
      ).rejects.toThrow(t('plugin_quota.probe_unsupported'));
      expect(post).not.toHaveBeenCalled();
      expect(list).not.toHaveBeenCalled();
    } finally {
      post.mockRestore();
    }
  });

  test('disabled and unsupported credentials never dispatch', async () => {
    const post = spyOn(apiClient, 'post');
    try {
      await expect(PLUGIN_CONFIG.fetchQuota({ ...file, disabled: true }, t)).rejects.toThrow();
      expect(
        await requestQuota({ ...file, disabled: true }, PLUGIN_CONFIG as never, t)
      ).toMatchObject({ status: 'error', message: t('plugin_quota.unavailable') });
      await expect(
        PLUGIN_CONFIG.fetchQuota({ ...file, supportsQuota: false }, t)
      ).rejects.toThrow();
      await expect(PLUGIN_CONFIG.fetchQuota({ ...file, authIndex: '' }, t)).rejects.toThrow();
      expect(post).not.toHaveBeenCalled();
    } finally {
      post.mockRestore();
    }
  });

  test('normalizes advertised snake-case capability at the API boundary', async () => {
    const get = spyOn(apiClient, 'get').mockResolvedValue({
      files: [
        {
          name: 'custom.json',
          type: 'custom-provider',
          auth_index: '1',
          supports_quota: '1',
          quota_provider: 'custom',
        },
      ],
    });
    try {
      const result = await authFilesApi.list();
      expect(result.files[0]).toMatchObject({
        supportsQuota: true,
        quotaProvider: 'custom',
        authIndex: '1',
      });
      expect(resolveQuotaProviderType(result.files[0])).toBe('plugin');
    } finally {
      get.mockRestore();
    }
  });

  test('built-in adapters precede generic capabilities and disabled files stay excluded', () => {
    for (const provider of [
      'claude',
      'codex',
      'kimi',
      'kimi-ai',
      'antigravity',
      'github-copilot',
      'xai',
      'devin',
      'meta',
      'qoder',
    ]) {
      const item = { ...file, provider };
      expect(resolveQuotaProviderType(item)).not.toBe('plugin');
      if (provider !== 'devin') expect(getQuotaCacheKey(item)).toBe(item.name);
    }
    expect(classifyQuotaFiles([{ ...file, disabled: true }])).toEqual([]);
    expect(resolveQuotaProviderType({ ...file, supportsQuota: false })).toBeNull();
    expect(
      classifyQuotaFiles([file, { name: 'claude.json', provider: 'claude' }]).map(
        (entry) => entry.type
      )
    ).toEqual(['claude', 'plugin']);
  });

  test('auth-file provider filtering retains generic quota access', () => {
    expect(resolveAuthFileQuotaType(file, 'all')).toBe('plugin');
    expect(resolveAuthFileQuotaType(file, 'custom-provider')).toBe('plugin');
    expect(resolveAuthFileQuotaType(file, 'plugin')).toBe('plugin');
    expect(resolveAuthFileQuotaType(file, 'claude')).toBeNull();
  });

  test('cache identities distinguish credentials in one file and invalidation removes them together', () => {
    const other = { ...file, authIndex: 'second' };
    const firstKey = getQuotaCacheKey(file);
    const secondKey = getQuotaCacheKey(other);
    expect(firstKey).not.toBe(secondKey);
    useQuotaStore.getState().setPluginQuota({
      [firstKey]: PLUGIN_CONFIG.buildLoadingState(),
      [secondKey]: PLUGIN_CONFIG.buildLoadingState(),
    });
    useQuotaStore.getState().clearQuotaCache([file.name]);
    expect(useQuotaStore.getState().pluginQuota).toEqual({});
  });

  test('resolves the quota plugin like the backend first pass, then a sole provider', () => {
    const plugins = [
      quotaPlugin({ id: 'unregistered', quotaProvider: 'kiro', registered: false }),
      quotaPlugin({ id: 'oauth-match', quotaProvider: 'other', oauthProvider: 'kiro' }),
      quotaPlugin({ id: 'kiro', quotaProvider: 'other-2' }),
      quotaPlugin({ id: 'quota-match', quotaProvider: 'kiro' }),
      quotaPlugin({ id: 'no-quota', quotaProvider: 'kiro', supportsQuota: false }),
    ];
    expect(resolveQuotaPluginId(plugins, ' Kiro ')).toBe('quota-match');
    expect(resolveQuotaPluginId(plugins.slice(0, 3), 'kiro')).toBe('kiro');
    expect(resolveQuotaPluginId(plugins.slice(0, 2), 'kiro')).toBe('oauth-match');
    expect(resolveQuotaPluginId(plugins, 'unknown')).toBeNull();
    expect(resolveQuotaPluginId([quotaPlugin({ id: 'only', quotaProvider: 'x' })], 'y')).toBe(
      'only'
    );
    expect(resolveQuotaPluginId(plugins, '')).toBeNull();
  });

  test('shares the plugin list across credentials and reports a missing plugin', async () => {
    const post = spyOn(apiClient, 'post').mockResolvedValue({});
    try {
      await Promise.all(
        ['1', '2', '3'].map((authIndex) =>
          PLUGIN_CONFIG.fetchQuota({ ...file, name: `kiro-${authIndex}.json`, authIndex }, t)
        )
      );
      expect(list).toHaveBeenCalledTimes(1);
      expect(post).toHaveBeenCalledTimes(3);

      list.mockResolvedValue(
        pluginList([
          quotaPlugin({ id: 'first', quotaProvider: 'other' }),
          quotaPlugin({ id: 'second', quotaProvider: 'other-2' }),
        ])
      );
      resetPluginQuotaRouteCache();
      await expect(PLUGIN_CONFIG.fetchQuota(file, t)).rejects.toThrow(
        t('plugin_quota.plugin_not_found', { provider: 'quota-provider' })
      );
    } finally {
      post.mockRestore();
    }
  });

  test('request ownership suppresses a stale response after connection invalidation', async () => {
    let resolve!: (value: unknown) => void;
    const post = spyOn(apiClient, 'post').mockImplementation(
      () =>
        new Promise((done) => {
          resolve = done;
        })
    );
    try {
      const pending = requestQuota(file, PLUGIN_CONFIG as never, t);
      while (post.mock.calls.length === 0) await Promise.resolve();
      useQuotaStore.getState().clearQuotaCache();
      resolve({ summary: [{ key: 'balance', label: 'Balance', value: 10 }] });
      expect(await pending).toEqual({ status: 'superseded' });
      expect(useQuotaStore.getState().pluginQuota).toEqual({});
    } finally {
      post.mockRestore();
    }
  });
});

describe('generic quota normalization and rendering', () => {
  test('preserves zero summary values and rejects malformed entries', () => {
    const metrics = normalizePluginQuotaSummary([
      null,
      3,
      { key: 'bad', label: 'Bad', value: '12' },
      { key: 'nan', label: 'NaN', value: NaN },
      { key: 'zero', label: 'Zero', value: 0, currency: 'bad!' },
      { key: 'balance', label: 'Balance', value: 12.5, format: 'currency', currency: 'USD' },
    ]);
    expect(metrics).toHaveLength(2);
    expect(metrics[0]).toMatchObject({ value: 0, currency: undefined });
    expect(metrics[1]).toMatchObject({ value: 12.5, currency: 'USD' });
  });

  test('normalizes generic camel/snake buckets without Antigravity labels or recurrence', () => {
    const data = normalizePluginQuota({
      groups: [
        {
          display_name: 'Messages',
          buckets: [
            { window: 'Weekly', remaining_fraction: 0.25, reset_time: '2099-01-02T00:00:00Z' },
            { window: 'Unknown', resetTime: 'bad' },
            { window: 'Zero', remainingFraction: 0 },
          ],
        },
      ],
    });
    const state = PLUGIN_CONFIG.buildSuccessState(data);
    const observations = PLUGIN_CONFIG.getObservations(state);
    expect(observations.windows.map((window) => window.periodHours)).toEqual([null, null, null]);
    expect(observations.windows.map((window) => window.remainingPercent)).toEqual([25, null, 0]);
    expect(observations.windows[1].resetAtMs).toBeNull();
    expect(observations.windows[0].label).toBe('Weekly');
    expect(
      buildTimelineLane({
        provider: 'plugin',
        name: file.name,
        displayName: 'Custom',
        quota: state,
      }).periodHours
    ).toBeNull();
  });

  test('a reported allowance without a reset remains a snapshot, not a fabricated clock', () => {
    const state = PLUGIN_CONFIG.buildSuccessState(
      normalizePluginQuota({ groups: [{ buckets: [{ remainingFraction: 0.6 }] }] })
    );
    const selected = selectQuotaWindow(PLUGIN_CONFIG.getObservations(state));
    expect(selected?.remainingPercent).toBe(60);
    expect(selected?.resetAtMs).toBeNull();
    expect(selected?.periodHours).toBeNull();
  });

  test('supplemental currency summaries never become an allowance percentage', () => {
    const data = normalizePluginQuota({
      subscription: { tier_name: 'Custom plan' },
      summary: [
        { key: 'balance', label: 'Balance', value: 42, format: 'currency', currency: 'USD' },
      ],
    });
    expect(PLUGIN_CONFIG.getObservations(PLUGIN_CONFIG.buildSuccessState(data)).windows).toEqual(
      []
    );
    const html = markup(data);
    expect(html).toContain('Custom plan');
    expect(html).toContain('Balance');
    expect(html).toContain('$42.00');
    expect(html).not.toContain('42%');
  });

  test('unknown percentages stay unknown while zero remains exhausted', () => {
    const html = markup(
      normalizePluginQuota({
        groups: [
          {
            displayName: 'Custom limits',
            buckets: [{ window: 'Unknown' }, { window: 'Exhausted', remainingFraction: 0 }],
          },
        ],
      })
    );
    expect(html).toContain('Custom limits');
    expect(html).toContain('Not reported');
    expect(html).toContain('0% remaining');
    expect(html).not.toContain('Invalid Date');
  });

  test('empty/malformed data renders the empty state without inventing account data', () => {
    for (const payload of [null, [], {}, { groups: [null, { buckets: 'bad' }], summary: [null] }]) {
      const data = normalizePluginQuota(payload);
      expect(data.groups).toEqual([]);
      expect(data.summary).toEqual([]);
      expect(markup(data)).toContain('No quota data reported');
    }
  });
});
