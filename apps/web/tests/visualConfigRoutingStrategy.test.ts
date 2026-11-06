import { describe, expect, test } from 'bun:test';
import { createElement, useState } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { parse as parseYaml } from 'yaml';
import { readFileSync } from 'node:fs';
import { normalizeRoutingStrategy } from '../src/services/api/routing';
import { ROUTING_STRATEGIES } from '../src/types/routing';
import { parseRoutingStrategy, useVisualConfig } from '../src/hooks/useVisualConfig';

describe('visual config weighted routing strategy', () => {
  test('recognizes the weighted-round-robin backend value', () => {
    expect(parseRoutingStrategy('weighted-round-robin')).toBe('weighted-round-robin');
    expect(parseRoutingStrategy('weightedroundrobin')).toBe('weighted-round-robin');
    expect(parseRoutingStrategy('wrr')).toBe('weighted-round-robin');
    expect(parseRoutingStrategy('fill-first')).toBe('fill-first');
    expect(parseRoutingStrategy('fillfirst')).toBe('fill-first');
    expect(parseRoutingStrategy('ff')).toBe('fill-first');
    expect(parseRoutingStrategy(undefined)).toBe('round-robin');
  });

  test('writes weighted-round-robin without coercing it to round-robin', () => {
    function Harness() {
      const visualConfig = useVisualConfig();
      const [phase, setPhase] = useState(0);

      if (phase === 0) {
        visualConfig.setVisualValues({ routingStrategy: 'weighted-round-robin' });
        setPhase(1);
      } else {
        return createElement(
          'pre',
          null,
          visualConfig.applyVisualChangesToYaml('routing:\n  strategy: round-robin\n')
        );
      }

      return null;
    }

    const markup = renderToStaticMarkup(createElement(Harness));
    const result = markup.slice('<pre>'.length, -'</pre>'.length);

    expect(parseYaml(result)).toEqual({ routing: { strategy: 'weighted-round-robin' } });
  });

  test('recognizes the soonest-reset backend value and its alias', () => {
    expect(parseRoutingStrategy('soonest-reset')).toBe('soonest-reset');
    expect(parseRoutingStrategy(' Soonest-Reset ')).toBe('soonest-reset');
    expect(parseRoutingStrategy('soonestreset')).toBe('soonest-reset');
    expect(normalizeRoutingStrategy('soonest-reset')).toBe('soonest-reset');
    expect(normalizeRoutingStrategy('soonest_reset')).toBe('soonest-reset');
    expect(normalizeRoutingStrategy('soonestreset')).toBe('soonest-reset');
  });

  test('writes soonest-reset without coercing it to round-robin', () => {
    function Harness() {
      const visualConfig = useVisualConfig();
      const [phase, setPhase] = useState(0);

      if (phase === 0) {
        visualConfig.setVisualValues({ routingStrategy: 'soonest-reset' });
        setPhase(1);
      } else {
        return createElement(
          'pre',
          null,
          visualConfig.applyVisualChangesToYaml('routing:\n  strategy: round-robin\n')
        );
      }

      return null;
    }

    const markup = renderToStaticMarkup(createElement(Harness));
    const result = markup.slice('<pre>'.length, -'</pre>'.length);

    expect(parseYaml(result)).toEqual({ routing: { strategy: 'soonest-reset' } });
  });

  test('lists soonest-reset and translates its labels in every locale', () => {
    expect(ROUTING_STRATEGIES).toContain('soonest-reset');
    for (const locale of ['en', 'zh-CN', 'zh-TW', 'ru', 'vi', 'ko', 'tr']) {
      const messages = JSON.parse(
        readFileSync(new URL(`../src/i18n/locales/${locale}.json`, import.meta.url), 'utf8')
      );
      expect(typeof messages.basic_settings.routing_strategy_soonest_reset).toBe('string');
      expect(typeof messages.config_management.visual.sections.network.strategy_soonest_reset).toBe(
        'string'
      );
      expect(typeof messages.routing.strategy_soonest_reset).toBe('string');
      expect(typeof messages.routing.strategy_soonest_reset_desc).toBe('string');
    }
  });
});
