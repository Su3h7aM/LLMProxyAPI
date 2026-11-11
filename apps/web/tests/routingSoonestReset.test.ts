import { describe, expect, test } from 'bun:test';
import {
  predictSoonestReset,
  type SoonestResetCredentialInput,
} from '@/features/routing/soonestReset';
import type { ClaudeQuotaState } from '@/types';
import { HOUR_MS } from '@/utils/time/durations';

const NOW = 1_786_000_000_000;

const claude = (
  windows: Array<{
    id: string;
    usedPercent: number | null;
    resetAtMs?: number | null;
    periodHours?: number | null;
  }>
): ClaudeQuotaState => ({
  status: 'success',
  windows: windows.map((window) => ({
    id: window.id,
    label: window.id,
    usedPercent: window.usedPercent,
    resetLabel: '-',
    resetAtMs: window.resetAtMs ?? null,
    periodHours: window.periodHours ?? null,
  })),
});

const input = (
  key: string,
  snapshots: ClaudeQuotaState | undefined | null
): SoonestResetCredentialInput => ({
  key,
  name: key,
  snapshots:
    snapshots === null || snapshots === undefined
      ? snapshots === undefined
        ? [{ provider: 'claude', quota: undefined }]
        : []
      : [{ provider: 'claude', quota: snapshots }],
});

const tiersOf = (...tiers: Array<{ id: string; memberKeys: string[] }>) => tiers;

describe('predictSoonestReset', () => {
  test('ranks a known reset before no data and exhausted last', () => {
    const prediction = predictSoonestReset(
      [
        input('cred:b', undefined),
        input(
          'cred:c',
          claude([{ id: 'five-hour', usedPercent: 100, resetAtMs: NOW + HOUR_MS, periodHours: 5 }])
        ),
        input(
          'cred:a',
          claude([{ id: 'five-hour', usedPercent: 20, resetAtMs: NOW + HOUR_MS, periodHours: 5 }])
        ),
      ],
      tiersOf({ id: 't0', memberKeys: ['cred:a', 'cred:b', 'cred:c'] }),
      NOW
    );
    expect(prediction.ranks.map((rank) => rank.key)).toEqual(['cred:a', 'cred:b', 'cred:c']);
    expect(prediction.ranks.map((rank) => rank.class)).toEqual([
      'known',
      'unknown',
      'exhausted',
    ]);
    expect(prediction.ranks.map((rank) => rank.rank)).toEqual([0, 1, 2]);
    expect(prediction.nextByTier.get('t0')).toBe('cred:a');
    expect(prediction.nextKey).toBe('cred:a');
  });

  test('ignores elapsed resets and credentials without snapshots', () => {
    const prediction = predictSoonestReset(
      [
        input(
          'cred:stale',
          claude([{ id: 'five-hour', usedPercent: 20, resetAtMs: NOW - 1, periodHours: 5 }])
        ),
        { key: 'cred:empty', name: 'empty', snapshots: [] },
      ],
      tiersOf({ id: 't0', memberKeys: ['cred:stale', 'cred:empty'] }),
      NOW
    );
    expect(prediction.ranks.every((rank) => rank.class === 'unknown')).toBe(true);
    expect(prediction.ranks.every((rank) => rank.resetAtMs === null)).toBe(true);
    // Tie on class and reset breaks on key.
    expect(prediction.ranks.map((rank) => rank.key)).toEqual(['cred:empty', 'cred:stale']);
  });

  test('an exhausted window outranks a known reset on the same credential', () => {
    const prediction = predictSoonestReset(
      [
        input(
          'cred:mixed',
          claude([
            { id: 'five-hour', usedPercent: 10, resetAtMs: NOW + HOUR_MS, periodHours: 5 },
            { id: 'seven-day', usedPercent: 100, resetAtMs: NOW + 6 * 24 * HOUR_MS, periodHours: 168 },
          ])
        ),
      ],
      tiersOf({ id: 't0', memberKeys: ['cred:mixed'] }),
      NOW
    );
    expect(prediction.ranks[0].class).toBe('exhausted');
    expect(prediction.ranks[0].resetAtMs).toBeNull();
  });

  test('the longest window decides the credential reset time', () => {
    const soon = NOW + HOUR_MS;
    const late = NOW + 6 * 24 * HOUR_MS;
    const prediction = predictSoonestReset(
      [
        input(
          'cred:long',
          claude([
            { id: 'five-hour', usedPercent: 10, resetAtMs: soon, periodHours: 5 },
            { id: 'seven-day', usedPercent: 40, resetAtMs: late, periodHours: 168 },
          ])
        ),
        input(
          'cred:short',
          claude([{ id: 'five-hour', usedPercent: 10, resetAtMs: soon, periodHours: 5 }])
        ),
      ],
      tiersOf({ id: 't0', memberKeys: ['cred:long', 'cred:short'] }),
      NOW
    );
    expect(prediction.byKey.get('cred:long')?.resetAtMs).toBe(late);
    expect(prediction.byKey.get('cred:short')?.resetAtMs).toBe(soon);
    expect(prediction.ranks.map((rank) => rank.key)).toEqual(['cred:short', 'cred:long']);
  });

  test('equal resets break ties on credential key', () => {
    const at = NOW + HOUR_MS;
    const prediction = predictSoonestReset(
      [
        input(
          'cred:z',
          claude([{ id: 'five-hour', usedPercent: 10, resetAtMs: at, periodHours: 5 }])
        ),
        input(
          'cred:a',
          claude([{ id: 'five-hour', usedPercent: 90, resetAtMs: at, periodHours: 5 }])
        ),
      ],
      tiersOf({ id: 't0', memberKeys: ['cred:z', 'cred:a'] }),
      NOW
    );
    expect(prediction.ranks.map((rank) => rank.key)).toEqual(['cred:a', 'cred:z']);
  });

  test('predicts the next credential per tier and overall', () => {
    const prediction = predictSoonestReset(
      [
        input(
          'cred:slow',
          claude([{ id: 'five-hour', usedPercent: 10, resetAtMs: NOW + 3 * HOUR_MS, periodHours: 5 }])
        ),
        input(
          'cred:fast',
          claude([{ id: 'five-hour', usedPercent: 10, resetAtMs: NOW + HOUR_MS, periodHours: 5 }])
        ),
        input('cred:none', undefined),
      ],
      tiersOf(
        { id: 't0', memberKeys: ['cred:slow'] },
        { id: 't1', memberKeys: ['cred:fast', 'cred:none'] }
      ),
      NOW
    );
    expect(prediction.nextByTier.get('t0')).toBe('cred:slow');
    expect(prediction.nextByTier.get('t1')).toBe('cred:fast');
    // Overall next is the first tier in order that holds a credential.
    expect(prediction.nextKey).toBe('cred:slow');
  });

  test('tiers without credentials predict nothing', () => {
    const prediction = predictSoonestReset(
      [input('cred:a', undefined)],
      tiersOf({ id: 't0', memberKeys: ['api-key-1'] }, { id: 't1', memberKeys: [] }),
      NOW
    );
    expect(prediction.nextByTier.get('t0')).toBeNull();
    expect(prediction.nextByTier.get('t1')).toBeNull();
    expect(prediction.nextKey).toBeNull();
    // Unranked tiers never move the credential itself.
    expect(prediction.byKey.get('cred:a')?.rank).toBe(0);
  });
});
