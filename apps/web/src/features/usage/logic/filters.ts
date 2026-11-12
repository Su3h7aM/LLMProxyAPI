/**
 * Shared usage filter vocabulary and its mapping to API query parameters.
 *
 * The dashboard and the Usage page use the same choices. An unusable custom
 * range resolves to null so the caller can show the range error instead of
 * sending a request the API would reject.
 */

import type { UsageQueryParams } from '@/types/usageStats';
import { resolveUsageRange, type UsageRangePreset } from './range';

export const USAGE_STATUS_CHOICES = ['all', 'ok', 'failed'] as const;
export type UsageStatusChoice = (typeof USAGE_STATUS_CHOICES)[number];

export const USAGE_STREAM_CHOICES = ['all', 'true', 'false'] as const;
export type UsageStreamChoice = (typeof USAGE_STREAM_CHOICES)[number];

export interface UsageFilterValues {
  rangePreset: UsageRangePreset;
  customFromMs: number | null;
  customToMs: number | null;
  provider: string[];
  model: string[];
  /** Credential id (`auth_id` in the API). */
  credential: string[];
  /** Hashed API key id (`api_key_id` in the API); never a raw key. */
  apiKey: string[];
  status: UsageStatusChoice;
  stream: UsageStreamChoice;
}

export const DEFAULT_USAGE_FILTERS: Readonly<UsageFilterValues> = Object.freeze({
  rangePreset: '24h',
  customFromMs: null,
  customToMs: null,
  provider: [],
  model: [],
  credential: [],
  apiKey: [],
  status: 'all',
  stream: 'all',
});

/**
 * Translate the shared filters into the contract's query.
 *
 * Empty multi-value lists stay in the object; the query builder drops them.
 * `status` and `stream` become absent when the user chose "all".
 */
export function usageFiltersToQuery(
  values: UsageFilterValues,
  nowMs: number,
  tz: string
): UsageQueryParams | null {
  const range = resolveUsageRange(
    {
      preset: values.rangePreset,
      customFromMs: values.customFromMs,
      customToMs: values.customToMs,
    },
    nowMs
  );
  if (!range) return null;
  return {
    ...range,
    tz,
    provider: values.provider,
    model: values.model,
    auth_id: values.credential,
    api_key_id: values.apiKey,
    status: values.status === 'all' ? undefined : values.status,
    stream: values.stream === 'all' ? undefined : values.stream === 'true',
  };
}
