import { expect, test } from 'bun:test';
import { openaiToResource } from '../src/features/providers/adapters';
import { normalizeConfigResponse } from '../src/services/api/transformers';

test('keeps backend indexes when normalization filters an unnamed OpenAI group', () => {
  const config = normalizeConfigResponse({
    'api-keys': {
      'openai-compatibility': [
        { 'base-url': 'https://invalid.example.com/v1', keys: [] },
        { name: 'first', 'base-url': 'https://api.example.com/v1', keys: [{ 'api-key': 'first' }] },
        {
          name: 'second',
          'base-url': 'https://gateway.example.com/v1',
          keys: [{ 'api-key': 'second' }],
        },
      ],
    },
  });
  expect(config.openaiCompatibility?.map((item) => item.sourceIndex)).toEqual([1, 2]);
  expect(openaiToResource(config.openaiCompatibility![1], 1).originalIndex).toBe(2);
});
