/**
 * View model for the usage timeline.
 *
 * One metric at a time (requests, tokens, or cost), with an optional
 * grouping into the top series plus the backend-folded `"other"` series.
 * The chart reads `TimeSeriesPoint.values[seriesId]`; the table behind
 * the disclosure reads the same shape.
 */

import type { TimeSeriesPoint, TimeSeriesSeries } from '@/components/ui/TimeSeriesChart';
import type {
  UsageCounters,
  UsageTimeseriesResponse,
  UsageTimeseriesGroupBy,
} from '@/types/usageStats';

export type UsageTimelineMetric = 'requests' | 'tokens' | 'cost';

export type UsageTimelineGroupBy = 'none' | UsageTimeseriesGroupBy;

export const TIMELINE_METRICS: readonly UsageTimelineMetric[] = ['requests', 'tokens', 'cost'];

export const TIMELINE_GROUPS: readonly UsageTimelineGroupBy[] = [
  'none',
  'model',
  'provider',
  'credential',
  'api_key',
];

/** Series palette: theme chart tokens, in order. */
export const TIMELINE_COLORS = [
  'var(--chart-1)',
  'var(--chart-2)',
  'var(--chart-3)',
  'var(--chart-4)',
  'var(--chart-5)',
  'var(--chart-6)',
] as const;

export const metricValue = (counters: UsageCounters, metric: UsageTimelineMetric): number => {
  switch (metric) {
    case 'tokens':
      return counters.total_tokens;
    case 'cost':
      return counters.cost_micro_usd;
    default:
      return counters.requests;
  }
};

export interface UsageTimelineModel {
  data: TimeSeriesPoint[];
  series: TimeSeriesSeries[];
}

/** Build the chart model; `label` names the total series. */
export function buildUsageTimeline(
  response: UsageTimeseriesResponse,
  metric: UsageTimelineMetric,
  totalLabel: string
): UsageTimelineModel {
  const groups = response.groups ?? [];
  if (groups.length === 0) {
    return {
      data: response.points.map((point) => ({
        timestamp: point.ts_ms,
        values: { total: metricValue(point.counters, metric) },
      })),
      series: [{ id: 'total', label: totalLabel, color: TIMELINE_COLORS[0], number: 1 }],
    };
  }
  const stamps = new Map<number, Record<string, number | null>>();
  for (const group of groups) {
    for (const point of group.points) {
      const values = stamps.get(point.ts_ms) ?? {};
      values[group.key] = metricValue(point.counters, metric);
      stamps.set(point.ts_ms, values);
    }
  }
  const ordered = [...stamps.entries()].sort((a, b) => a[0] - b[0]);
  for (const [, values] of ordered) {
    for (const group of groups) {
      if (!(group.key in values)) values[group.key] = 0;
    }
  }
  return {
    data: ordered.map(([timestamp, values]) => ({ timestamp, values })),
    series: groups.map((group, index) => ({
      id: group.key,
      label: group.label || group.key,
      color: TIMELINE_COLORS[index % TIMELINE_COLORS.length],
      number: index + 1,
    })),
  };
}
