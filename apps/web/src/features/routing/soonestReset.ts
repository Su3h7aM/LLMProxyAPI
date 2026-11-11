/**
 * Soonest-reset prediction for the routing page.
 *
 * Pure and clock-free (`nowMs` is always passed in). It mirrors the backend
 * rule in `sdk/cliproxy/auth/selector_soonest_reset.go` from cached quota
 * snapshots only: no fetch, no fan-out.
 *
 * Backend rule, per credential:
 *  1. Skip windows whose reset already passed (their snapshot is stale).
 *  2. Any remaining exhausted window puts the credential last.
 *  3. Otherwise the longest window decides the credential reset time.
 *  4. Rank: known reset first, then no data, then exhausted; ties break on
 *     reset time, then on credential key.
 *
 * Panel mapping: a quota observation window is exhausted when it reports
 * zero percent remaining (`remainingPercent <= 0`); the backend reads
 * `Used-Percent >= 100` (Codex) or a rejected status (Claude). A rejected
 * Claude window fails the quota read, so the panel sees it as no data.
 */

import type { AuthFileItem } from '@/types';
import { QUOTA_PROVIDER_DATA } from '../quota/providers/registry';
import { getQuotaObservations } from '../quota/providers/registry';
import type {
  QuotaCardState,
  QuotaProviderType,
  QuotaStore,
} from '../quota/providers/types';
import { QUOTA_TAB_ORDER } from '../quota/constants';
import { getQuotaCacheFileName, getQuotaCacheKey } from '@/utils/quota/identity';
import { resolveQuotaProviderType } from '../quota/logic';

/** Rank class. A lower class sorts first, as in the backend selector. */
export type SoonestResetClass = 'known' | 'unknown' | 'exhausted';

const CLASS_ORDER: Record<SoonestResetClass, number> = {
  known: 0,
  unknown: 1,
  exhausted: 2,
};

export interface SoonestResetSnapshot {
  provider: QuotaProviderType;
  quota: QuotaCardState | undefined;
}

export interface SoonestResetCredentialInput {
  /** Routing member key. Identity and final tie-break. */
  key: string;
  /** Credential display name. */
  name: string;
  /** Cached quota snapshots for this credential (several for multi-identity files). */
  snapshots: SoonestResetSnapshot[];
}

export interface SoonestResetCredentialRank {
  key: string;
  name: string;
  class: SoonestResetClass;
  /** Deciding reset instant, null when unknown or exhausted without a reset. */
  resetAtMs: number | null;
  /** Position in board order, best first. */
  rank: number;
}

export interface SoonestResetTier {
  id: string;
  memberKeys: readonly string[];
}

export interface SoonestResetPrediction {
  /** Best first. */
  ranks: SoonestResetCredentialRank[];
  byKey: Map<string, SoonestResetCredentialRank>;
  /** Predicted next credential member key per tier, null when the tier holds none. */
  nextByTier: Map<string, string | null>;
  /** Pick of the first tier (in order) that holds a credential, else null. */
  nextKey: string | null;
}

const compareRanks = (
  left: SoonestResetCredentialRank,
  right: SoonestResetCredentialRank
): number =>
  CLASS_ORDER[left.class] - CLASS_ORDER[right.class] ||
  (left.resetAtMs ?? Number.POSITIVE_INFINITY) -
    (right.resetAtMs ?? Number.POSITIVE_INFINITY) ||
  (left.key < right.key ? -1 : left.key > right.key ? 1 : 0);

function rankCredential(
  input: SoonestResetCredentialInput,
  nowMs: number
): SoonestResetCredentialRank {
  let exhausted = false;
  let bestAtMs: number | null = null;
  let bestPeriodHours = -1;
  let bestId = '';

  for (const snapshot of input.snapshots) {
    const observations = getQuotaObservations(snapshot.provider, snapshot.quota);
    for (const window of observations.windows) {
      if (!window.recovers) continue;
      // Elapsed resets describe quota that already renewed.
      if (window.resetAtMs !== null && window.resetAtMs <= nowMs) continue;
      if (window.remainingPercent !== null && window.remainingPercent <= 0) {
        exhausted = true;
        continue;
      }
      if (window.resetAtMs === null) continue;
      const periodHours = window.periodHours ?? 0;
      if (
        periodHours > bestPeriodHours ||
        (periodHours === bestPeriodHours &&
          (window.resetAtMs < (bestAtMs ?? Number.POSITIVE_INFINITY) ||
            (window.resetAtMs === bestAtMs && window.id < bestId)))
      ) {
        bestPeriodHours = periodHours;
        bestAtMs = window.resetAtMs;
        bestId = window.id;
      }
    }
  }

  return {
    key: input.key,
    name: input.name,
    class: exhausted ? 'exhausted' : bestAtMs !== null ? 'known' : 'unknown',
    resetAtMs: exhausted ? null : bestAtMs,
    rank: -1,
  };
}

export function predictSoonestReset(
  inputs: readonly SoonestResetCredentialInput[],
  tiers: readonly SoonestResetTier[],
  nowMs: number
): SoonestResetPrediction {
  const ranks = inputs.map((input) => rankCredential(input, nowMs)).sort(compareRanks);
  ranks.forEach((rank, index) => {
    rank.rank = index;
  });
  const byKey = new Map(ranks.map((rank) => [rank.key, rank]));
  const orderOf = (key: string): number => byKey.get(key)?.rank ?? Number.POSITIVE_INFINITY;

  const nextByTier = new Map<string, string | null>();
  for (const tier of tiers) {
    let pick: string | null = null;
    for (const key of tier.memberKeys) {
      if (!byKey.has(key)) continue;
      if (pick === null || orderOf(key) < orderOf(pick)) pick = key;
    }
    nextByTier.set(tier.id, pick);
  }

  let nextKey: string | null = null;
  for (const tier of tiers) {
    const pick = nextByTier.get(tier.id);
    if (pick) {
      nextKey = pick;
      break;
    }
  }

  return { ranks, byKey, nextByTier, nextKey };
}

/**
 * Collect the cached quota snapshots for one routing credential.
 *
 * Matches the store entries whose physical file is `credentialName`, covering
 * both plain keys and multi-identity keys (`name + separator + auth_index`).
 * Returns one snapshot per match; the caller ranks over their union.
 */
export function resolveCredentialSnapshots(
  files: readonly AuthFileItem[],
  quotaState: QuotaStore,
  credentialName: string
): SoonestResetSnapshot[] {
  const file = files.find((candidate) => candidate.name === credentialName);
  const provider = file ? resolveQuotaProviderType(file) : null;
  const cacheKey = file ? getQuotaCacheKey(file) : credentialName;
  const snapshots: SoonestResetSnapshot[] = [];

  for (const type of QUOTA_TAB_ORDER) {
    if (provider !== null && type !== provider) continue;
    const adapter = QUOTA_PROVIDER_DATA[type];
    const records = adapter.storeSelector(quotaState) as Record<string, QuotaCardState>;
    for (const [key, quota] of Object.entries(records)) {
      if (key === cacheKey || getQuotaCacheFileName(key) === credentialName) {
        snapshots.push({ provider: type, quota });
      }
    }
  }
  return snapshots;
}
