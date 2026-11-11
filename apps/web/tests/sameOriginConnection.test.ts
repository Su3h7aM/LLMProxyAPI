import { afterEach, describe, expect, test } from 'bun:test';
import { getManagementApiUrl, getSameOriginApiBase } from '../src/utils/connection';

const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window');

const stubOrigin = (origin: string | undefined) => {
  Object.defineProperty(globalThis, 'window', {
    configurable: true,
    value: origin === undefined ? {} : { location: { origin } },
  });
};

afterEach(() => {
  if (originalWindow) Object.defineProperty(globalThis, 'window', originalWindow);
  else Reflect.deleteProperty(globalThis, 'window');
});

describe('same-origin management connection', () => {
  test('uses the page origin plus the v8 prefix', () => {
    stubOrigin('https://proxy.example');
    expect(getSameOriginApiBase()).toBe('https://proxy.example');
    expect(getManagementApiUrl()).toBe('https://proxy.example/v8/management');
  });

  test('trims a trailing slash from the origin', () => {
    stubOrigin('https://proxy.example/');
    expect(getManagementApiUrl()).toBe('https://proxy.example/v8/management');
  });

  test('falls back to a relative v8 prefix without a window', () => {
    stubOrigin(undefined);
    expect(getSameOriginApiBase()).toBe('');
    expect(getManagementApiUrl()).toBe('/v8/management');
  });
});
