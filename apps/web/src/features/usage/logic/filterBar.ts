/**
 * Pure helpers of the usage filter bar.
 *
 * The component itself only wires these to the shared filter store and the
 * option queries. Keeping the mapping, the chips, and the custom-range
 * handling here makes them testable without rendering.
 */

import {
  usageFiltersToQuery,
  type UsageFilterValues,
  type UsageStatusChoice,
  type UsageStreamChoice,
} from './filters';
import {
  USAGE_RANGE_PRESETS,
  type UsageRangePreset,
} from './range';
import { DEFAULT_USAGE_FILTERS } from '@/stores/useUsageFilterStore';import type { UsageBreakdownGroupBy, UsageQueryParams } from '@/types/usageStats';

export const PRESET_KEYS: Record<Exclude<UsageRangePreset, 'custom'>, string> = {
  '1h': 'usage_page.range_1h',
  '24h': 'usage_page.range_24h',
  '7d': 'usage_page.range_7d',
  '30d': 'usage_page.range_30d',
  '90d': 'usage_page.range_90d',
};

export const FILTER_BAR_PRESETS: readonly UsageRangePreset[] = USAGE_RANGE_PRESETS;

const pad2 = (value: number): string => String(value).padStart(2, '0');

/** Local `datetime-local` value (`YYYY-MM-DDTHH:MM`) for epoch milliseconds. */
export function toDateTimeLocal(ms: number): string {
  const date = new Date(ms);
  if (!Number.isFinite(date.getTime())) return '';
  return (
    `${date.getFullYear()}-${pad2(date.getMonth() + 1)}-${pad2(date.getDate())}` +
    `T${pad2(date.getHours())}:${pad2(date.getMinutes())}`
  );
}

/** Epoch milliseconds for a `datetime-local` value, or null when unusable. */
export function fromDateTimeLocal(value: string): number | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$/.exec(value.trim());
  if (!match) return null;
  const ms = new Date(
    Number(match[1]),
    Number(match[2]) - 1,
    Number(match[3]),
    Number(match[4]),
    Number(match[5]),
    0,
    0
  ).getTime();
  return Number.isFinite(ms) ? ms : null;
}

export interface UsageFilterChip {
  id: string;
  text: string;
  onRemove: () => void;
}

export interface UsageChipActions {
  setMultiFilter: (key: 'provider' | 'model' | 'credential' | 'apiKey', values: readonly string[]) => void;
  setStatus: (status: UsageStatusChoice) => void;
  setStream: (stream: UsageStreamChoice) => void;
}

/** Active filters as removable chips, in display order. */
export function buildUsageFilterChips(
  values: UsageFilterValues,
  t: (key: string) => string,
  actions: UsageChipActions
): UsageFilterChip[] {
  const chips: UsageFilterChip[] = [
    ...values.provider.map((value) => ({
      id: `provider:${value}`,
      text: value,
      onRemove: () =>
        actions.setMultiFilter(
          'provider',
          values.provider.filter((item) => item !== value)
        ),
    })),
    ...values.model.map((value) => ({
      id: `model:${value}`,
      text: value,
      onRemove: () =>
        actions.setMultiFilter(
          'model',
          values.model.filter((item) => item !== value)
        ),
    })),
    ...values.credential.map((value) => ({
      id: `credential:${value}`,
      text: value,
      onRemove: () =>
        actions.setMultiFilter(
          'credential',
          values.credential.filter((item) => item !== value)
        ),
    })),
    ...values.apiKey.map((value) => ({
      id: `apiKey:${value}`,
      text: value,
      onRemove: () =>
        actions.setMultiFilter(
          'apiKey',
          values.apiKey.filter((item) => item !== value)
        ),
    })),
  ];
  if (values.status !== DEFAULT_USAGE_FILTERS.status) {
    chips.push({
      id: `status:${values.status}`,
      text: t(values.status === 'ok' ? 'usage_page.status_ok' : 'usage_page.status_failed'),
      onRemove: () => actions.setStatus(DEFAULT_USAGE_FILTERS.status),
    });
  }
  if (values.stream !== DEFAULT_USAGE_FILTERS.stream) {
    chips.push({
      id: `stream:${values.stream}`,
      text: t(values.stream === 'true' ? 'usage_page.stream_on' : 'usage_page.stream_off'),
      onRemove: () => actions.setStream(DEFAULT_USAGE_FILTERS.stream),
    });
  }
  return chips;
}

/** Custom range draft: the query for the draft bounds, or null when unusable. */
export function resolveCustomDraft(
  draftFrom: string,
  draftTo: string,
  values: UsageFilterValues,
  nowMs: number,
  tz: string
): UsageQueryParams | null {
  const fromMs = fromDateTimeLocal(draftFrom);
  const toMs = fromDateTimeLocal(draftTo);
  if (fromMs === null || toMs === null) return null;
  return usageFiltersToQuery(
    { ...values, rangePreset: 'custom', customFromMs: fromMs, customToMs: toMs },
    nowMs,
    tz
  );
}

export interface OptionDimension {
  key: 'provider' | 'model' | 'credential' | 'apiKey';
  groupBy: UsageBreakdownGroupBy;
  labelKey: string;
}

export const OPTION_DIMENSIONS: readonly OptionDimension[] = [
  { key: 'provider', groupBy: 'provider', labelKey: 'usage_page.filter_provider' },
  { key: 'model', groupBy: 'model', labelKey: 'usage_page.filter_model' },
  { key: 'credential', groupBy: 'credential', labelKey: 'usage_page.filter_credential' },
  { key: 'apiKey', groupBy: 'api_key', labelKey: 'usage_page.filter_api_key' },
];

/** Base filters for the option lists: range, zone, status and stream only. */
export function optionBaseQuery(
  values: UsageFilterValues,
  nowMs: number,
  tz: string
): UsageQueryParams | null {
  const full = usageFiltersToQuery(values, nowMs, tz);
  if (!full) return null;
  return {
    from_ms: full.from_ms,
    to_ms: full.to_ms,
    tz: full.tz,
    status: full.status,
    stream: full.stream,
  };
}
