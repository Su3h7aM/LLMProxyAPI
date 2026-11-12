/**
 * Stable mapping from usage filters to the query string of the v8 usage API.
 *
 * Multi-value filters repeat the parameter (`provider=a&provider=b`), as the
 * contract requires. Empty values are omitted. Key order is fixed so the
 * same filters always produce the same string, which also makes the query a
 * safe cache and request key.
 */

import type { UsageEventsResponse, UsageQueryParams } from '@/types/usageStats';

/** Key order of the contract: common filters first, then route-specific ones. */
const QUERY_KEYS = [
  'from_ms',
  'to_ms',
  'tz',
  'provider',
  'model',
  'alias',
  'auth_id',
  'api_key_id',
  'status',
  'stream',
  'q',
  'bucket',
  'group_by',
  'top',
  'sort',
  'order',
  'limit',
  'offset',
  'cursor',
  'metric',
  'format',
  'source',
] as const satisfies readonly (keyof UsageQueryParams)[];

const MULTI_KEYS = new Set<keyof UsageQueryParams>([
  'provider',
  'model',
  'alias',
  'auth_id',
  'api_key_id',
]);

const cleanValues = (values: readonly string[]): string[] =>
  Array.from(new Set(values.map((value) => value.trim()).filter(Boolean)));

const appendValue = (
  params: URLSearchParams,
  key: string,
  value: string | number | boolean | undefined
) => {
  if (value === undefined) return;
  const text = typeof value === 'string' ? value.trim() : String(value);
  if (text.length > 0) params.append(key, text);
};

/** Query string without the leading `?`; empty when no filter is set. */
export function toUsageQueryString(query: UsageQueryParams): string {
  const params = new URLSearchParams();
  for (const key of QUERY_KEYS) {
    const value = query[key];
    if (value === undefined) continue;
    if (MULTI_KEYS.has(key)) {
      for (const item of cleanValues(value as string[])) params.append(key, item);
      continue;
    }
    appendValue(params, key, value as string | number | boolean | undefined);
  }
  return params.toString();
}

/** Full v8 path for one usage route, with the query when it is not empty. */
export function usageQueryPath(
  route: string,
  query: UsageQueryParams = {},
  base = '/observability/usage'
): string {
  const suffix = toUsageQueryString(query);
  return `${base}${route}${suffix ? `?${suffix}` : ''}`;
}

/** Cursor of the next events page, or null at the end. */
export function nextUsageCursor(
  response: Pick<UsageEventsResponse, 'next_cursor'>
): string | null {
  const cursor = response.next_cursor?.trim();
  return cursor ? cursor : null;
}

/** Query for the next events page; the same query when the cursor is empty. */
export function withUsageCursor(
  query: UsageQueryParams,
  cursor: string | null
): UsageQueryParams {
  if (!cursor) {
    const rest = { ...query };
    delete rest.cursor;
    return rest;
  }
  return { ...query, cursor };
}
