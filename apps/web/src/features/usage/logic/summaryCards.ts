/**
 * View models for the six summary cards of the Usage page.
 *
 * Counts carry the change against the previous period; the latency cards
 * read the percentile block, which is null with `source:"hourly"` for
 * ranges that only hold hourly aggregates.
 */

import {
  computeUsageDelta,
  computeUsageRateDelta,
  type UsageDelta,
} from './delta';
import {
  formatDurationMs,
  formatMicroUsd,
  formatMicroUsdExact,
  formatTokenCount,
  formatUsageRate,
} from './format';
import { hasUnpricedRequests, isCostEstimated } from './flags';
import type { UsageSummaryResponse } from '@/types/usageStats';

export type UsageSummaryCardKey =
  | 'requests'
  | 'tokens'
  | 'cost'
  | 'failures'
  | 'latency_avg'
  | 'latency_p95';

export type UsageSummaryDeltaUnit = 'percent' | 'points';

export interface UsageSummaryCard {
  key: UsageSummaryCardKey;
  label: string;
  value: string;
  /** Full value for a `title`, when the display form is rounded or compact. */
  title?: string;
  delta: UsageDelta | null;
  deltaUnit: UsageSummaryDeltaUnit;
  /** The p95 card has no previous period in the contract, so no delta line. */
  showDelta: boolean;
  /** Cost marks. */
  estimate: boolean;
  unpriced: number;
  /** Token kinds under the tokens card; the latency note under the p95 card. */
  detail?: string;
  detailTitle?: string;
}

/** Translate hook of the caller; tests pass the key through. */
export type UsageText = (key: string, vars?: Record<string, string | number>) => string;

const count = (value: number, locale: string): string => value.toLocaleString(locale);

/** The six cards in reading order. */
export function buildUsageSummaryCards(
  summary: UsageSummaryResponse,
  locale: string,
  t: UsageText
): UsageSummaryCard[] {
  const { totals, previous, latency } = summary;
  const hourly = latency.source === 'hourly';
  const tokensTitle = [
    `${t('usage_page.tokens_input')} ${count(totals.input_tokens, locale)}`,
    `${t('usage_page.tokens_output')} ${count(totals.output_tokens, locale)}`,
    `${t('usage_page.tokens_reasoning')} ${count(totals.reasoning_tokens, locale)}`,
    `${t('usage_page.tokens_cache_read')} ${count(totals.cache_read_tokens, locale)}`,
    `${t('usage_page.tokens_cache_write')} ${count(totals.cache_write_tokens, locale)}`,
  ].join(' · ');
  return [
    {
      key: 'requests',
      label: t('usage_page.card_requests'),
      value: count(totals.requests, locale),
      delta: computeUsageDelta(totals.requests, previous.requests),
      deltaUnit: 'percent',
      showDelta: true,
      estimate: false,
      unpriced: 0,
    },
    {
      key: 'tokens',
      label: t('usage_page.card_tokens'),
      value: formatTokenCount(totals.total_tokens, locale),
      title: `${count(totals.total_tokens, locale)} (${tokensTitle})`,
      delta: computeUsageDelta(totals.total_tokens, previous.total_tokens),
      deltaUnit: 'percent',
      showDelta: true,
      estimate: false,
      unpriced: 0,
      detail: tokensTitle,
    },
    {
      key: 'cost',
      label: t('usage_page.card_cost'),
      value: formatMicroUsd(totals.cost_micro_usd, locale),
      title: formatMicroUsdExact(totals.cost_micro_usd, locale),
      delta: computeUsageDelta(totals.cost_micro_usd, previous.cost_micro_usd),
      deltaUnit: 'percent',
      showDelta: true,
      estimate: isCostEstimated(totals),
      unpriced: hasUnpricedRequests(totals) ? totals.unpriced_requests : 0,
    },
    {
      key: 'failures',
      label: t('usage_page.card_failures'),
      value: formatUsageRate(totals.failure_rate, locale),
      delta: computeUsageRateDelta(
        totals.failure_rate,
        previous.failure_rate,
        previous.requests
      ),
      deltaUnit: 'points',
      showDelta: true,
      estimate: false,
      unpriced: 0,
    },
    {
      key: 'latency_avg',
      label: t('usage_page.card_latency_avg'),
      value: formatDurationMs(totals.latency_avg_ms),
      delta: computeUsageDelta(totals.latency_avg_ms, previous.latency_avg_ms),
      deltaUnit: 'percent',
      showDelta: true,
      estimate: false,
      unpriced: 0,
    },
    {
      key: 'latency_p95',
      label: t('usage_page.card_latency_p95'),
      value: hourly || latency.p95_ms === null ? t('usage_page.latency_unavailable') : formatDurationMs(latency.p95_ms),
      title: hourly ? t('usage_page.latency_hourly_hint') : undefined,
      delta: null,
      deltaUnit: 'percent',
      showDelta: false,
      estimate: false,
      unpriced: 0,
      detail: hourly ? t('usage_page.latency_hourly_hint') : undefined,
    },
  ];
}
