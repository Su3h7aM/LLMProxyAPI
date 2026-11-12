/**
 * Traffic over time: requests, tokens, or cost per bucket, with an
 * optional top-5-plus-other split. The text table behind the chart
 * disclosure is the accessible alternative.
 */

import { useMemo, useState, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { TimeSeriesChart } from '@/components/ui/TimeSeriesChart';
import { SegmentedControl } from '@/components/ui/SegmentedControl';
import { Select } from '@/components/ui/Select';
import { EmptyState } from '@/components/ui/EmptyState';
import { Skeleton } from '@/components/ui/Skeleton';
import { size } from '@/design-system';
import { usageApi } from '@/services/api/usage';
import { useUsageQuery } from '@/features/usage/hooks/useUsageQuery';
import {
  formatMicroUsd,
  formatTokenCount,
} from '@/features/usage/logic/format';
import {
  buildUsageTimeline,
  TIMELINE_GROUPS,
  TIMELINE_METRICS,
  type UsageTimelineGroupBy,
  type UsageTimelineMetric,
} from '@/features/usage/logic/timeline';
import type { UsageSectionProps } from './usageSection';
import styles from '../UsagePage.module.scss';

const METRIC_KEYS: Record<UsageTimelineMetric, string> = {
  requests: 'usage_page.metric_requests',
  tokens: 'usage_page.metric_tokens',
  cost: 'usage_page.metric_cost',
};

const GROUP_KEYS: Record<UsageTimelineGroupBy, string> = {
  none: 'usage_page.group_none',
  model: 'usage_page.group_model',
  provider: 'usage_page.group_provider',
  credential: 'usage_page.group_credential',
  api_key: 'usage_page.group_api_key',
};

export function UsageTimeline({ query, queryKey, enabled, tz }: UsageSectionProps) {
  const { t, i18n } = useTranslation();
  const [metric, setMetric] = useState<UsageTimelineMetric>('requests');
  const [groupBy, setGroupBy] = useState<UsageTimelineGroupBy>('none');

  const timelineQuery = useMemo(
    () => ({
      ...query,
      bucket: 'auto' as const,
      ...(groupBy === 'none' ? {} : { group_by: groupBy, top: 5 }),
    }),
    [query, groupBy]
  );
  const result = useUsageQuery({
    queryKey: `usage-timeline|${metric}|${queryKey}`,
    enabled,
    load: (signal) => usageApi.getTimeseries(timelineQuery, { signal }),
  });

  const formatX = useMemo(() => {
    const format = new Intl.DateTimeFormat(i18n.language, {
      month: 'short',
      day: 'numeric',
      hour: 'numeric',
      minute: '2-digit',
      timeZone: tz,
    });
    return (timestamp: number) => format.format(timestamp);
  }, [i18n.language, tz]);

  const formatY = useMemo(() => {
    if (metric === 'cost') {
      return (value: number) => formatMicroUsd(value, i18n.language);
    }
    return (value: number) => formatTokenCount(value, i18n.language);
  }, [metric, i18n.language]);

  const model = result.data
    ? buildUsageTimeline(result.data, metric, t(METRIC_KEYS[metric]))
    : null;

  const bucketLabel = result.data
    ? t(
        result.data.bucket === 'hour'
          ? 'dashboard.usage_bucket_hour'
          : 'dashboard.usage_bucket_day'
      )
    : '';

  let body: ReactNode;
  if (result.loading && !result.data) {
    body = (
      <Skeleton
        height={size['240']}
        role="status"
        aria-label={t('usage_page.state_loading')}
      />
    );
  } else if (result.error) {
    body = (
      <div className={styles.notice} role="alert">
        <p className={styles.noticeText}>{result.error.message}</p>
        <button
          type="button"
          className={`${styles.resetButton} ${styles.retryButton}`}
          onClick={() => {
            void result.refresh().catch(() => undefined);
          }}
        >
          {t('usage_page.retry')}
        </button>
      </div>
    );
  } else if (!model || model.data.length === 0) {
    body = <EmptyState title={t('usage_page.chart_empty')} />;
  } else {
    body = (
      <TimeSeriesChart
        id="usage-timeline"
        data={model.data}
        series={model.series}
        xDomain={[query.from_ms ?? 0, query.to_ms ?? 0]}
        formatX={formatX}
        formatY={formatY}
        ariaLabel={`${t('usage_page.timeline_title')}, ${t(METRIC_KEYS[metric])}, ${bucketLabel}`}
        emptyText={t('usage_page.chart_empty')}
      />
    );
  }

  return (
    <section className={styles.section} aria-labelledby="usage-timeline-title">
      <div className={styles.sectionHead}>
        <h2 className={styles.sectionTitle} id="usage-timeline-title">
          {t('usage_page.timeline_title')}
        </h2>
        <div className={styles.controls}>
          <SegmentedControl<UsageTimelineMetric>
            options={TIMELINE_METRICS.map((option) => ({
              value: option,
              label: t(METRIC_KEYS[option]),
            }))}
            value={metric}
            onChange={setMetric}
            ariaLabel={t('usage_page.metric_label')}
          />
          <Select
            value={groupBy}
            options={TIMELINE_GROUPS.map((option) => ({
              value: option,
              label: t(GROUP_KEYS[option]),
            }))}
            onChange={(value) => setGroupBy(value as UsageTimelineGroupBy)}
            ariaLabel={t('usage_page.group_label')}
          />
        </div>
      </div>
      {body}
    </section>
  );
}
