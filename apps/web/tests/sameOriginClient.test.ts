import { afterEach, describe, expect, test } from 'bun:test';
import type { AxiosAdapter } from 'axios';
import { apiClient } from '../src/services/api/client';

const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window');

const stubWindow = (events: Event[]) => {
  Object.defineProperty(globalThis, 'window', {
    configurable: true,
    value: {
      location: { origin: 'https://proxy.example' },
      dispatchEvent: (event: Event) => {
        events.push(event);
        return true;
      },
    },
  });
};

afterEach(() => {
  apiClient.setConfig({ managementKey: '' });
  if (originalWindow) Object.defineProperty(globalThis, 'window', originalWindow);
  else Reflect.deleteProperty(globalThis, 'window');
});

describe('same-origin v8 management transport', () => {
  test('uses the page origin v8 prefix and bearer authentication', async () => {
    stubWindow([]);
    apiClient.setConfig({ managementKey: 'fixture-only' });
    const adapter: AxiosAdapter = async (config) => {
      expect(config.baseURL).toBe('https://proxy.example/v8/management');
      expect(config.headers.Authorization).toBe('Bearer fixture-only');
      return { data: { status: 'ok' }, status: 200, statusText: 'OK', headers: {}, config };
    };
    await apiClient.get('/config', { adapter });
  });

  test('reports backend version headers without any plugin gate', async () => {
    const events: Event[] = [];
    stubWindow(events);
    apiClient.setConfig({ managementKey: 'fixture-only' });
    const adapter: AxiosAdapter = async (config) => ({
      data: {},
      status: 200,
      statusText: 'OK',
      config,
      headers: { 'x-cpa-version': 'v8.0.3', 'x-cpa-build-date': '2026-11-11' },
    });
    await apiClient.get('/config', { adapter });
    expect(events.map((event) => event.type)).toEqual(['server-version-update']);
    expect((events[0] as CustomEvent).detail).toEqual({
      version: 'v8.0.3',
      buildDate: '2026-11-11',
    });
  });

  test('a 401 response emits the logout event', async () => {
    const events: Event[] = [];
    stubWindow(events);
    apiClient.setConfig({ managementKey: 'stale-key' });
    const adapter: AxiosAdapter = async (config) => {
      throw { isAxiosError: true, message: 'Unauthorized', config, response: { status: 401 } };
    };
    await expect(apiClient.get('/config', { adapter })).rejects.toThrow();
    expect(events.map((event) => event.type)).toEqual(['unauthorized']);
  });
});

test('a captured connection keeps its credentials and suppresses stale events', async () => {
  const events: Event[] = [];
  stubWindow(events);
  apiClient.setConfig({ managementKey: 'original-fixture' });
  const captured = apiClient.captureConnection();
  apiClient.setConfig({ managementKey: 'new-fixture' });
  const adapter: AxiosAdapter = async (config) => {
    expect(config.baseURL).toBe('https://proxy.example/v8/management');
    expect(config.headers.Authorization).toBe('Bearer original-fixture');
    return {
      data: {},
      status: 200,
      statusText: 'OK',
      config,
      headers: { 'x-cpa-version': 'old' },
    };
  };
  await captured.get('/config', { adapter });
  expect(events).toEqual([]);
});
