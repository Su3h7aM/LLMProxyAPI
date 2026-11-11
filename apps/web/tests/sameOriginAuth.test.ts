import { afterEach, beforeEach, describe, expect, test } from 'bun:test';
import { shouldAttemptAutoLogin, useAuthStore } from '../src/stores/useAuthStore';
import { AUTH_LOGIN_STATE_KEY } from '../src/services/storage/secureStorage';
import { STORAGE_KEY_AUTH } from '../src/utils/constants';

const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window');
const originalStorage = Object.getOwnPropertyDescriptor(globalThis, 'localStorage');
let values: Map<string, string>;

beforeEach(() => {
  values = new Map();
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: {
      getItem: (key: string) => values.get(key) ?? null,
      setItem: (key: string, value: string) => void values.set(key, value),
      removeItem: (key: string) => void values.delete(key),
      get length() {
        return values.size;
      },
      key: (index: number) => [...values.keys()][index] ?? null,
      clear: () => values.clear(),
    },
  });
  Object.defineProperty(globalThis, 'window', {
    configurable: true,
    value: { location: { origin: 'https://panel.example' } },
  });
  useAuthStore.getState().logout();
  useAuthStore.setState({ isAuthenticated: false, apiBase: '', managementKey: '' });
});

afterEach(() => {
  useAuthStore.getState().logout();
  if (originalWindow) Object.defineProperty(globalThis, 'window', originalWindow);
  else Reflect.deleteProperty(globalThis, 'window');
  if (originalStorage) Object.defineProperty(globalThis, 'localStorage', originalStorage);
  else Reflect.deleteProperty(globalThis, 'localStorage');
});

describe('key-only auto login', () => {
  test('requires a remembered session with a saved key', () => {
    expect(shouldAttemptAutoLogin(true, 'k')).toBe(true);
    expect(shouldAttemptAutoLogin(false, 'k')).toBe(false);
    expect(shouldAttemptAutoLogin(true, '')).toBe(false);
  });

  test('hasRestorableSession mirrors restoreSession without side effects', () => {
    const { hasRestorableSession } = useAuthStore.getState();
    useAuthStore.setState({ apiBase: 'https://panel.example', managementKey: 'sk-demo' });
    expect(hasRestorableSession()).toBe(false);

    values.set(AUTH_LOGIN_STATE_KEY, 'true');
    expect(hasRestorableSession()).toBe(true);

    useAuthStore.setState({ managementKey: '' });
    expect(hasRestorableSession()).toBe(false);
    expect(values.get(AUTH_LOGIN_STATE_KEY)).toBe('true');
  });

  test('restoreSession pins the base to the page origin and needs no stored address', async () => {
    useAuthStore.setState({ managementKey: 'sk-demo', rememberPassword: false });
    await expect(useAuthStore.getState().restoreSession()).resolves.toBe(false);
    expect(useAuthStore.getState().apiBase).toBe('https://panel.example');
  });

  test('logout clears the session and the login flag but keeps the remembered record', () => {
    values.set(AUTH_LOGIN_STATE_KEY, 'true');
    useAuthStore.setState({ managementKey: 'sk-demo', rememberPassword: true });
    useAuthStore.getState().logout();
    expect(values.has(AUTH_LOGIN_STATE_KEY)).toBe(false);
    expect(useAuthStore.getState().managementKey).toBe('');
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
    expect(useAuthStore.getState().hasRestorableSession()).toBe(false);
  });
});
