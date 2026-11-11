import { describe, expect, test } from 'bun:test';
import { normalizeConfigResponse } from '../src/services/api/transformers';

describe('v8 persisted configuration normalization', () => {
  test('uses backend AI Studio authentication default when the persisted field is absent', () => {
    expect(normalizeConfigResponse({ 'config-version': 8 }).wsAuth).toBe(true);
    expect(
      normalizeConfigResponse({ oauth: { providers: { aistudio: { 'ws-auth': false } } } }).wsAuth
    ).toBe(false);
  });

  test('reads only Antigravity credits from the OAuth provider path', () => {
    expect(normalizeConfigResponse({}).quotaExceeded).toEqual({ antigravityCredits: false });
    for (const enabled of [true, false]) {
      const oauth = { providers: { antigravity: { 'antigravity-credits': enabled } } };
      expect(normalizeConfigResponse({ oauth }).quotaExceeded).toEqual({
        switchProject: undefined,
        switchPreviewModel: undefined,
        antigravityCredits: enabled,
      });
      const config = normalizeConfigResponse({
        oauth,
        'quota-exceeded': {
          'switch-project': true,
          'switch-preview-model': true,
          'antigravity-credits': !enabled,
        },
      });
      // The Go runtime still reads switch-project and switch-preview-model from
      // the top-level quota-exceeded block; only antigravity-credits moved to
      // the OAuth provider tree.
      expect(config.quotaExceeded).toEqual({
        switchProject: true,
        switchPreviewModel: true,
        antigravityCredits: enabled,
      });
    }
    expect(
      normalizeConfigResponse({ 'quota-exceeded': { 'antigravity-credits': true } }).quotaExceeded
    ).toEqual({
      switchProject: undefined,
      switchPreviewModel: undefined,
      antigravityCredits: false,
    });
  });

  test('does not interpret legacy root settings as v8 values', () => {
    const config = normalizeConfigResponse({
      'api-keys': ['legacy-client'],
      'codex-api-key': [{ 'api-key': 'legacy-upstream' }],
      'proxy-url': 'https://old.invalid',
      'ws-auth': false,
      requests: { 'proxy-url': 'https://current.invalid' },
    });
    expect(config.apiKeys).toEqual([]);
    expect(config.codexApiKeys).toEqual([]);
    expect(config.proxyUrl).toBe('https://current.invalid');
    expect(config.wsAuth).toBe(true);
  });

  test('flattens v8 provider groups into per-key models', () => {
    const config = normalizeConfigResponse({
      'api-keys': {
        codex: [
          {
            name: 'team',
            'base-url': 'https://codex.example',
            keys: [{ 'api-key': 'secret-1', weight: 3 }, { 'api-key': 'secret-2' }],
          },
        ],
      },
    });
    expect(config.codexApiKeys).toEqual([
      {
        apiKey: 'secret-1',
        baseUrl: 'https://codex.example',
        weight: 3,
      },
      { apiKey: 'secret-2', baseUrl: 'https://codex.example' },
    ]);
  });
});
