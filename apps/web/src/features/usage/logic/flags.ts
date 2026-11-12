import type { UsageCounters } from '@/types/usageStats';

/** Part of the cost comes from late pricing, so the card shows it as an estimate. */
export const isCostEstimated = (counters: Pick<UsageCounters, 'estimated_cost_micro_usd'>): boolean =>
  Number.isFinite(counters.estimated_cost_micro_usd) && counters.estimated_cost_micro_usd > 0;

/** Some requests in the range have no price, so the cost total is incomplete. */
export const hasUnpricedRequests = (counters: Pick<UsageCounters, 'unpriced_requests'>): boolean =>
  Number.isFinite(counters.unpriced_requests) && counters.unpriced_requests > 0;
