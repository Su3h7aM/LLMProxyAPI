import { useState, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import type { UsageBreakdownItem, UsageTimeseriesResponse } from '@/types/usageStats';
import {
  formatMicroUsd,
  formatMicroUsdExact,
  formatPercentagePoints,
  formatTokenCount,
  formatUsageChangePercent,
  formatUsageRate,
} from '@/features/usage/logic/format';
import type { UsageDelta } from '@/features/usage/logic/delta';
import { isUsageDisabledError } from '@/services/api/usage';
import { useDashboardUsage, type DashboardUsageData } from '../hooks/useDashboardUsage';
import {
  buildUsageCards,
  DASHBOARD_USAGE_RANGES,
  topUsageModels,
  usageSparkPoints,
  type DashboardUsageRange,
  type UsageCard,
  type UsageCardDeltaUnit,
} from '../logic/usage';
import { Sparkline } from './Sparkline';
import styles from './UsagePanel.module.scss';

const RANGE_LABELS: Record<DashboardUsageRange, string> = {
  '24h': 'dashboard.usage_range_24h',
  '7d': 'dashboard.usage_range_7d',
  '30d': 'dashboard.usage_range_30d',
};

const CARD_LABELS: Record<UsageCard['key'], string> = {
  requests: 'dashboard.usage_card_requests',
  tokens: 'dashboard.usage_card_tokens',
  cost: 'dashboard.usage_card_cost',
  failures: 'dashboard.usage_card_failure_rate',
};

function DeltaLine({ delta, unit }: { delta: UsageDelta | null; unit: UsageCardDeltaUnit }) {
  const { t, i18n } = useTranslation();
  if (!delta) {
    return <span className={styles.deltaNone}>{t('dashboard.usage_delta_no_data')}</span>;
  }
  if (delta.direction === 'flat') {
    return <span className={styles.deltaNone}>{t('dashboard.usage_delta_flat')}</span>;
  }
  const arrow = delta.direction === 'up' ? '↑' : '↓';
  const amount =
    unit === 'points'
      ? formatPercentagePoints(delta.value, i18n.language)
      : formatUsageChangePercent(delta.percent ?? 0, i18n.language);
  return (
    <span className={delta.direction === 'up' ? styles.deltaUp : styles.deltaDown}>
      {t(
        unit === 'points'
          ? 'dashboard.usage_delta_points'
          : 'dashboard.usage_delta_percent',
        { arrow, amount }
      )}
    </span>
  );
}

function UsageCards({ data }: { data: DashboardUsageData }) {
  const { t, i18n } = useTranslation();
  const cards = buildUsageCards(data.summary, i18n.language);
  return (
    <dl className={styles.cards}>
      {cards.map((card) => (
        <div key={card.key} className={styles.card}>
          <dt className={styles.cardLabel}>{t(CARD_LABELS[card.key])}</dt>
          <dd className={styles.cardValue} title={card.title}>
            {card.value}
          </dd>
          <dd className={styles.cardMeta}>
            <DeltaLine delta={card.delta} unit={card.deltaUnit} />
            {card.estimate && (
              <span className={styles.mark} title={t('dashboard.usage_estimate_hint')}>
                {t('dashboard.usage_estimate')}
              </span>
            )}
            {card.unpriced > 0 && (
              <span className={styles.mark} title={t('dashboard.usage_unpriced_hint')}>
                {t('dashboard.usage_unpriced', {
                  count: card.unpriced,
                  formatted: card.unpriced.toLocaleString(i18n.language),
                })}
              </span>
            )}
          </dd>
        </div>
      ))}
    </dl>
  );
}

function UsageTrend({
  series,
  range,
}: {
  series: UsageTimeseriesResponse;
  range: DashboardUsageRange;
}) {
  const { t } = useTranslation();
  const points = usageSparkPoints(series);
  const bucket = t(
    series.bucket === 'hour' ? 'dashboard.usage_bucket_hour' : 'dashboard.usage_bucket_day'
  );
  return (
    <figure className={styles.trend}>
      <figcaption className={styles.trendHead}>
        <span>{t('dashboard.usage_over_time')}</span>
        <span className={styles.trendRange}>{t(RANGE_LABELS[range])}</span>
      </figcaption>
      <Sparkline
        points={points}
        ariaLabel={t('dashboard.usage_sparkline_label', {
          bucket,
          range: t(RANGE_LABELS[range]),
        })}
      />
    </figure>
  );
}

function TopModels({ models }: { models: UsageBreakdownItem[] }) {
  const { t, i18n } = useTranslation();
  if (models.length === 0) {
    return <p className={styles.note}>{t('dashboard.usage_no_models')}</p>;
  }
  return (
    <table className={styles.table}>
      <caption className={styles.caption}>{t('dashboard.usage_top_models')}</caption>
      <thead>
        <tr>
          <th scope="col">{t('dashboard.usage_col_model')}</th>
          <th scope="col">{t('dashboard.usage_col_requests')}</th>
          <th scope="col">{t('dashboard.usage_col_share')}</th>
          <th scope="col">{t('dashboard.usage_col_tokens')}</th>
          <th scope="col">{t('dashboard.usage_col_cost')}</th>
        </tr>
      </thead>
      <tbody>
        {models.map((item) => (
          <tr key={item.key}>
            <th scope="row" className={styles.model} title={item.label}>
              {item.label}
            </th>
            <td>{item.counters.requests.toLocaleString(i18n.language)}</td>
            <td>{formatUsageRate(item.share, i18n.language)}</td>
            <td>{formatTokenCount(item.counters.total_tokens, i18n.language)}</td>
            <td title={formatMicroUsdExact(item.counters.cost_micro_usd, i18n.language)}>
              {formatMicroUsd(item.counters.cost_micro_usd, i18n.language)}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

export interface UsagePanelContentProps {
  range: DashboardUsageRange;
  onRangeChange: (range: DashboardUsageRange) => void;
  data: DashboardUsageData | null;
  error: Error | null;
  loading: boolean;
  onRetry: () => void;
}

/** Presentational section; tests render this without the network. */
export function UsagePanelContent({
  range,
  onRangeChange,
  data,
  error,
  loading,
  onRetry,
}: UsagePanelContentProps) {
  const { t, i18n } = useTranslation();
  const rangeLabel = t(RANGE_LABELS[range]);
  const disabled = isUsageDisabledError(error);

  let body: ReactNode;
  if (disabled) {
    body = (
      <div className={styles.off}>
        <p className={styles.offTitle}>{t('dashboard.usage_disabled_title')}</p>
        <p className={styles.note}>{t('dashboard.usage_disabled_detail')}</p>
      </div>
    );
  } else if (error) {
    body = (
      <div className={styles.off} role="alert">
        <p className={styles.offTitle}>{t('dashboard.usage_error')}</p>
        <p className={styles.note}>{error.message}</p>
        <button type="button" className={styles.retry} onClick={onRetry}>
          {t('dashboard.usage_retry')}
        </button>
      </div>
    );
  } else if (!data) {
    body = <p className={styles.note}>{t('dashboard.usage_loading')}</p>;
  } else if (data.summary.totals.requests === 0) {
    body = (
      <div className={styles.off}>
        <p className={styles.offTitle}>{t('dashboard.usage_empty_title')}</p>
        <p className={styles.note}>{t('dashboard.usage_empty_detail', { range: rangeLabel })}</p>
      </div>
    );
  } else {
    const models = topUsageModels(data.breakdown);
    body = (
      <div className={styles.body}>
        <UsageCards data={data} />
        <UsageTrend series={data.series} range={range} />
        <TopModels models={models} />
        <p className={styles.note}>
          {t('dashboard.usage_failures_note', {
            failures: data.summary.totals.failures.toLocaleString(i18n.language),
            requests: data.summary.totals.requests.toLocaleString(i18n.language),
          })}
        </p>
      </div>
    );
  }

  return (
    <div className={styles.panel}>
      <header className={styles.header}>
        <div className={styles.headerText}>
          <h2 className={styles.title}>{t('dashboard.usage_title')}</h2>
          <p className={styles.description}>
            {t('dashboard.usage_description', { range: rangeLabel })}
          </p>
        </div>
        <div className={styles.ranges} role="group" aria-label={t('dashboard.usage_range_label')}>
          {DASHBOARD_USAGE_RANGES.map((option) => (
            <button
              key={option}
              type="button"
              className={[styles.range, option === range ? styles.rangeActive : '']
                .filter(Boolean)
                .join(' ')}
              aria-pressed={option === range}
              onClick={() => onRangeChange(option)}
            >
              {t(RANGE_LABELS[option])}
            </button>
          ))}
        </div>
      </header>
      {body}
      {loading && data && <p className={styles.note}>{t('dashboard.usage_refreshing')}</p>}
    </div>
  );
}

/** Connected dashboard section: local range selector over the shared usage API. */
export function UsagePanel({ enabled }: { enabled: boolean }) {
  const [range, setRange] = useState<DashboardUsageRange>('24h');
  const usage = useDashboardUsage(range, enabled);
  return (
    <UsagePanelContent
      range={range}
      onRangeChange={setRange}
      data={usage.data}
      error={usage.error}
      loading={usage.loading}
      onRetry={() => {
        void usage.refresh().catch(() => undefined);
      }}
    />
  );
}
