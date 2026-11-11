import { afterEach, beforeEach, describe, expect, test } from 'bun:test';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import '@/i18n';
import { useQuotaActions } from '@/features/quota/hooks/useQuotaActions';
import { useQuotaBatchLoader } from '@/features/quota/hooks/useQuotaBatchLoader';
import { classifyQuotaFiles } from '@/features/quota/logic';
import { QUOTA_ADAPTERS } from '@/features/quota/providers';
import { resolveQuotaEntryCheck } from '@/features/quota/refreshScope';
import { apiCallApi, authFilesApi, type ApiCallRequest, type ApiCallResult } from '@/services/api';
import { useQuotaStore } from '@/stores/useQuotaStore';
import type { AuthFileItem } from '@/types';
import { XAI_API_CHAT_URL, XAI_API_ME_URL } from '@/utils/quota';

function consumer() {
  let hooks!: {
    batch: ReturnType<typeof useQuotaBatchLoader>;
    actions: ReturnType<typeof useQuotaActions>;
  };
  function Capture() {
    hooks = { batch: useQuotaBatchLoader(), actions: useQuotaActions(false) };
    return null;
  }
  renderToStaticMarkup(createElement(Capture));
  return hooks;
}

const originalRequest = apiCallApi.request;
const originalList = authFilesApi.list;
const originalNow = Date.now;
const response = (statusCode: number, body: unknown): ApiCallResult => ({
  statusCode,
  body,
  bodyText: JSON.stringify(body),
  header: {},
});

beforeEach(() => useQuotaStore.getState().clearQuotaCache());
afterEach(() => {
  apiCallApi.request = originalRequest;
  authFilesApi.list = originalList;
  Date.now = originalNow;
  useQuotaStore.getState().clearQuotaCache();
});

describe('provider-owned non-generative observations', () => {
  for (const route of ['Dashboard', 'Quota']) {
    for (const mode of ['paid', 'free-billing-failure']) {
      test(`${route} entry and re-entry never generate inference for ${mode} xAI`, async () => {
        const requests: ApiCallRequest[] = [];
        apiCallApi.request = async (payload) => {
          requests.push(payload);
          return response(403, { error: 'billing unavailable' });
        };
        const file: AuthFileItem = {
          name: 'xai.json',
          provider: 'xai',
          authIndex: 'xai:1',
          ...(mode === 'paid' ? { using_api: true, prefix: 'paid' } : {}),
        };
        const generation = useQuotaStore.getState().cacheGeneration;
        for (let entry = 0; entry < 2; entry += 1) {
          const targets = resolveQuotaEntryCheck(
            null,
            false,
            {
              sessionGeneration: generation,
              filesGeneration: generation,
              hasError: false,
              disabled: false,
            },
            classifyQuotaFiles([file])
          );
          expect(targets).toHaveLength(mode === 'paid' ? 0 : 1);
          const result = await consumer().batch.loadQuota(targets!);
          if (mode !== 'paid') expect(result?.failed).toBe(1);
          expect(requests.every((request) => request.method === 'GET')).toBe(true);
          expect(requests.some((request) => request.url === XAI_API_CHAT_URL)).toBe(false);
          expect(useQuotaStore.getState().xaiQuota[file.name]?.checkedAt).toBeUndefined();
        }
        expect(requests).toHaveLength(mode === 'paid' ? 0 : 4);
      });
    }
  }

  test('mixed paid and observable xAI targets report skips separately from real failures', async () => {
    apiCallApi.request = async () => response(403, { error: 'billing unavailable' });
    const paid = {
      name: 'paid.json',
      provider: 'xai',
      authIndex: 'paid',
      using_api: true,
      prefix: 'paid',
    };
    const free = { name: 'free.json', provider: 'xai', authIndex: 'free' };
    const entries = classifyQuotaFiles([paid, free]);
    const generation = useQuotaStore.getState().cacheGeneration;
    const targets = resolveQuotaEntryCheck(
      null,
      false,
      {
        sessionGeneration: generation,
        filesGeneration: generation,
        hasError: false,
        disabled: false,
      },
      entries
    );
    expect(targets?.map(({ file }) => file.name)).toEqual(['free.json']);
    const result = await consumer().batch.loadQuota(entries);
    expect(result).toMatchObject({ success: 0, failed: 1, skipped: 1 });
    expect(useQuotaStore.getState().xaiQuota['paid.json']).toBeUndefined();
    expect(useQuotaStore.getState().xaiQuota['free.json']).toMatchObject({
      status: 'error',
      errorStatus: 403,
    });
  });

  test('explicit health action still sends the deliberate one-token paid inference test', async () => {
    const requests: ApiCallRequest[] = [];
    apiCallApi.request = async (payload) => {
      requests.push(payload);
      return response(
        200,
        payload.url === XAI_API_ME_URL ? { user_id: 'fixture' } : { choices: [] }
      );
    };
    const file = {
      name: 'paid.json',
      provider: 'xai',
      authIndex: 'xai:2',
      using_api: true,
      prefix: 'paid',
    };
    Date.now = () => 6000;
    await consumer().actions.testHealth(file, QUOTA_ADAPTERS.xai);
    const chat = requests.find((request) => request.method === 'POST');
    expect(chat?.url).toBe(XAI_API_CHAT_URL);
    expect(JSON.parse(chat!.data!)).toEqual({
      model: 'grok-4.5',
      messages: [{ role: 'user', content: 'ping' }],
      max_tokens: 1,
      stream: false,
    });
    expect(QUOTA_ADAPTERS.xai.healthTest?.labelFallback).toContain('may incur charges');
    const accepted = useQuotaStore.getState().xaiQuota[file.name];
    expect(accepted).toMatchObject({
      status: 'success',
      checkedAt: 6000,
      billing: { mode: 'paid-health', userId: 'fixture' },
    });
    const generation = useQuotaStore.getState().cacheGeneration;
    const targets = resolveQuotaEntryCheck(
      null,
      false,
      {
        sessionGeneration: generation,
        filesGeneration: generation,
        hasError: false,
        disabled: false,
      },
      classifyQuotaFiles([file])
    );
    expect(targets).toEqual([]);
    const result = await consumer().batch.loadQuota(classifyQuotaFiles([file]));
    expect(result).toMatchObject({ success: 0, failed: 0, skipped: 1 });
    await consumer().actions.refreshQuota(file, QUOTA_ADAPTERS.xai);
    expect(useQuotaStore.getState().xaiQuota[file.name]).toBe(accepted);
    expect(requests).toHaveLength(2);
  });

  test('manual quota observation cannot enter the free-billing health fallback either', async () => {
    const requests: ApiCallRequest[] = [];
    apiCallApi.request = async (payload) => {
      requests.push(payload);
      return response(403, { error: 'billing unavailable' });
    };
    await consumer().actions.refreshQuota(
      { name: 'free.json', provider: 'xai', authIndex: 'xai:3' },
      QUOTA_ADAPTERS.xai
    );
    expect(requests).toHaveLength(2);
    expect(requests.every((request) => request.method === 'GET')).toBe(true);
  });
});
