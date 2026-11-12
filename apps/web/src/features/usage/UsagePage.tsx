/**
 * Deep usage statistics page (`/usage`).
 *
 * The filter state is shared with the dashboard section
 * (`useUsageFilterStore`). The shell owns the summary query for the
 * page-level states (disabled database, empty range, load error); each
 * section below owns its own latest-wins query, so one failing section
 * never blanks the page.
 */

import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { PageHeader } from '@/components/ui/PageHeader';
import { RefreshButton } from '@/components/ui/RefreshButton';
import { EmptyState } from '@/components/ui/EmptyState';
import { Skeleton } from '@/components/ui/Skeleton';
import { formatDateTimeValue } from '@/utils/format';
import { useHeaderRefresh } from '@/hooks/useHeaderRefresh';
import { useNow } from '@/hooks/useNow';
import { size } from '@/design-system';
import { usageApi, isUsageDisabledError } from '@/services/api/usage';
import { useAuthStore, useUsageFilterStore } from '@/stores';
import { usageFiltersToQuery } from '@/features/usage/logic/filters';
import { getBrowserTimeZone } from '@/features/usage/logic/range';
import { useUsageQuery } from '@/features/usage/hooks/useUsageQuery';
import { UsageFilterBar } from './components/UsageFilterBar';
import { UsageSummaryCards } from './components/UsageSummaryCards';
import { UsageTimeline } from './components/UsageTimeline';
import { UsageBreakdown } from './components/UsageBreakdown';
import type { UsageSectionProps } from './components/usageSection';
import styles from './UsagePage.module.scss';

export interface UsagePageBody {
  data: { totals: { requests: number } } | null;
  error: Error | null;
  loading: boolean;
}

/** Page-level states; sections render only when the summary has data. */
export function UsagePageContent({
  body,
  onRetry,
}: {
  body: UsagePageBody;
  onRetry: () => void;
}) {
  const { t } = useTranslation();
  const disabled = body.error ? isUsageDisabledError(body.error) : false;
  const empty = !disabled && body.data !== null && body.data.totals.requests === 0;

  if (body.loading && body.data === null && !body.error) {
    return (
      <div
        className={styles.skeletonGrid}
        role="status"
        aria-label={t('usage_page.state_loading')}
      >
        {Array.from({ length: 6 }, (_, index) => (
          <Skeleton key={index} height={size['96']} />
        ))}
      </div>
    );
  }
  if (disabled) {
    const [before, after] = t('usage_page.disabled_detail').split(
      'observability.usage-db.enabled'
    );
    return (
      <div className={styles.notice} role="status">
        <h2 className={styles.noticeTitle}>{t('usage_page.disabled_title')}</h2>
        <p className={styles.noticeText}>
          {before}
          <code>observability.usage-db.enabled</code>
          {after}
        </p>
      </div>
    );
  }
  if (body.error) {
    return (
      <div className={styles.notice} role="alert">
        <h2 className={styles.noticeTitle}>{t('usage_page.state_error')}</h2>
        <p className={styles.noticeText}>{body.error.message}</p>
        <button
          type="button"
          className={`${styles.resetButton} ${styles.retryButton}`}
          onClick={onRetry}
        >
          {t('usage_page.retry')}
        </button>
      </div>
    );
  }
  if (empty) {
    return (
      <EmptyState
        title={t('usage_page.empty_title')}
        description={t('usage_page.empty_hint')}
      />
    );
  }
  // Sections land in the next commits (cards, timeline, breakdown,
  // failures, heatmap, events).
  return null;
}

export function UsagePage() {
  const { t, i18n } = useTranslation();
  const connectionStatus = useAuthStore((state) => state.connectionStatus);
  const enabled = connectionStatus === 'connected';
  const filters = useUsageFilterStore();
  const tz = useMemo(() => getBrowserTimeZone(), []);
  // Minute clock: range presets end "now" without an impure render call.
  const nowMs = useNow(enabled);
  // One refresh action reloads every section.
  const [generation, setGeneration] = useState(0);
  const refreshAll = () => setGeneration((value) => value + 1);
  const summaryQuery = useMemo(
    () =>
      usageFiltersToQuery(
        {
          rangePreset: filters.rangePreset,
          customFromMs: filters.customFromMs,
          customToMs: filters.customToMs,
          provider: filters.provider,
          model: filters.model,
          credential: filters.credential,
          apiKey: filters.apiKey,
          status: filters.status,
          stream: filters.stream,
        },
        nowMs,
        tz
      ),
    [
      filters.rangePreset,
      filters.customFromMs,
      filters.customToMs,
      filters.provider,
      filters.model,
      filters.credential,
      filters.apiKey,
      filters.status,
      filters.stream,
      nowMs,
      tz,
    ]
  );
  const summaryKey = summaryQuery ? JSON.stringify(summaryQuery) : 'invalid-range';

  const summary = useUsageQuery({
    queryKey: `usage-summary|${summaryKey}|${generation}`,
    enabled: enabled && summaryQuery !== null,
    load: (signal) => usageApi.getSummary(summaryQuery ?? {}, { signal }),
  });
  useHeaderRefresh(refreshAll, enabled);

  const sections: UsageSectionProps | null =
    enabled && summaryQuery
      ? { query: summaryQuery, queryKey: `${summaryKey}|${generation}`, enabled: true, tz }
      : null;
  const ready = summary.data !== null && summary.data.totals.requests > 0;

  return (
    <div className={styles.page}>
      <PageHeader
        title={t('usage_page.title')}
        description={t('usage_page.description')}
        meta={
          summary.receivedAt !== null ? (
            <span>
              {t('usage_page.last_updated', {
                time: formatDateTimeValue(summary.receivedAt, i18n.language),
              })}
            </span>
          ) : undefined
        }
        actions={
          <RefreshButton
            onClick={refreshAll}
            label={t('common.refresh')}
            refreshing={summary.loading}
          />
        }
      />
      <UsageFilterBar enabled={enabled} nowMs={nowMs} />
      {summaryQuery === null && (
        <p className={styles.rangeError} role="alert">
          {t('usage_page.range_error')}
        </p>
      )}
      {!enabled && summary.data === null ? (
        <div
          className={styles.skeletonGrid}
          role="status"
          aria-label={t('usage_page.state_loading')}
        >
          {Array.from({ length: 6 }, (_, index) => (
            <Skeleton key={index} height={size['96']} />
          ))}
        </div>
      ) : (
        <UsagePageContent
          body={{ data: summary.data, error: summary.error, loading: summary.loading }}
          onRetry={refreshAll}
        />
      )}
      {ready && sections && summary.data && (
        <>
          <UsageSummaryCards summary={summary.data} />
          <UsageTimeline {...sections} />
          <UsageBreakdown {...sections} />
        </>
      )}
    </div>
  );
}
