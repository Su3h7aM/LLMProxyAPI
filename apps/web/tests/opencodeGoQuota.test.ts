import { afterEach, describe, expect, test } from 'bun:test';
import type { TFunction } from 'i18next';
import { readFileSync } from 'node:fs';
import { OPENCODE_GO_CONFIG, buildOpencodeGoSessionId, parseOpencodeGoUsage } from '@/features/quota/providers/opencodeGo/data';
import { OPENCODE_GO_USAGE_URL } from '@/utils/quota';
import { apiCallApi, type ApiCallRequest, type ApiCallResult } from '@/services/api';

const t = ((key: string) => key) as unknown as TFunction;
const originalApiCallRequest = apiCallApi.request;

const result = (statusCode: number, body: unknown = null): ApiCallResult => ({
  statusCode,
  header: {},
  bodyText: body === null ? '' : JSON.stringify(body),
  body,
});

const usageBody = () => ({
  usage: {
    rolling: { status: 'ok', percent: 37, resetsAt: '2026-08-26T14:12:03.000Z' },
    weekly: { status: 'ok', percent: 0, resetsAt: '2026-09-01T00:00:00.000Z' },
    monthly: { status: 'rate-limited', percent: 99, resetsAt: '2026-09-01T01:26:45.000Z' },
  },
});

describe('OpenCode Go quota', () => {
  afterEach(() => {
    apiCallApi.request = originalApiCallRequest;
  });

  test('parses rolling, weekly and monthly windows', () => {
    const rows = parseOpencodeGoUsage(usageBody());
    expect(rows).toHaveLength(3);
    expect(rows[0]).toMatchObject({ id: 'rolling', used: 37, limit: 100 });
    expect(rows[0].resetAtMs).toBe(Date.parse('2026-08-26T14:12:03.000Z'));
    // A 0% window carries no meaningful reset.
    expect(rows[1]).toMatchObject({ id: 'weekly', used: 0, resetAtMs: null });
    // A rate-limited window reads as spent.
    expect(rows[2]).toMatchObject({ id: 'monthly', used: 100 });
  });

  test('empty usage throws', () => {
    expect(parseOpencodeGoUsage({ usage: {} })).toEqual([]);
    expect(parseOpencodeGoUsage(null)).toEqual([]);
  });

  test('session id follows the OpenCode shape', () => {
    const id = buildOpencodeGoSessionId();
    expect(id.startsWith('ses_')).toBe(true);
    expect(id).toHaveLength(4 + 12 + 14);
    expect(buildOpencodeGoSessionId()).not.toBe(id);
  });

  test('fetch asks the fixed usage route with session attribution', async () => {
    const requests: ApiCallRequest[] = [];
    apiCallApi.request = (async (req: ApiCallRequest) => {
      requests.push(req);
      return result(200, usageBody());
    }) as typeof apiCallApi.request;

    const file = { auth_index: 'abc123', provider: 'opencode-go' } as never;
    const summary = await OPENCODE_GO_CONFIG.fetchQuota(file, t);
    expect(requests).toHaveLength(1);
    expect(requests[0].url).toBe(OPENCODE_GO_USAGE_URL);
    expect(requests[0].authIndex).toBe('abc123');
    const session = requests[0].header?.['x-opencode-session'];
    expect(typeof session === 'string' && session.startsWith('ses_')).toBe(true);
    expect(requests[0].header?.['x-opencode-session-id']).toBe(session);
    expect(summary.planName).toBe('OpenCode Go');
    expect(summary.rows).toHaveLength(3);
  });

  test('missing auth_index throws', async () => {
    await expect(OPENCODE_GO_CONFIG.fetchQuota({ provider: 'opencode-go' } as never, t)).rejects.toThrow();
  });

  test('provides the provider name in every locale', () => {
    for (const locale of ['en', 'zh-CN', 'zh-TW', 'ru', 'vi', 'ko', 'tr']) {
      const messages = JSON.parse(readFileSync(`src/i18n/locales/${locale}.json`, 'utf8'));
      expect(messages.providersPage.providerNames.opencodeGo).toBe('OpenCode Go');
      expect(messages.auth_files['filter_opencode-go']).toBe('OpenCode Go');
    }
  });
});
