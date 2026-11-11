import { beforeAll, describe, expect, test } from 'bun:test';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { MemoryRouter } from 'react-router-dom';
import i18n from '@/i18n';
import { SoonestResetSection } from '@/features/routing/components/SoonestResetSection';
import type { SoonestResetCredentialRank } from '@/features/routing/soonestReset';
import { formatInstantShort } from '@/utils/quota';
import { HOUR_MS } from '@/utils/time/durations';

const NOW = 1_786_000_000_000;
const SOON = NOW + 2 * HOUR_MS;

const rank = (overrides: Partial<SoonestResetCredentialRank> & { key: string }): SoonestResetCredentialRank => ({
  name: overrides.key,
  class: 'unknown',
  resetAtMs: null,
  rank: 0,
  ...overrides,
});

const render = (ranks: SoonestResetCredentialRank[], nextKey: string | null) =>
  renderToStaticMarkup(
    createElement(
      MemoryRouter,
      null,
      createElement(SoonestResetSection, { data: { ranks, nextKey }, nowMs: NOW })
    )
  );

beforeAll(async () => {
  await i18n.changeLanguage('en');
});

describe('SoonestResetSection', () => {
  test('names the next credential with relative time and absolute title', () => {
    const markup = render(
      [
        rank({ key: 'cred:slow', name: 'slow', class: 'known', resetAtMs: SOON + HOUR_MS, rank: 1 }),
        rank({ key: 'cred:fast', name: 'fast', class: 'known', resetAtMs: SOON, rank: 0 }),
      ],
      'cred:fast'
    );
    expect(markup).toContain('Soonest reset');
    expect(markup).toContain('Next credential');
    expect(markup).toContain('fast');
    expect(markup).toContain(`title="${formatInstantShort(SOON)}"`);
    expect(markup).toContain('in 2 hours');
  });

  test('marks exhausted and unknown credentials without a fetch hint when data exists', () => {
    const markup = render(
      [
        rank({ key: 'cred:fast', name: 'fast', class: 'known', resetAtMs: SOON, rank: 0 }),
        rank({ key: 'cred:gone', name: 'gone', class: 'exhausted', rank: 2 }),
        rank({ key: 'cred:new', name: 'new', class: 'unknown', rank: 1 }),
      ],
      'cred:fast'
    );
    expect(markup).toContain('Quota exhausted');
    expect(markup).toContain('No reset time yet');
    expect(markup).not.toContain('/quota');
  });

  test('points at the Quota page when no snapshot is cached', () => {
    const markup = render(
      [rank({ key: 'cred:new', name: 'new', class: 'unknown', rank: 0 })],
      'cred:new'
    );
    expect(markup).toContain('Open the Quota page to load reset times.');
    expect(markup).toContain('href="/quota"');
    expect(markup).toContain('Quota Management');
  });

  test('renders nothing without credential members', () => {
    expect(render([], null)).toBe('');
  });
});
