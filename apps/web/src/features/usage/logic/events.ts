/**
 * Pure helpers of the events table: the filter identity without the
 * moving time bounds, and the export query for the current filters.
 */

import type {
  UsageEventListItem,
  UsageQueryParams,
} from '@/types/usageStats';
import type { UsageExportFormat } from '@/types/usageStats';

/** Filter identity without the moving time bounds. */
export function eventsResetKey(query: UsageQueryParams, q: string): string {
  const stable = { ...query };
  delete stable.from_ms;
  delete stable.to_ms;
  return `${JSON.stringify(stable)}|${q}`;
}

/** Export query: the current filters plus the search text and the format. */
export function eventsExportQuery(
  query: UsageQueryParams,
  q: string,
  format: UsageExportFormat
): UsageQueryParams {
  return { ...query, ...(q ? { q } : {}), format };
}

export interface EventsPage {
  id: string;
  items: UsageEventListItem[];
  next: string | null;
}

export interface EventsPageData {
  items: UsageEventListItem[];
  next_cursor: string;
  total_estimate: number;
  total_exact?: boolean;
}

/**
 * Merge one fetched page: replace the page with the same identity
 * (a refetch after refresh or a minute tick), else append it.
 */
export function mergeEventsPages(
  previous: EventsPage[],
  id: string,
  data: EventsPageData
): EventsPage[] {
  const next = data.next_cursor?.trim() ? data.next_cursor : null;
  const entry: EventsPage = { id, items: data.items, next };
  const index = previous.findIndex((page) => page.id === id);
  if (index === -1) return [...previous, entry];
  const merged = [...previous];
  merged[index] = entry;
  return merged;
}
