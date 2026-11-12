/**
 * Usage observability client (v8).
 *
 * One function per route under `/v8/management/observability/usage/...`, as
 * defined by `docs/usage-api-contract.md`, the status route of issue 0043, and
 * the price routes of issue 0044. Every route answers 503 when the usage
 * database is disabled; that case becomes `UsageDisabledError` so the UI can
 * show a disabled state instead of a generic failure.
 */

import type { AxiosRequestConfig, AxiosResponse } from 'axios';
import { apiClient } from './client';
import { usageQueryPath } from '@/utils/usageQuery';
import type {
  UsageBreakdownResponse,
  UsageDbStatus,
  UsageEventDetail,
  UsageEventsResponse,
  UsageExportFormat,
  UsageFailuresResponse,
  UsageHeatmapResponse,
  UsageKeyLabel,
  UsageKeyLabelsResponse,
  UsageMaintenanceAction,
  UsageMaintenanceResponse,
  UsagePriceCorrectionInput,
  UsagePriceCorrectionResponse,
  UsagePriceHistoryResponse,
  UsagePriceInput,
  UsagePriceItem,
  UsagePriceListQuery,
  UsagePriceListResponse,
  UsagePriceStatusResponse,
  UsagePriceSyncResponse,
  UsagePriceUnpricedResponse,
  UsageQueryParams,
  UsageSummaryResponse,
  UsageTimeseriesResponse,
} from '@/types/usageStats';

const USAGE_TIMEOUT_MS = 30 * 1000;

export interface UsageRequestOptions {
  signal?: AbortSignal;
  timeout?: number;
}

/** The usage database is disabled on this server (HTTP 503). */
export class UsageDisabledError extends Error {
  readonly status = 503;

  constructor() {
    super('usage db disabled');
    this.name = 'UsageDisabledError';
  }
}

export const isUsageDisabledError = (error: unknown): error is UsageDisabledError =>
  error instanceof UsageDisabledError;

const statusOf = (error: unknown): number | undefined => {
  const status = (error as { status?: unknown } | null)?.status;
  return typeof status === 'number' ? status : undefined;
};

const guardUsageErrors = async <T>(request: () => Promise<T>): Promise<T> => {
  try {
    return await request();
  } catch (error) {
    if (statusOf(error) === 503) throw new UsageDisabledError();
    throw error;
  }
};

const requestConfig = (options?: UsageRequestOptions): AxiosRequestConfig => ({
  timeout: options?.timeout ?? USAGE_TIMEOUT_MS,
  ...(options?.signal ? { signal: options.signal } : {}),
});

const getUsage = <T>(route: string, query: UsageQueryParams, options?: UsageRequestOptions) =>
  guardUsageErrors(() => apiClient.get<T>(usageQueryPath(route, query), requestConfig(options)));

const FILENAME_PATTERN = /filename="?([^";]+)"?/i;

const filenameFromDisposition = (disposition: string | undefined, format: UsageExportFormat) => {
  const match = disposition ? FILENAME_PATTERN.exec(disposition) : null;
  return match?.[1]?.trim() || `usage.${format}`;
};

export interface UsageExportFile {
  blob: Blob;
  filename: string;
}

export const usageApi = {
  /** `GET /observability/usage/status`. */
  getStatus: (options?: UsageRequestOptions): Promise<UsageDbStatus> =>
    guardUsageErrors(() =>
      apiClient.get<UsageDbStatus>('/observability/usage/status', requestConfig(options))
    ),

  /** `GET /observability/usage/summary`. */
  getSummary: (
    query: UsageQueryParams = {},
    options?: UsageRequestOptions
  ): Promise<UsageSummaryResponse> => getUsage('/summary', query, options),

  /** `GET /observability/usage/timeseries`. */
  getTimeseries: (
    query: UsageQueryParams = {},
    options?: UsageRequestOptions
  ): Promise<UsageTimeseriesResponse> => getUsage('/timeseries', query, options),

  /** `GET /observability/usage/breakdown`. */
  getBreakdown: (
    query: UsageQueryParams,
    options?: UsageRequestOptions
  ): Promise<UsageBreakdownResponse> => getUsage('/breakdown', query, options),

  /** `GET /observability/usage/events`. */
  getEvents: (
    query: UsageQueryParams = {},
    options?: UsageRequestOptions
  ): Promise<UsageEventsResponse> => getUsage('/events', query, options),

  /** `GET /observability/usage/events/{id}`. */
  getEvent: (id: number, options?: UsageRequestOptions): Promise<UsageEventDetail> =>
    guardUsageErrors(() =>
      apiClient.get<UsageEventDetail>(
        `/observability/usage/events/${encodeURIComponent(String(id))}`,
        requestConfig(options)
      )
    ),

  /** `GET /observability/usage/failures`. */
  getFailures: (
    query: UsageQueryParams = {},
    options?: UsageRequestOptions
  ): Promise<UsageFailuresResponse> => getUsage('/failures', query, options),

  /** `GET /observability/usage/heatmap`. */
  getHeatmap: (
    query: UsageQueryParams = {},
    options?: UsageRequestOptions
  ): Promise<UsageHeatmapResponse> => getUsage('/heatmap', query, options),

  /**
   * `GET /observability/usage/export`. Streams a file; the caller owns the blob
   * and the download. A 503 is still reported as `UsageDisabledError`.
   */
  exportUsage: (
    query: UsageQueryParams,
    options?: UsageRequestOptions
  ): Promise<UsageExportFile> =>
    guardUsageErrors(async () => {
      const response: AxiosResponse<Blob> = await apiClient.getRaw(
        usageQueryPath('/export', query),
        { ...requestConfig(options), responseType: 'blob' }
      );
      const disposition = response.headers?.['content-disposition'];
      return {
        blob: response.data,
        filename: filenameFromDisposition(
          typeof disposition === 'string' ? disposition : undefined,
          query.format ?? 'jsonl'
        ),
      };
    }),

  /** `GET /observability/usage/key-labels`. */
  getKeyLabels: (options?: UsageRequestOptions): Promise<UsageKeyLabelsResponse> =>
    getUsage('/key-labels', {}, options),

  /** `PUT /observability/usage/key-labels`. An empty label clears the label. */
  putKeyLabel: (
    apiKeyId: string,
    label: string,
    options?: UsageRequestOptions
  ): Promise<UsageKeyLabel> =>
    guardUsageErrors(() =>
      apiClient.put<UsageKeyLabel>(
        '/observability/usage/key-labels',
        { api_key_id: apiKeyId, label },
        requestConfig(options)
      )
    ),

  /** `POST /observability/usage/maintenance`. */
  postMaintenance: (
    action: UsageMaintenanceAction,
    options?: UsageRequestOptions
  ): Promise<UsageMaintenanceResponse> =>
    guardUsageErrors(() =>
      apiClient.post<UsageMaintenanceResponse>(
        '/observability/usage/maintenance',
        { action },
        requestConfig(options)
      )
    ),

  /** `GET /observability/usage/prices`. */
  listPrices: (
    query: UsagePriceListQuery = {},
    options?: UsageRequestOptions
  ): Promise<UsagePriceListResponse> =>
    guardUsageErrors(() =>
      apiClient.get<UsagePriceListResponse>(
        usageQueryPath('/prices', {
          provider: query.provider ? [query.provider] : undefined,
          model: query.model ? [query.model] : undefined,
          source: query.source,
          q: query.q,
          limit: query.limit,
          offset: query.offset,
        }),
        requestConfig(options)
      )
    ),

  /** `GET /observability/usage/prices/history`. */
  getPriceHistory: (
    provider: string,
    model: string,
    options?: UsageRequestOptions
  ): Promise<UsagePriceHistoryResponse> =>
    guardUsageErrors(() =>
      apiClient.get<UsagePriceHistoryResponse>(
        usageQueryPath('/prices/history', { provider: [provider], model: [model] }),
        requestConfig(options)
      )
    ),

  /** `PUT /observability/usage/prices`. Rates are USD per 1M tokens. */
  putPrice: (input: UsagePriceInput, options?: UsageRequestOptions): Promise<UsagePriceItem> =>
    guardUsageErrors(() =>
      apiClient.put<UsagePriceItem>(
        '/observability/usage/prices',
        input,
        requestConfig(options)
      )
    ),

  /** `DELETE /observability/usage/prices/manual`. Restores the synced price from now. */
  deleteManualPrice: (
    provider: string,
    model: string,
    options?: UsageRequestOptions
  ): Promise<{ provider: string; model: string; restored: boolean }> =>
    guardUsageErrors(() =>
      apiClient.delete<{ provider: string; model: string; restored: boolean }>(
        usageQueryPath('/prices/manual', { provider: [provider], model: [model] }),
        requestConfig(options)
      )
    ),

  /** `POST /observability/usage/prices/sync`. */
  syncPrices: (options?: UsageRequestOptions): Promise<UsagePriceSyncResponse> =>
    guardUsageErrors(() =>
      apiClient.post<UsagePriceSyncResponse>(
        '/observability/usage/prices/sync',
        undefined,
        requestConfig(options)
      )
    ),

  /** `GET /observability/usage/prices/status`. */
  getPriceStatus: (options?: UsageRequestOptions): Promise<UsagePriceStatusResponse> =>
    guardUsageErrors(() =>
      apiClient.get<UsagePriceStatusResponse>(
        '/observability/usage/prices/status',
        requestConfig(options)
      )
    ),

  /** `POST /observability/usage/prices/price-unpriced`. */
  priceUnpriced: (
    body: { from_ms?: number; to_ms?: number; provider?: string; model?: string } = {},
    options?: UsageRequestOptions
  ): Promise<UsagePriceUnpricedResponse> =>
    guardUsageErrors(() =>
      apiClient.post<UsagePriceUnpricedResponse>(
        '/observability/usage/prices/price-unpriced',
        body,
        requestConfig(options)
      )
    ),

  /** `POST /observability/usage/prices/correct`. Writes an audit row. */
  correctPrices: (
    input: UsagePriceCorrectionInput,
    options?: UsageRequestOptions
  ): Promise<UsagePriceCorrectionResponse> =>
    guardUsageErrors(() =>
      apiClient.post<UsagePriceCorrectionResponse>(
        '/observability/usage/prices/correct',
        input,
        requestConfig(options)
      )
    ),
};
