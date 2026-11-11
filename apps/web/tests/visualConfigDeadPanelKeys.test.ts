import { describe, expect, test } from 'bun:test';
import { parse as parseYaml } from 'yaml';
import { findConfigFieldById } from '@/features/config/searchIndex';
import { DEFAULT_VISUAL_VALUES } from '@/types/visualConfig';
import type { VisualConfigValues } from '@/types/visualConfig';
import { runVisualConfig } from './helpers/visualConfig';

// The fork keeps the released keys `disable-auto-update-panel` and
// `panel-github-repository`: the Go side loads them and ignores them because
// the panel is embedded in the binary. The config page has no editor for
// them, so a save of another field must keep them byte for byte in place.

const DEAD_KEYS = {
  'disable-auto-update-panel': true,
  'panel-github-repository': 'https://example.test/panel',
} as const;

type SectionName = 'remote-management' | 'management';

const yamlWithSection = (section: SectionName) => `server:
  host: 127.0.0.1
  port: 8317
${section}:
  allow-remote: true
  disable-control-panel: false
  disable-auto-update-panel: ${DEAD_KEYS['disable-auto-update-panel']}
  panel-github-repository: ${DEAD_KEYS['panel-github-repository']}
  secret-key: test-key
`;

const savedYaml = (section: SectionName, patch: Partial<VisualConfigValues>) => {
  const yaml = yamlWithSection(section);
  const config = runVisualConfig(yaml, [patch]);
  return parseYaml(config.applyVisualChangesToYaml(yaml)) as Record<string, unknown>;
};

describe('config page has no editor for the dead panel keys', () => {
  test('the keys are absent from the visual values and the field index', () => {
    expect(DEFAULT_VISUAL_VALUES).not.toHaveProperty('rmDisableAutoUpdatePanel');
    expect(DEFAULT_VISUAL_VALUES).not.toHaveProperty('rmPanelRepo');
    expect(findConfigFieldById('rmDisableAutoUpdatePanel')).toBeUndefined();
    expect(findConfigFieldById('rmPanelRepo')).toBeUndefined();
  });
});

describe('dead panel keys survive a save of another field', () => {
  for (const section of ['remote-management', 'management'] as const) {
    test(`unchanged on an unrelated edit in ${section}`, () => {
      const parsed = savedYaml(section, { host: '0.0.0.0' });
      expect(parsed.server).toMatchObject({ host: '0.0.0.0', port: 8317 });
      expect(parsed[section]).toMatchObject(DEAD_KEYS);
    });
  }

  test('unchanged when another field of the same section is edited', () => {
    const parsed = savedYaml('management', { rmAllowRemote: false });
    const section = parsed.management as Record<string, unknown>;
    expect(section['allow-remote']).toBe(false);
    expect(section).toMatchObject(DEAD_KEYS);
  });
});
