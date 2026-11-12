import { afterAll, beforeEach, describe, expect, test } from 'bun:test';
import {
  DEFAULT_USAGE_FILTERS,
  readUsageFilters,
  sanitizeUsageFilters,
  useUsageFilterStore,
  USAGE_FILTER_STORAGE_KEY,
  writeUsageFilters,
} from '@/stores/useUsageFilterStore';

const originalWindow = (globalThis as { window?: unknown }).window;

function installSessionStorage() {
  const store = new Map<string, string>();
  const storage = {
    getItem: (key: string) => store.get(key) ?? null,
    setItem: (key: string, value: string) => void store.set(key, value),
    removeItem: (key: string) => void store.delete(key),
    clear: () => store.clear(),
    key: (index: number) => [...store.keys()][index] ?? null,
    get length() {
      return store.size;
    },
  };
  Object.defineProperty(globalThis, 'window', {
    value: { sessionStorage: storage },
    configurable: true,
    writable: true,
  });
  return storage;
}

let storage: ReturnType<typeof installSessionStorage>;

beforeEach(() => {
  storage = installSessionStorage();
  useUsageFilterStore.getState().reset();
  storage.clear();
});

afterAll(() => {
  if (originalWindow === undefined) {
    delete (globalThis as { window?: unknown }).window;
  } else {
    (globalThis as { window?: unknown }).window = originalWindow;
  }
});

describe('usage filter sanitizing', () => {
  test('keeps valid values and removes duplicates, blanks, and unknown choices', () => {
    expect(
      sanitizeUsageFilters({
        rangePreset: '7d',
        customFromMs: 10,
        customToMs: 'no',
        provider: [' claude ', 'claude', ''],
        model: [1, 'sonnet'],
        credential: null,
        apiKey: ['abc'],
        status: 'failed',
        stream: 'sometimes',
      })
    ).toEqual({
      rangePreset: '7d',
      customFromMs: 10,
      customToMs: null,
      provider: ['claude'],
      model: ['sonnet'],
      credential: [],
      apiKey: ['abc'],
      status: 'failed',
      stream: 'all',
    });
  });

  test('falls back to the defaults for a non-object payload', () => {
    expect(sanitizeUsageFilters(null)).toEqual(DEFAULT_USAGE_FILTERS);
    expect(sanitizeUsageFilters('x')).toEqual(DEFAULT_USAGE_FILTERS);
  });
});

describe('session storage persistence', () => {
  test('round-trips and tolerates malformed or missing values', () => {
    const values = {
      ...DEFAULT_USAGE_FILTERS,
      rangePreset: '30d' as const,
      provider: ['codex'],
      status: 'ok' as const,
    };
    writeUsageFilters(values);
    expect(readUsageFilters()).toEqual(values);

    storage.setItem(USAGE_FILTER_STORAGE_KEY, '{not json');
    expect(readUsageFilters()).toBeNull();
    storage.setItem(USAGE_FILTER_STORAGE_KEY, '[]');
    expect(readUsageFilters()).toBeNull();
  });

  test('store actions write only the session key', () => {
    useUsageFilterStore.getState().setRangePreset('90d');
    useUsageFilterStore.getState().setStatus('failed');
    useUsageFilterStore.getState().setStream('true');
    useUsageFilterStore.getState().setMultiFilter('credential', ['auth-1', 'auth-1']);
    useUsageFilterStore.getState().setCustomRange(100, 200);

    expect(useUsageFilterStore.getState()).toMatchObject({
      rangePreset: '90d',
      status: 'failed',
      stream: 'true',
      credential: ['auth-1'],
      customFromMs: 100,
      customToMs: 200,
    });
    expect(readUsageFilters()).toMatchObject({
      rangePreset: '90d',
      status: 'failed',
      stream: 'true',
      credential: ['auth-1'],
    });
    expect(storage.getItem(USAGE_FILTER_STORAGE_KEY)).not.toBeNull();
  });

  test('reset restores the defaults in memory and in storage', () => {
    useUsageFilterStore.getState().setRangePreset('1h');
    useUsageFilterStore.getState().reset();
    expect(useUsageFilterStore.getState()).toMatchObject(DEFAULT_USAGE_FILTERS);
    expect(readUsageFilters()).toEqual(DEFAULT_USAGE_FILTERS);
  });

  test('a storage without sessionStorage does not break the store', () => {
    Object.defineProperty(globalThis, 'window', {
      value: {},
      configurable: true,
      writable: true,
    });
    useUsageFilterStore.getState().setStatus('ok');
    expect(useUsageFilterStore.getState().status).toBe('ok');
  });
});
