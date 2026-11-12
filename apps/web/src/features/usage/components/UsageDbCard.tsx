import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Card } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { RefreshButton } from '@/components/ui/RefreshButton';
import { useUsageQuery } from '@/features/usage/hooks/useUsageQuery';
import {
  usageDbCardState,
  type UsageDbCardState,
} from '@/features/usage/logic/usageDb';
import { formatDurationMs } from '@/features/usage/logic/format';
import { useHeaderRefresh } from '@/hooks/useHeaderRefresh';
import { isUsageDisabledError, usageApi } from '@/services/api/usage';
import { useNotificationStore } from '@/stores';
import type {
  UsageDbStatus,
  UsageMaintenanceAction,
  UsageMaintenanceResponse,
} from '@/types/usageStats';
import { formatDateTimeValue, formatFileSize } from '@/utils/format';
import styles from './usageDbCard.module.scss';

interface UsageDbCardContentProps {
  state: UsageDbCardState;
  status: UsageDbStatus | null;
  error: string | null;
  loading: boolean;
  maintenance: UsageMaintenanceResponse | null;
  maintenanceError: string | null;
  maintenanceRunning: UsageMaintenanceAction | null;
  onRefresh: () => void;
  onMaintenance: (action: UsageMaintenanceAction) => void;
}

const timeOrNever = (value: number, locale: string, never: string): string =>
  value > 0 ? formatDateTimeValue(value, locale) : never;

function SummaryItem({ label, value }: { label: string; value: string }) {
  return (
    <div className={styles.item}>
      <div className={styles.label}>{label}</div>
      <div className={styles.value}>{value}</div>
    </div>
  );
}

export function UsageDbCardContent({
  state,
  status,
  error,
  loading,
  maintenance,
  maintenanceError,
  maintenanceRunning,
  onRefresh,
  onMaintenance,
}: UsageDbCardContentProps) {
  const { t, i18n } = useTranslation();
  const locale = i18n.language;
  const never = t('usage_db.never');
  const busy = maintenanceRunning !== null;

  return (
    <Card
      className={styles.card}
      title={t('usage_db.title')}
      extra={
        <RefreshButton
          onClick={onRefresh}
          label={t('usage_db.refresh')}
          refreshing={loading}
          disabled={state === 'disabled'}
        />
      }
    >
      <div data-state={state}>
        <p className={styles.description}>{t('usage_db.description')}</p>

        {state === 'disabled' ? (
          <div className={styles.disabled}>
            <div className={styles.disabledTitle}>{t('usage_db.disabled_title')}</div>
            <p className={styles.description}>{t('usage_db.disabled_detail')}</p>
          </div>
        ) : null}

        {state === 'loading' ? (
          <p className="hint" role="status">
            {t('usage_db.loading')}
          </p>
        ) : null}

        {state === 'error' ? (
          <div className={styles.error} role="alert">
            <p className={styles.errorText}>{t('usage_db.error')}</p>
            {error ? <p className={styles.description}>{error}</p> : null}
            <Button variant="secondary" size="sm" onClick={onRefresh}>
              {t('usage_db.retry')}
            </Button>
          </div>
        ) : null}

        {status && (state === 'ok' || state === 'warning') ? (
          <>
            {state === 'warning' ? (
              <div className="status-badge warning">
                {t('usage_db.warning', {
                  dropped: status.dropped,
                  errors: status.write_errors,
                })}
              </div>
            ) : null}
            <div className={styles.grid}>
              <SummaryItem
                label={t('usage_db.enabled')}
                value={
                  status.enabled ? t('usage_db.enabled_value') : t('usage_db.disabled_value')
                }
              />
              <SummaryItem label={t('usage_db.path')} value={status.path || '—'} />
              <SummaryItem label={t('usage_db.size')} value={formatFileSize(status.size_bytes)} />
              <SummaryItem
                label={t('usage_db.schema_version')}
                value={String(status.schema_version)}
              />
              <SummaryItem
                label={t('usage_db.events')}
                value={status.events.toLocaleString(locale)}
              />
              <SummaryItem
                label={t('usage_db.oldest_event')}
                value={timeOrNever(status.oldest_ts_ms, locale, never)}
              />
              <SummaryItem
                label={t('usage_db.newest_event')}
                value={timeOrNever(status.newest_ts_ms, locale, never)}
              />
              <SummaryItem
                label={t('usage_db.buffer')}
                value={t('usage_db.buffer_value', {
                  depth: status.buffer_depth,
                  capacity: status.buffer_capacity,
                })}
              />
              <SummaryItem label={t('usage_db.dropped')} value={String(status.dropped)} />
              <SummaryItem
                label={t('usage_db.write_errors')}
                value={String(status.write_errors)}
              />
              <SummaryItem
                label={t('usage_db.retention_days')}
                value={t('usage_db.retention_value', { days: status.retention_days })}
              />
              <SummaryItem
                label={t('usage_db.last_prune')}
                value={timeOrNever(status.last_prune_ts_ms, locale, never)}
              />
            </div>
            {status.last_error ? (
              <p className={styles.errorText}>
                {t('usage_db.last_error')}: {status.last_error}
              </p>
            ) : null}
            <div className={styles.actions}>
              <Button
                variant="secondary"
                size="sm"
                loading={maintenanceRunning === 'prune'}
                disabled={busy}
                onClick={() => onMaintenance('prune')}
              >
                {t('usage_db.prune')}
              </Button>
              <Button
                variant="secondary"
                size="sm"
                loading={maintenanceRunning === 'checkpoint'}
                disabled={busy}
                onClick={() => onMaintenance('checkpoint')}
              >
                {t('usage_db.checkpoint')}
              </Button>
              <Button
                variant="danger-outline"
                size="sm"
                loading={maintenanceRunning === 'vacuum'}
                disabled={busy}
                onClick={() => onMaintenance('vacuum')}
              >
                {t('usage_db.vacuum')}
              </Button>
            </div>
            {maintenance ? (
              <p className={styles.result} role="status">
                {maintenance.action === 'prune'
                  ? t('usage_db.prune_result', {
                      deleted: maintenance.deleted ?? 0,
                      duration: formatDurationMs(maintenance.duration_ms),
                    })
                  : maintenance.action === 'checkpoint'
                    ? t('usage_db.checkpoint_result', {
                        checkpointed: maintenance.checkpointed ?? 0,
                        frames: maintenance.log_frames ?? 0,
                        busy: maintenance.busy ?? 0,
                        duration: formatDurationMs(maintenance.duration_ms),
                      })
                    : t('usage_db.vacuum_result', {
                        duration: formatDurationMs(maintenance.duration_ms),
                      })}
              </p>
            ) : null}
            {maintenanceError ? (
              <p className={styles.errorText} role="alert">
                {t('usage_db.action_failed', { error: maintenanceError })}
              </p>
            ) : null}
          </>
        ) : null}
      </div>
    </Card>
  );
}

export function UsageDbCard() {
  const { t } = useTranslation();
  const showNotification = useNotificationStore((state) => state.showNotification);
  const showConfirmation = useNotificationStore((state) => state.showConfirmation);
  const [maintenance, setMaintenance] = useState<UsageMaintenanceResponse | null>(null);
  const [maintenanceError, setMaintenanceError] = useState<string | null>(null);
  const [maintenanceRunning, setMaintenanceRunning] =
    useState<UsageMaintenanceAction | null>(null);

  const query = useUsageQuery<UsageDbStatus>({
    load: (signal) => usageApi.getStatus({ signal }),
    queryKey: 'usage-db-status',
  });
  useHeaderRefresh(() => {
    void query.refresh().catch(() => undefined);
  });

  const runMaintenance = async (action: UsageMaintenanceAction) => {
    setMaintenanceRunning(action);
    setMaintenanceError(null);
    try {
      const result = await usageApi.postMaintenance(action);
      setMaintenance(result);
      await query.refresh();
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : String(error);
      setMaintenanceError(message);
      showNotification(t('usage_db.action_failed', { error: message }), 'error');
    } finally {
      setMaintenanceRunning(null);
    }
  };

  const requestMaintenance = (action: UsageMaintenanceAction) => {
    if (action !== 'vacuum') {
      void runMaintenance(action);
      return;
    }
    showConfirmation({
      title: t('usage_db.vacuum_title'),
      message: t('usage_db.vacuum_message'),
      variant: 'danger',
      confirmText: t('usage_db.vacuum_confirm'),
      onConfirm: () => runMaintenance(action),
    });
  };

  const error = query.error;
  const state = usageDbCardState({
    status: query.data,
    error,
    loading: query.loading,
    disabled: isUsageDisabledError(error),
  });

  return (
    <UsageDbCardContent
      state={state}
      status={query.data}
      error={error && !isUsageDisabledError(error) ? error.message : null}
      loading={query.loading}
      maintenance={maintenance}
      maintenanceError={maintenanceError}
      maintenanceRunning={maintenanceRunning}
      onRefresh={() => void query.refresh().catch(() => undefined)}
      onMaintenance={requestMaintenance}
    />
  );
}
