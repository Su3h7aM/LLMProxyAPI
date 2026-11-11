// OpenCode Go keys are stored in the v8 `api-keys.opencode-go` groups. The
// backend maps them to the legacy `opencode-go-api-key` list. The panel must
// read that group back into `openCodeGoApiKeys`, or the counter stays at zero.

import { describe, expect, test } from 'bun:test';
import { normalizeConfigResponse } from '../src/services/api/transformers';

describe('OpenCode Go provider keys in config', () => {
  test('api-keys.opencode-go groups populate openCodeGoApiKeys', () => {
    const config = normalizeConfigResponse({
      'api-keys': {
        'opencode-go': [{ name: 'opencode-go-1', keys: [{ 'api-key': 'go-key-1', priority: 2 }] }],
      },
    });

    expect(config.openCodeGoApiKeys?.length).toBe(1);
    expect(config.openCodeGoApiKeys?.[0]).toMatchObject({
      apiKey: 'go-key-1',
      priority: 2,
    });
  });

  test('missing api-keys.opencode-go yields an empty list', () => {
    const config = normalizeConfigResponse({});
    expect(config.openCodeGoApiKeys ?? []).toEqual([]);
  });
});
