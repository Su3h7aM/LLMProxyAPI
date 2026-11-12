/**
 * Types for the v8 usage observability API.
 *
 * Source: `docs/usage-api-contract.md` (summary, timeseries, breakdown, events,
 * failures, heatmap, export, key labels, maintenance), the status route of
 * issue 0043, and the price routes of issue 0044. Time is epoch milliseconds
 * (`*_ms`), cost is integer micro-USD (`*_micro_usd`, 1,000,000 = 1 USD),
 * and rates are 0 to 1.
 *
 * These shapes replace `types/usage.ts`, which describes a data format the
 * backend no longer serves.
 */

/** Counter block used by summary, timeseries, breakdown, and event aggregates. */
export interface UsageCounters {
  requests: number;
  failures: number;
  failure_rate: number;
  input_tokens: number;
  output_tokens: number;
  reasoning_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
  total_tokens: number;
  cost_micro_usd: number;
  /** Part of `cost_micro_usd` that came from late pricing (`price-book-late`). */
  estimated_cost_micro_usd: number;
  unpriced_requests: number;
  latency_avg_ms: number;
  latency_max_ms: number;
  ttft_avg_ms: number;
}

/** Half-open range: `from_ms` inclusive, `to_ms` exclusive. */
export interface UsageRange {
  from_ms: number;
  to_ms: number;
}

export type UsageLatencySource = 'events' | 'hourly';

export interface UsageLatency {
  p50_ms: number | null;
  p95_ms: number | null;
  source: UsageLatencySource;
}

/** `GET /observability/usage/summary`. */
export interface UsageSummaryResponse {
  range: UsageRange;
  totals: UsageCounters;
  /** The same length of time immediately before `range`. */
  previous: UsageCounters;
  latency: UsageLatency;
  first_event_ms: number;
}

export type UsageBucket = 'hour' | 'day' | 'week' | 'auto';
export type UsageTimeBucket = Exclude<UsageBucket, 'auto'>;
export type UsageTimeseriesGroupBy = 'model' | 'provider' | 'credential' | 'api_key';

export interface UsageTimeseriesPoint {
  ts_ms: number;
  counters: UsageCounters;
}

export interface UsageTimeseriesGroup {
  key: string;
  label: string;
  points: UsageTimeseriesPoint[];
}

/** `GET /observability/usage/timeseries`. */
export interface UsageTimeseriesResponse {
  bucket: UsageTimeBucket;
  tz: string;
  points: UsageTimeseriesPoint[];
  /** Present only with `group_by`. */
  groups?: UsageTimeseriesGroup[];
}

export type UsageBreakdownGroupBy =
  | 'model'
  | 'provider'
  | 'alias'
  | 'credential'
  | 'api_key'
  | 'status_class'
  | 'stream';

export type UsageBreakdownSort = 'requests' | 'tokens' | 'cost' | 'failures' | 'latency';
export type UsageSortOrder = 'desc' | 'asc';

/** Extra fields per group: `provider` for model, `auth_type` and `provider` for credential. */
export interface UsageBreakdownMeta {
  provider?: string;
  auth_type?: string;
}

export interface UsageBreakdownItem {
  key: string;
  label: string;
  /** Share of requests in the filtered range, 0 to 1. */
  share: number;
  counters: UsageCounters;
  meta?: UsageBreakdownMeta;
}

/** `GET /observability/usage/breakdown`. */
export interface UsageBreakdownResponse {
  group_by: UsageBreakdownGroupBy;
  total: number;
  items: UsageBreakdownItem[];
}

export type UsageEventCostSource = 'upstream' | 'price-book' | 'price-book-late' | 'none';

/** One row of the events list. `fail_summary` is only in the detail route. */
export interface UsageEventListItem {
  id: number;
  request_id: string;
  trace_id: string;
  ts_ms: number;
  provider: string;
  model: string;
  alias: string;
  executor: string;
  auth_id: string;
  auth_type: string;
  account_label: string;
  api_key_id: string;
  api_key_label: string;
  stream: boolean;
  status_class: string;
  failed: boolean;
  fail_code: string;
  latency_ms: number;
  ttft_ms: number;
  input_tokens: number;
  output_tokens: number;
  reasoning_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
  total_tokens: number;
  service_tier: string;
  reasoning_effort: string;
  cost_micro_usd: number | null;
  cost_source: UsageEventCostSource;
}

/** The price version used for one event; null when no price matched. */
export interface UsageEventPrice {
  provider: string;
  model: string;
  valid_from_ms: number;
  input: number;
  output: number;
  cache_read: number;
  cache_write: number;
  source: string;
}

/** `GET /observability/usage/events/{id}`. */
export interface UsageEventDetail extends UsageEventListItem {
  fail_summary: string;
  session_id: string;
  access_provider: string;
  auth_index: string;
  price_id: number | null;
  price: UsageEventPrice | null;
}

/** `GET /observability/usage/events`. */
export interface UsageEventsResponse {
  items: UsageEventListItem[];
  /** Opaque cursor for the next page; empty when there is no next page. */
  next_cursor: string;
  total_estimate: number;
  /** Added when `total_estimate` is not exact. */
  total_exact?: boolean;
}

export interface UsageFailureGroup {
  fail_code: string;
  status_class: string;
  provider: string;
  auth_id: string;
  auth_label: string;
  model: string;
  count: number;
  first_ms: number;
  last_ms: number;
}

/** Recent failures carry the list fields plus the failure text. */
export interface UsageFailureEvent extends UsageEventListItem {
  fail_summary: string;
}

/** `GET /observability/usage/failures`. */
export interface UsageFailuresResponse {
  groups: UsageFailureGroup[];
  recent: UsageFailureEvent[];
}

export type UsageHeatmapMetric = 'requests' | 'tokens' | 'cost' | 'failures';

/** `GET /observability/usage/heatmap`: 7 rows (Monday first), 24 values each. */
export interface UsageHeatmapResponse {
  tz: string;
  metric: UsageHeatmapMetric;
  matrix: number[][];
  max: number;
}

export type UsageExportFormat = 'jsonl' | 'csv';

/** One label per hashed client API key. The raw key is never present. */
export interface UsageKeyLabel {
  api_key_id: string;
  label: string;
  seen: boolean;
}

/** `GET /observability/usage/key-labels`. */
export interface UsageKeyLabelsResponse {
  labels: UsageKeyLabel[];
}

export type UsageMaintenanceAction = 'prune' | 'vacuum' | 'checkpoint';

/** `POST /observability/usage/maintenance`. Fields depend on the action. */
export interface UsageMaintenanceResponse {
  action: UsageMaintenanceAction;
  duration_ms: number;
  deleted?: number;
  /** `checkpoint` only. */
  busy?: number;
  /** `checkpoint` only: WAL frames in the log before the checkpoint. */
  log_frames?: number;
  /** `checkpoint` only: WAL frames written back. */
  checkpointed?: number;
}

/** `GET /observability/usage/status` (issue 0043). */
export interface UsageDbStatus {
  enabled: boolean;
  path: string;
  size_bytes: number;
  schema_version: number;
  events: number;
  oldest_ts_ms: number;
  newest_ts_ms: number;
  buffer_depth: number;
  buffer_capacity: number;
  dropped: number;
  write_errors: number;
  last_error: string;
  retention_days: number;
  last_prune_ts_ms: number;
}

export type UsagePriceTierKind = 'context' | 'service' | 'tier';

/** One tier of a price version. Null rates fall back to the base rates. */
export interface UsagePriceTier {
  kind: string;
  threshold_tokens: number;
  service: string;
  input: number | null;
  output: number | null;
  cache_read: number | null;
  cache_write: number | null;
}

/**
 * One versioned price. `input`, `output`, `cache_read`, `cache_write`, and
 * `reasoning` are integer micro-USD per 1M tokens.
 */
export interface UsagePriceItem {
  id: number;
  provider: string;
  model: string;
  valid_from_ms: number;
  valid_to_ms: number | null;
  input: number;
  output: number;
  cache_read: number;
  cache_write: number;
  reasoning: number | null;
  source: string;
  synced_at_ms: number;
  note: string;
  tiers: UsagePriceTier[];
}

/** Query of `GET /observability/usage/prices`. */
export interface UsagePriceListQuery {
  provider?: string;
  model?: string;
  source?: string;
  q?: string;
  limit?: number;
  offset?: number;
}

/** `GET /observability/usage/prices`. */
export interface UsagePriceListResponse {
  prices: UsagePriceItem[];
  total: number;
  limit: number;
  offset: number;
}

/** `GET /observability/usage/prices/history`. */
export interface UsagePriceHistoryResponse {
  provider: string;
  model: string;
  versions: UsagePriceItem[];
}

/** Body of `PUT /observability/usage/prices`; rates are USD per 1M tokens. */
export interface UsagePriceInput {
  provider: string;
  model: string;
  input: number;
  output: number;
  cache_read?: number;
  cache_write?: number;
  reasoning?: number;
  effective_from_ms?: number;
  note?: string;
}

/** `POST /observability/usage/prices/sync`. */
export interface UsagePriceSyncResponse {
  added: number;
  updated: number;
  unchanged: number;
  skipped_overrides: number;
  error: string;
  duration_ms: number;
}

export interface UsageUnmatchedPriceModel {
  provider: string;
  model: string;
  events: number;
}

/** `GET /observability/usage/prices/status`. */
export interface UsagePriceStatusResponse {
  last_sync_ms: number;
  last_attempt_ms: number;
  last_error: string;
  counts_by_source: Record<string, number>;
  unmatched: UsageUnmatchedPriceModel[];
}

/** `POST /observability/usage/prices/price-unpriced`. */
export interface UsagePriceUnpricedResponse {
  priced: number;
  skipped_no_price: number;
  duration_ms: number;
}

/** Body of `POST /observability/usage/prices/correct`. */
export interface UsagePriceCorrectionInput {
  from_ms?: number;
  to_ms?: number;
  provider?: string;
  model?: string;
  confirm: true;
  reason: string;
}

/** `POST /observability/usage/prices/correct`. */
export interface UsagePriceCorrectionResponse {
  correction_id: number;
  events_updated: number;
  old_cost_micro_usd: number;
  new_cost_micro_usd: number;
}

export type UsageStatusFilter = 'ok' | 'failed';

/** Common query parameters of every usage route. */
export interface UsageQueryFilters {
  from_ms?: number;
  to_ms?: number;
  /** IANA time zone for day and week buckets and the heatmap. */
  tz?: string;
  provider?: string[];
  model?: string[];
  alias?: string[];
  auth_id?: string[];
  api_key_id?: string[];
  status?: UsageStatusFilter;
  stream?: boolean;
  /** Text search in events and failures only. */
  q?: string;
}

/** Common filters plus the parameters of one route. */
export interface UsageQueryParams extends UsageQueryFilters {
  bucket?: UsageBucket;
  group_by?: UsageTimeseriesGroupBy | UsageBreakdownGroupBy;
  top?: number;
  sort?: UsageBreakdownSort;
  order?: UsageSortOrder;
  limit?: number;
  offset?: number;
  cursor?: string;
  metric?: UsageHeatmapMetric;
  format?: UsageExportFormat;
  /** Price list only. */
  source?: string;
}
