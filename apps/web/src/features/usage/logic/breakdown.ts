/**
 * Pure helpers of the breakdown section: the contract query per tab and
 * the filter patch a row click applies.
 */

import type { UsageFilterValues } from './filters';
import type {
  UsageBreakdownGroupBy,
  UsageBreakdownItem,
  UsageBreakdownSort,
  UsageQueryParams,
  UsageSortOrder,
} from '@/types/usageStats';

export type BreakdownTab = 'models' | 'providers' | 'credentials' | 'api_keys' | 'status';

export const BREAKDOWN_TABS: readonly BreakdownTab[] = [
  'models',
  'providers',
  'credentials',
  'api_keys',
  'status',
];

export const BREAKDOWN_TAB_GROUP: Record<BreakdownTab, UsageBreakdownGroupBy> = {
  models: 'model',
  providers: 'provider',
  credentials: 'credential',
  api_keys: 'api_key',
  status: 'status_class',
};

export const BREAKDOWN_PAGE_LIMIT = 10;

/** Contract query for one breakdown page. */
export function breakdownQueryFor(
  base: UsageQueryParams,
  tab: BreakdownTab,
  sort: UsageBreakdownSort,
  order: UsageSortOrder,
  offset: number
): UsageQueryParams {
  return {
    ...base,
    group_by: BREAKDOWN_TAB_GROUP[tab],
    sort,
    order,
    limit: BREAKDOWN_PAGE_LIMIT,
    offset,
  };
}

export type BreakdownFilterPatch =
  | { status: 'ok' | 'failed' }
  | { model: string[] }
  | { provider: string[] }
  | { credential: string[] }
  | { apiKey: string[] };

/**
 * Store patch for a row click. Status classes fold onto the ok/failed
 * filter: `2xx` means ok, anything else means failed.
 */
export function breakdownFilterPatch(
  tab: BreakdownTab,
  key: string,
  current: UsageFilterValues
): BreakdownFilterPatch {
  switch (tab) {
    case 'models':
      return { model: current.model.includes(key) ? current.model : [...current.model, key] };
    case 'providers':
      return {
        provider: current.provider.includes(key) ? current.provider : [...current.provider, key],
      };
    case 'credentials':
      return {
        credential: current.credential.includes(key)
          ? current.credential
          : [...current.credential, key],
      };
    case 'api_keys':
      return { apiKey: current.apiKey.includes(key) ? current.apiKey : [...current.apiKey, key] };
    case 'status':
      return { status: key === '2xx' ? 'ok' : 'failed' };
  }
}

/** Display fallback for an unlabeled key: `key-<first 6 of the id>`, never raw. */
export function fallbackKeyLabel(apiKeyId: string): string {
  return `key-${apiKeyId.slice(0, 6)}`;
}

export interface KeyLabelSave {
  read: (key: string) => string | undefined;
  write: (key: string, label: string) => void;
  putKeyLabel: (id: string, label: string) => Promise<unknown>;
}

/**
 * Optimistic key-label save with rollback. Applies the draft (or the
 * `key-<first 6>` fallback when cleared), sends it, and restores the
 * previous label when the request fails. Returns true on success.
 */
export async function saveUsageKeyLabel(
  deps: KeyLabelSave,
  item: Pick<UsageBreakdownItem, 'key' | 'label'>,
  draft: string
): Promise<boolean> {
  const previous = deps.read(item.key) ?? item.label;
  const next = draft.trim();
  deps.write(item.key, next || fallbackKeyLabel(item.key));
  try {
    await deps.putKeyLabel(item.key, next);
    return true;
  } catch {
    deps.write(item.key, previous);
    return false;
  }
}
