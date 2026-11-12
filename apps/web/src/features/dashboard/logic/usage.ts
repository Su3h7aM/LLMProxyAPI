/**
 * View model for the dashboard usage section.
 *
 * The API returns counters, not cards. This module turns one summary into the
 * four numbers the section shows, with the change against the previous period,
 * and names the estimate and unpriced marks. It is pure so the render tests do
 * not need the network.
 */

import type {
  UsageBreakdownItem,
  UsageBreakdownResponse,
  UsageSummaryResponse,
  UsageTimeseriesResponse,
} from '@/types/usageStats';
import {
  computeUsageDelta,
  computeUsageRateDelta,
  type UsageDelta,
} from '@/features/usage/logic/delta';
import {
  formatMicroUsd,
  formatMicroUsdExact,
  formatTokenCount,
  formatUsageRate,
} from '@/features/usage/logic/format';
import { hasUnpricedRequests, isCostEstimated } from '@/features/usage/logic/flags';
import { USAGE_RANGE_MS } from '@/features/usage/logic/range';

export type DashboardUsageRange = '24h' | '7d' | '30d';

export const DASHBOARD_USAGE_RANGES: readonly DashboardUsageRange[] = ['24h', '7d', '30d'];

/** Hour buckets for one day; day buckets for the longer ranges. */
export const dashboardUsageBucket = (range: DashboardUsageRange): 'hour' | 'day' =>
  range === '24h' ? 'hour' : 'day';

/** Range query for the dashboard, relative to the moment of the request. */
export const dashboardUsageQuery = (
  range: DashboardUsageRange,
  nowMs: number,
  tz: string
): { from_ms: number; to_ms: number; tz: string } => ({
  from_ms: nowMs - USAGE_RANGE_MS[range],
  to_ms: nowMs,
  tz,
});

export type UsageCardKey = 'requests' | 'tokens' | 'cost' | 'failures';
export type UsageCardDeltaUnit = 'percent' | 'points';

export interface UsageCard {
  key: UsageCardKey;
  value: string;
  /** Full value for a title, when the display form is rounded or compact. */
  title?: string;
  delta: UsageDelta | null;
  deltaUnit: UsageCardDeltaUnit;
  /** Estimated and unpriced marks of the cost card. */
  estimate: boolean;
  unpriced: number;
}

const count = (value: number, locale: string): string => value.toLocaleString(locale);

/** The four cards of the section, in reading order. */
export function buildUsageCards(summary: UsageSummaryResponse, locale: string): UsageCard[] {
  const { totals, previous } = summary;
  return [
    {
      key: 'requests',
      value: count(totals.requests, locale),
      delta: computeUsageDelta(totals.requests, previous.requests),
      deltaUnit: 'percent',
      estimate: false,
      unpriced: 0,
    },
    {
      key: 'tokens',
      value: formatTokenCount(totals.total_tokens, locale),
      title: count(totals.total_tokens, locale),
      delta: computeUsageDelta(totals.total_tokens, previous.total_tokens),
      deltaUnit: 'percent',
      estimate: false,
      unpriced: 0,
    },
    {
      key: 'cost',
      value: formatMicroUsd(totals.cost_micro_usd, locale),
      title: formatMicroUsdExact(totals.cost_micro_usd, locale),
      delta: computeUsageDelta(totals.cost_micro_usd, previous.cost_micro_usd),
      deltaUnit: 'percent',
      estimate: isCostEstimated(totals),
      unpriced: hasUnpricedRequests(totals) ? totals.unpriced_requests : 0,
    },
    {
      key: 'failures',
      value: formatUsageRate(totals.failure_rate, locale),
      delta: computeUsageRateDelta(
        totals.failure_rate,
        previous.failure_rate,
        previous.requests
      ),
      deltaUnit: 'points',
      estimate: false,
      unpriced: 0,
    },
  ];
}

/** Request totals per bucket, in time order; the sparkline has no gaps to fill. */
export function usageSparkPoints(series: UsageTimeseriesResponse | null): number[] {
  return series ? series.points.map((point) => point.counters.requests) : [];
}

/** Top models by the order the API returned (requests, descending by default). */
export function topUsageModels(
  breakdown: UsageBreakdownResponse | null,
  limit = 5
): UsageBreakdownItem[] {
  return breakdown ? breakdown.items.slice(0, limit) : [];
}
