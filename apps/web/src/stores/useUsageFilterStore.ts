/**
 * Shared usage filters, persisted for the current tab only.
 *
 * The Usage page and the dashboard read the same filter vocabulary. The store
 * writes to `sessionStorage`, never `localStorage`: a filter choice must not
 * follow the browser into a new session. The values are re-validated on read,
 * so a stale or hand-edited payload cannot reach the API.
 */

import { create } from 'zustand';
import {
  DEFAULT_USAGE_FILTERS,
  USAGE_STATUS_CHOICES,
  USAGE_STREAM_CHOICES,
  type UsageFilterValues,
  type UsageStatusChoice,
  type UsageStreamChoice,
} from '@/features/usage/logic/filters';
import { isUsageRangePreset, type UsageRangePreset } from '@/features/usage/logic/range';

export {
  DEFAULT_USAGE_FILTERS,
  USAGE_STATUS_CHOICES,
  USAGE_STREAM_CHOICES,
  type UsageFilterValues,
  type UsageStatusChoice,
  type UsageStreamChoice,
} from '@/features/usage/logic/filters';

export const USAGE_FILTER_STORAGE_KEY = 'usagePage.filters';

type MultiFilterKey = 'provider' | 'model' | 'credential' | 'apiKey';

export interface UsageFilterState extends UsageFilterValues {
  setRangePreset: (preset: UsageRangePreset) => void;
  setCustomRange: (fromMs: number | null, toMs: number | null) => void;
  setMultiFilter: (key: MultiFilterKey, values: readonly string[]) => void;
  setStatus: (status: UsageStatusChoice) => void;
  setStream: (stream: UsageStreamChoice) => void;
  reset: () => void;
}

const stringList = (value: unknown): string[] => {
  if (!Array.isArray(value)) return [];
  return Array.from(
    new Set(value.filter((item): item is string => typeof item === 'string').map((item) => item.trim()))
  ).filter(Boolean);
};

const finiteOrNull = (value: unknown): number | null =>
  typeof value === 'number' && Number.isFinite(value) ? value : null;

const isStatusChoice = (value: unknown): value is UsageStatusChoice =>
  typeof value === 'string' && (USAGE_STATUS_CHOICES as readonly string[]).includes(value);

const isStreamChoice = (value: unknown): value is UsageStreamChoice =>
  typeof value === 'string' && (USAGE_STREAM_CHOICES as readonly string[]).includes(value);

/** Normalize a stored or partial value; unknown fields and values fall back to defaults. */
export function sanitizeUsageFilters(value: unknown): UsageFilterValues {
  if (!value || typeof value !== 'object') return { ...DEFAULT_USAGE_FILTERS };
  const input = value as Record<string, unknown>;
  return {
    rangePreset: isUsageRangePreset(input.rangePreset)
      ? input.rangePreset
      : DEFAULT_USAGE_FILTERS.rangePreset,
    customFromMs: finiteOrNull(input.customFromMs),
    customToMs: finiteOrNull(input.customToMs),
    provider: stringList(input.provider),
    model: stringList(input.model),
    credential: stringList(input.credential),
    apiKey: stringList(input.apiKey),
    status: isStatusChoice(input.status) ? input.status : DEFAULT_USAGE_FILTERS.status,
    stream: isStreamChoice(input.stream) ? input.stream : DEFAULT_USAGE_FILTERS.stream,
  };
}

type ReadableStorage = Pick<Storage, 'getItem'>;
type WritableStorage = Pick<Storage, 'setItem'>;

const browserSessionStorage = (): Storage | null => {
  if (typeof window === 'undefined') return null;
  try {
    return window.sessionStorage ?? null;
  } catch {
    return null;
  }
};

/** Read the tab's filters; null when nothing usable is stored. */
export function readUsageFilters(
  storage: ReadableStorage | null = browserSessionStorage()
): UsageFilterValues | null {
  if (!storage) return null;
  try {
    const raw = storage.getItem(USAGE_FILTER_STORAGE_KEY);
    if (!raw) return null;
    const parsed: unknown = JSON.parse(raw);
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return null;
    return sanitizeUsageFilters(parsed);
  } catch {
    return null;
  }
}

/** Write the tab's filters. Absent storage or a quota error is ignored. */
export function writeUsageFilters(
  values: UsageFilterValues,
  storage: WritableStorage | null = browserSessionStorage()
): void {
  if (!storage) return;
  try {
    storage.setItem(USAGE_FILTER_STORAGE_KEY, JSON.stringify(sanitizeUsageFilters(values)));
  } catch {
    // Session storage can be unavailable (private mode, quota). The in-memory
    // store keeps working for this page view.
  }
}

const pickValues = (state: UsageFilterState): UsageFilterValues => ({
  rangePreset: state.rangePreset,
  customFromMs: state.customFromMs,
  customToMs: state.customToMs,
  provider: state.provider,
  model: state.model,
  credential: state.credential,
  apiKey: state.apiKey,
  status: state.status,
  stream: state.stream,
});

export const useUsageFilterStore = create<UsageFilterState>()((set) => {
  const commit = (patch: Partial<UsageFilterValues>) => {
    set((state) => {
      const next = { ...pickValues(state), ...patch };
      writeUsageFilters(next);
      return patch;
    });
  };

  return {
    ...(readUsageFilters() ?? DEFAULT_USAGE_FILTERS),
    setRangePreset: (rangePreset) => commit({ rangePreset }),
    setCustomRange: (customFromMs, customToMs) => commit({ customFromMs, customToMs }),
    setMultiFilter: (key, values) => commit({ [key]: stringList(values) } as Partial<UsageFilterValues>),
    setStatus: (status) => commit({ status }),
    setStream: (stream) => commit({ stream }),
    reset: () => {
      const defaults = { ...DEFAULT_USAGE_FILTERS };
      writeUsageFilters(defaults);
      set(defaults);
    },
  };
});
