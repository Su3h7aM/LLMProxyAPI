/**
 * Usage data for the dashboard section.
 *
 * Three calls (summary, timeseries, top models) share one load so the section
 * has one loading state, one error, and one retry. The range is resolved at
 * request time, so each poll covers the latest window.
 */

import { useMemo } from 'react';
import { usageApi } from '@/services/api';
import type {
  UsageBreakdownResponse,
  UsageSummaryResponse,
  UsageTimeseriesResponse,
} from '@/types/usageStats';
import { getBrowserTimeZone } from '@/features/usage/logic/range';
import { useUsageQuery, type UseUsageQueryResult } from '@/features/usage/hooks/useUsageQuery';
import {
  dashboardUsageBucket,
  dashboardUsageQuery,
  type DashboardUsageRange,
} from '../logic/usage';

export interface DashboardUsageData {
  summary: UsageSummaryResponse;
  series: UsageTimeseriesResponse;
  breakdown: UsageBreakdownResponse;
}

const USAGE_POLL_MS = 30_000;
const USAGE_TIMEOUT_MS = 25_000;

export function useDashboardUsage(
  range: DashboardUsageRange,
  enabled: boolean
): UseUsageQueryResult<DashboardUsageData> {
  const tz = useMemo(() => getBrowserTimeZone(), []);

  return useUsageQuery<DashboardUsageData>({
    queryKey: `dashboard-usage|${range}|${tz}`,
    enabled,
    intervalMs: USAGE_POLL_MS,
    timeoutMs: USAGE_TIMEOUT_MS,
    load: async (signal) => {
      const query = dashboardUsageQuery(range, Date.now(), tz);
      const [summary, series, breakdown] = await Promise.all([
        usageApi.getSummary(query, { signal }),
        usageApi.getTimeseries(
          { ...query, bucket: dashboardUsageBucket(range) },
          { signal }
        ),
        usageApi.getBreakdown(
          { ...query, group_by: 'model', sort: 'requests', limit: 5 },
          { signal }
        ),
      ]);
      return { summary, series, breakdown };
    },
  });
}
