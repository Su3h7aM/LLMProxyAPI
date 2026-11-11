import { afterEach, describe, expect, test } from 'bun:test';
import i18n from '@/i18n';
import { CLAUDE_CONFIG } from '@/features/quota/providers/claude/data';
import { apiCallApi, type ApiCallRequest } from '@/services/api';
import {
  CLAUDE_PROFILE_URL,
  CLAUDE_REQUEST_HEADERS,
  CLAUDE_USAGE_URL,
} from '@/utils/quota/constants';

const originalRequest = apiCallApi.request;

afterEach(() => {
  apiCallApi.request = originalRequest;
});

describe('Claude request identity', () => {
  test('sends a claude-cli User-Agent: the api-call backend adds none', () => {
    expect(CLAUDE_REQUEST_HEADERS['User-Agent']).toBe('claude-cli/2.1.280 (external, cli)');
    expect(CLAUDE_REQUEST_HEADERS['anthropic-beta']).toBe('oauth-2025-04-20');
  });

  test('sends the User-Agent on usage and profile requests', async () => {
    const requests: ApiCallRequest[] = [];
    apiCallApi.request = async (request) => {
      requests.push(request);
      return { statusCode: 200, header: {}, body: {}, bodyText: '{}' };
    };

    await expect(
      CLAUDE_CONFIG.fetchQuota({ name: 'claude.json', authIndex: 'claude:fixture' }, i18n.t)
    ).rejects.toThrow();

    expect(requests.map((request) => request.url).sort()).toEqual(
      [CLAUDE_PROFILE_URL, CLAUDE_USAGE_URL].sort()
    );
    for (const request of requests) {
      expect(request.header?.['User-Agent']).toBe('claude-cli/2.1.280 (external, cli)');
    }
  });
});
