import { afterEach, describe, expect, test } from 'bun:test';
import {
  buildCodexResponsesEndpoint,
  buildOpenAIChatCompletionsEndpoint,
} from '../src/components/providers/utils';
import { apiCallApi } from '../src/services/api/apiCall';
import { modelsApi } from '../src/services/api/models';

const originalApiCallRequest = apiCallApi.request;
afterEach(() => {
  apiCallApi.request = originalApiCallRequest;
});

describe('Kimi compatible endpoints', () => {
  test('builds standard OpenAI and Codex endpoints', () => {
    expect(buildOpenAIChatCompletionsEndpoint('https://api.moonshot.ai/v1')).toBe(
      'https://api.moonshot.ai/v1/chat/completions'
    );
    expect(buildCodexResponsesEndpoint('https://api.moonshot.ai/v1')).toBe(
      'https://api.moonshot.ai/v1/responses'
    );
  });
  test('discovers models through the versioned OpenAI endpoint', async () => {
    let requestedUrl = '';
    apiCallApi.request = (async (payload) => {
      requestedUrl = payload.url;
      return { statusCode: 200, header: {}, bodyText: '', body: { data: [] } };
    }) as typeof apiCallApi.request;
    await modelsApi.fetchModelsViaApiCall('https://api.moonshot.ai/v1', 'test-key');
    expect(requestedUrl).toBe('https://api.moonshot.ai/v1/models');
  });
});
