import { describe, expect, test } from 'bun:test';
import { buildProviderGroups } from '../src/features/providers/useProviderWorkbench';
import type { Config, OpenAIProviderConfig, ProviderKeyConfig } from '../src/types';

/**
 * Ports the rule of fork commit 4d1b8e47: endpoints whose base URL matches a
 * removed sponsor brand stay in the generic provider groups instead of
 * vanishing from the workbench.
 */
describe('provider groups keep sponsor-matching endpoints', () => {
  const key = (baseUrl: string): ProviderKeyConfig => ({ apiKey: 'test-key', baseUrl });
  const openai = (name: string, baseUrl: string): OpenAIProviderConfig => ({
    name,
    baseUrl,
    apiKeyEntries: [{ apiKey: 'test-key' }],
  });

  test.each([
    ['gemini', 'geminiApiKeys', ['https://api.qnaigc.com', 'https://api.moonshot.cn']],
    [
      'codex',
      'codexApiKeys',
      [
        'https://api.apikey.fan/v1',
        'https://api.fenno.ai/v1',
        'https://api.qnaigc.com/v1',
        'https://api.moonshot.ai/v1',
      ],
    ],
    [
      'claude',
      'claudeApiKeys',
      [
        'https://api.apikey.fan',
        'https://api.fenno.ai',
        'https://api.qnaigc.com',
        'https://api.moonshot.ai/anthropic',
      ],
    ],
  ] as const)('%s keeps configured service endpoints in the generic group', (brand, field, urls) => {
    const config = { [field]: urls.map((baseUrl) => key(baseUrl)) } as Config;
    const resources = buildProviderGroups(config).find((group) => group.id === brand)!.resources;
    expect(resources).toHaveLength(urls.length);
    for (let index = 0; index < resources.length; index++) {
      expect(resources[index].brand).toBe(brand);
      expect(resources[index].baseUrl).toBe(urls[index]);
    }
  });

  test('openaiCompatibility keeps sponsor-matching endpoints in the generic group', () => {
    const urls = [
      'https://api.apikey.fan/v1',
      'https://api.qnaigc.com/v1',
      'https://api.moonshot.ai/v1',
    ];
    const config: Config = {
      openaiCompatibility: urls.map((baseUrl, index) => openai(`service-${index}`, baseUrl)),
    };
    const resources = buildProviderGroups(config).find(
      (group) => group.id === 'openaiCompatibility'
    )!.resources;
    expect(resources).toHaveLength(urls.length);
    for (let index = 0; index < resources.length; index++) {
      expect(resources[index].brand).toBe('openaiCompatibility');
      expect(resources[index].baseUrl).toBe(urls[index]);
    }
  });

  test('has no sponsor brand groups', () => {
    const ids = buildProviderGroups({}).map((group) => group.id);
    for (const brand of ['apikeyFun', 'fennoAI', 'qiniuCloud', 'kimi']) {
      expect(ids).not.toContain(brand);
    }
  });
});
