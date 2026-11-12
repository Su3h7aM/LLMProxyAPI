import { useTranslation } from 'react-i18next';
import { Card } from '@/components/ui/Card';
import { RefreshButton } from '@/components/ui/RefreshButton';
import { formatDurationMs } from '@/features/usage/logic/format';
import type { PriceSyncConfigState } from '@/features/usage/logic/prices';
import type { UsagePriceStatusResponse, UsagePriceSyncResponse } from '@/types/usageStats';
import { formatDateTimeValue } from '@/utils/format';
import styles from './pricePanels.module.scss';

interface PriceSyncPanelProps {
  status: UsagePriceStatusResponse | null;
  config: PriceSyncConfigState | null;
  result: UsagePriceSyncResponse | null;
  error: string | null;
  statusError: string | null;
  syncing: boolean;
  onSync: () => void;
}

const timeOrNever = (value: number, locale: string, never: string): string =>
  value > 0 ? formatDateTimeValue(value, locale) : never;

export function PriceSyncPanel({
  status,
  config,
  result,
  error,
  statusError,
  syncing,
  onSync,
}: PriceSyncPanelProps) {
  const { t, i18n } = useTranslation();
  const locale = i18n.language;
  const never = t('usage_prices.sync.never');
  const counts = Object.entries(status?.counts_by_source ?? {}).sort(([left], [right]) =>
    left.localeCompare(right)
  );

  return (
    <Card
      className={styles.panel}
      title={t('usage_prices.sync.title')}
      extra={
        <RefreshButton
          onClick={onSync}
          label={t('usage_prices.sync.sync_now')}
          refreshingLabel={t('usage_prices.sync.syncing')}
          refreshing={syncing}
        />
      }
    >
      <div className={styles.summaryGrid}>
        <div className={styles.summaryItem}>
          <div className={styles.summaryLabel}>{t('usage_prices.sync.last_sync')}</div>
          <div className={styles.summaryValue}>
            {timeOrNever(status?.last_sync_ms ?? 0, locale, never)}
          </div>
        </div>
        <div className={styles.summaryItem}>
          <div className={styles.summaryLabel}>{t('usage_prices.sync.last_attempt')}</div>
          <div className={styles.summaryValue}>
            {timeOrNever(status?.last_attempt_ms ?? 0, locale, never)}
          </div>
        </div>
        <div className={styles.summaryItem}>
          <div className={styles.summaryLabel}>{t('usage_prices.sync.counts')}</div>
          <div className={styles.summaryValue}>
            {counts.length === 0
              ? t('usage_prices.sync.counts_empty')
              : counts
                  .map(([source, count]) =>
                    t('usage_prices.sync.counts_value', { source, count })
                  )
                  .join(' · ')}
          </div>
        </div>
        {config ? (
          <div className={styles.summaryItem}>
            <div className={styles.summaryLabel}>{t('usage_prices.sync.config_title')}</div>
            <div className={styles.summaryValue}>
              {config.enabled
                ? t('usage_prices.sync.config_enabled')
                : t('usage_prices.sync.config_disabled')}
              {' · '}
              {t('usage_prices.sync.config_interval', { hours: config.intervalHours })}
            </div>
            {config.url ? (
              <div className={`${styles.summaryValue} ${styles.mono}`}>
                {t('usage_prices.sync.config_url', { url: config.url })}
              </div>
            ) : null}
            <div className={styles.hint}>{t('usage_prices.sync.config_readonly')}</div>
          </div>
        ) : null}
      </div>

      {status?.last_error ? (
        <p className={styles.errorText} role="alert">
          {t('usage_prices.sync.last_error')}: {status.last_error}
        </p>
      ) : null}
      {statusError ? (
        <p className={styles.errorText} role="alert">
          {t('usage_prices.load_error')} {statusError}
        </p>
      ) : null}

      {result ? (
        <p className={styles.hint} role="status">
          {t('usage_prices.sync.result')}: {' '}
          {t('usage_prices.sync.result_counts', {
            added: result.added,
            updated: result.updated,
            unchanged: result.unchanged,
            skipped: result.skipped_overrides,
          })}{' '}
          {t('usage_prices.sync.result_duration', {
            duration: formatDurationMs(result.duration_ms),
          })}
          {result.error ? (
            <span className={styles.errorText}> {result.error}</span>
          ) : null}
        </p>
      ) : null}
      {error ? (
        <p className={styles.errorText} role="alert">
          {t('usage_prices.sync.failed', { error })}
        </p>
      ) : null}
    </Card>
  );
}
