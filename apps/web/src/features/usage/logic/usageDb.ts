/**
 * Usage database card state.
 *
 * The status route answers 200 even when the database is off (`enabled`
 * false); the query routes answer 503. Both mean "disabled" for the card.
 * A dropped record or a write error is a warning: data is missing, and the
 * card must not look healthy.
 */

import type { UsageDbStatus } from '@/types/usageStats';

export type UsageDbCardState = 'loading' | 'disabled' | 'error' | 'warning' | 'ok';

export interface UsageDbStateInput {
  status: UsageDbStatus | null;
  error: Error | null;
  loading: boolean;
  disabled: boolean;
}

export const hasUsageDbWarnings = (
  status: Pick<UsageDbStatus, 'dropped' | 'write_errors'>
): boolean => status.dropped > 0 || status.write_errors > 0;

export function usageDbCardState({
  status,
  error,
  loading,
  disabled,
}: UsageDbStateInput): UsageDbCardState {
  if (disabled || (status !== null && !status.enabled)) return 'disabled';
  if (status && hasUsageDbWarnings(status)) return 'warning';
  if (status) return 'ok';
  if (error) return 'error';
  if (loading) return 'loading';
  return 'error';
}
