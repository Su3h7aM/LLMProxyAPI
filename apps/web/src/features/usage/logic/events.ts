/**
 * Pure helpers of the events table: the filter identity without the
 * moving time bounds, and the export query for the current filters.
 */

import type { UsageQueryParams } from '@/types/usageStats';
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
