import { afterEach, describe, expect, spyOn, test } from 'bun:test';
import { apiClient } from '@/services/api/client';
import { oauthApi } from '@/services/api/oauth';
import { vertexApi } from '@/services/api/vertex';

const spies: Array<{ mockRestore(): void }> = [];
afterEach(() => spies.splice(0).forEach((spy) => spy.mockRestore()));

const mock = (method: 'get' | 'post' | 'postForm', value: unknown = {}) => {
  const spy = spyOn(apiClient, method).mockResolvedValue(value as never);
  spies.push(spy);
  return spy;
};

describe('OAuth v8 management contract', () => {
  test('maps UI anthropic to claude and plugins use the shared login route', async () => {
    const get = mock('get');
    const post = mock('post');
    await oauthApi.startAuth('anthropic');
    expect(get).toHaveBeenLastCalledWith('/oauth/auth-url', {
      params: { provider: 'claude', is_webui: true },
    });
    await oauthApi.startAuth('example-plugin');
    expect(get).toHaveBeenLastCalledWith('/oauth/auth-url', {
      params: { provider: 'example-plugin' },
    });
    await oauthApi.submitCallback('anthropic', 'https://example.test/callback?code=test');
    expect(post).toHaveBeenLastCalledWith(
      '/oauth/callback',
      { provider: 'claude', redirect_url: 'https://example.test/callback?code=test' },
      undefined
    );
  });

  test('rejects a provider name that is not a management path key', () => {
    expect(() => oauthApi.startAuth('bad provider')).toThrow('Invalid OAuth provider');
  });

  test('imports a Vertex service account through the OAuth import route', async () => {
    const form = mock('postForm', { status: 'ok' });
    await vertexApi.importCredential(new File(['{}'], 'fixture.json'), 'europe-west4');
    expect(form.mock.calls.at(-1)?.[0]).toBe('/oauth/import?provider=vertex');
    const body = form.mock.calls.at(-1)?.[1] as FormData;
    expect(body.get('file')).toBeInstanceOf(File);
    expect(body.get('location')).toBe('europe-west4');
  });
});
