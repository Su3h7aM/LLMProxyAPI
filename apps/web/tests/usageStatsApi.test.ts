import { afterEach, describe, expect, spyOn, test } from 'bun:test';
import { apiClient } from '@/services/api/client';
import {
  isUsageDisabledError,
  UsageDisabledError,
  usageApi,
} from '@/services/api/usage';

const restore: Array<() => void> = [];

afterEach(() => {
  while (restore.length > 0) restore.pop()?.();
});

const mockGet = (value: unknown) => {
  const spy = spyOn(apiClient, 'get').mockResolvedValue(value);
  restore.push(() => spy.mockRestore());
  return spy;
};

const mockPut = (value: unknown) => {
  const spy = spyOn(apiClient, 'put').mockResolvedValue(value);
  restore.push(() => spy.mockRestore());
  return spy;
};

const mockPost = (value: unknown) => {
  const spy = spyOn(apiClient, 'post').mockResolvedValue(value);
  restore.push(() => spy.mockRestore());
  return spy;
};

const mockDelete = (value: unknown) => {
  const spy = spyOn(apiClient, 'delete').mockResolvedValue(value);
  restore.push(() => spy.mockRestore());
  return spy;
};

describe('usage query building', () => {
  test('repeats multi-value filters, omits empty values, and keeps contract order', async () => {
    const get = mockGet({});
    await usageApi.getSummary(
      {
        from_ms: 1000,
        to_ms: 2000,
        tz: 'Europe/Berlin',
        provider: ['claude', ' codex ', ''],
        model: ['claude-sonnet-5-5'],
        status: 'ok',
        stream: true,
        q: '  ',
      },
      { signal: new AbortController().signal }
    );

    expect(get).toHaveBeenCalledTimes(1);
    const [path, config] = get.mock.calls[0];
    expect(path).toBe(
      '/observability/usage/summary?from_ms=1000&to_ms=2000&tz=Europe%2FBerlin&provider=claude&provider=codex&model=claude-sonnet-5-5&status=ok&stream=true'
    );
    expect(config).toMatchObject({ timeout: 30_000 });
    expect(config?.signal).toBeInstanceOf(AbortSignal);
  });

  test('adds route parameters after the common filters', async () => {
    const get = mockGet({});
    await usageApi.getBreakdown({
      from_ms: 1,
      to_ms: 2,
      group_by: 'model',
      sort: 'cost',
      order: 'asc',
      limit: 5,
    });
    expect(get.mock.calls[0][0]).toBe(
      '/observability/usage/breakdown?from_ms=1&to_ms=2&group_by=model&sort=cost&order=asc&limit=5'
    );
  });

  test('events paging sends the cursor and limit', async () => {
    const get = mockGet({});
    await usageApi.getEvents({ cursor: 'next-page', limit: 50, status: 'failed' });
    expect(get.mock.calls[0][0]).toBe(
      '/observability/usage/events?status=failed&limit=50&cursor=next-page'
    );
  });

  test('one event encodes its id in the path', async () => {
    const get = mockGet({});
    await usageApi.getEvent(981);
    expect(get.mock.calls[0][0]).toBe('/observability/usage/events/981');
    expect(get.mock.calls[0][1]).toMatchObject({ timeout: 30_000 });
  });

  test('price list forwards source and paging', async () => {
    const get = mockGet({});
    await usageApi.listPrices({
      provider: 'claude',
      model: 'sonnet',
      source: 'manual',
      q: 'son',
      limit: 10,
      offset: 5,
    });
    expect(get.mock.calls[0][0]).toBe(
      '/observability/usage/prices?provider=claude&model=sonnet&q=son&limit=10&offset=5&source=manual'
    );
  });

  test('price history names one provider and model', async () => {
    const get = mockGet({});
    await usageApi.getPriceHistory('claude', 'sonnet');
    expect(get.mock.calls[0][0]).toBe(
      '/observability/usage/prices/history?provider=claude&model=sonnet'
    );
  });
});

describe('usage route coverage', () => {
  test('status, timeseries, failures, heatmap, and key labels use their v8 paths', async () => {
    const get = mockGet({});
    await usageApi.getStatus();
    await usageApi.getTimeseries({ group_by: 'model', top: 3 });
    await usageApi.getFailures();
    await usageApi.getHeatmap({ metric: 'cost' });
    await usageApi.getKeyLabels();
    expect(get.mock.calls.map(([path]) => path)).toEqual([
      '/observability/usage/status',
      '/observability/usage/timeseries?group_by=model&top=3',
      '/observability/usage/failures',
      '/observability/usage/heatmap?metric=cost',
      '/observability/usage/key-labels',
    ]);
  });

  test('mutations send the documented bodies', async () => {
    const put = mockPut({ api_key_id: 'abc', label: 'mine', seen: true });
    const post = mockPost({ action: 'prune', duration_ms: 3 });
    const del = mockDelete({ provider: 'claude', model: 'sonnet', restored: true });

    await usageApi.putKeyLabel('abc', 'mine');
    await usageApi.postMaintenance('prune');
    await usageApi.deleteManualPrice('claude', 'sonnet');

    expect(put.mock.calls[0]).toEqual([
      '/observability/usage/key-labels',
      { api_key_id: 'abc', label: 'mine' },
      { timeout: 30_000 },
    ]);
    expect(post.mock.calls[0]).toEqual([
      '/observability/usage/maintenance',
      { action: 'prune' },
      { timeout: 30_000 },
    ]);
    expect(del.mock.calls[0][0]).toBe(
      '/observability/usage/prices/manual?provider=claude&model=sonnet'
    );
  });

  test('price sync, unpriced fill, and correction post to their routes', async () => {
    const post = mockPost({});
    await usageApi.syncPrices();
    await usageApi.priceUnpriced({ provider: 'claude', model: 'sonnet' });
    await usageApi.correctPrices({
      from_ms: 1,
      to_ms: 2,
      confirm: true,
      reason: 'wrong price',
    });
    expect(post.mock.calls.map(([path]) => path)).toEqual([
      '/observability/usage/prices/sync',
      '/observability/usage/prices/price-unpriced',
      '/observability/usage/prices/correct',
    ]);
    expect(post.mock.calls[1][1]).toEqual({ provider: 'claude', model: 'sonnet' });
    expect(post.mock.calls[2][1]).toEqual({
      from_ms: 1,
      to_ms: 2,
      confirm: true,
      reason: 'wrong price',
    });
  });
});

describe('usage disabled state', () => {
  test('a 503 becomes a typed disabled error on every route family', async () => {
    const disabled = Object.assign(new Error('usage database is disabled'), { status: 503 });
    const get = spyOn(apiClient, 'get').mockRejectedValue(disabled);
    const post = spyOn(apiClient, 'post').mockRejectedValue(disabled);
    restore.push(() => get.mockRestore(), () => post.mockRestore());

    for (const request of [
      () => usageApi.getSummary(),
      () => usageApi.getEvents(),
      () => usageApi.getPriceStatus(),
      () => usageApi.syncPrices(),
    ]) {
      await expect(request()).rejects.toBeInstanceOf(UsageDisabledError);
      await expect(request()).rejects.toMatchObject({ status: 503 });
    }
  });

  test('a 400 keeps its original message and is not reported as disabled', async () => {
    const bad = Object.assign(new Error('from_ms must be before to_ms'), { status: 400 });
    spyOn(apiClient, 'get').mockRejectedValue(bad);
    const error = await usageApi.getSummary().catch((cause: unknown) => cause);
    expect(error).toBe(bad);
    expect(isUsageDisabledError(error)).toBe(false);
  });

  test('a disabled export is typed too', async () => {
    const raw = spyOn(apiClient, 'getRaw').mockRejectedValue(
      Object.assign(new Error('usage db disabled'), { status: 503 })
    );
    restore.push(() => raw.mockRestore());
    await expect(usageApi.exportUsage({})).rejects.toBeInstanceOf(UsageDisabledError);
  });
});

describe('usage export', () => {
  test('returns the blob and the server filename', async () => {
    const blob = new Blob(['{}'], { type: 'application/jsonl' });
    const raw = spyOn(apiClient, 'getRaw').mockResolvedValue({
      data: blob,
      headers: { 'content-disposition': 'attachment; filename=usage-1-2.jsonl' },
    } as never);
    restore.push(() => raw.mockRestore());

    const result = await usageApi.exportUsage({
      from_ms: 1,
      to_ms: 2,
      format: 'jsonl',
    });
    expect(raw.mock.calls[0][0]).toBe(
      '/observability/usage/export?from_ms=1&to_ms=2&format=jsonl'
    );
    expect(result.filename).toBe('usage-1-2.jsonl');
    expect(result.blob).toBe(blob);
  });

  test('falls back to a format filename without a disposition', async () => {
    const blob = new Blob(['']);
    const raw = spyOn(apiClient, 'getRaw').mockResolvedValue({ data: blob, headers: {} } as never);
    restore.push(() => raw.mockRestore());
    expect((await usageApi.exportUsage({ format: 'csv' })).filename).toBe('usage.csv');
  });
});
