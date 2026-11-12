/**
 * Failures: the group table (code, class, provider, credential, model,
 * count, first and last seen) and the recent failures list. A group
 * click applies the provider, credential, and model filters plus the
 * ok/failed status of the class. A recent row opens the event drawer.
 */

import { useState, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/Table';
import { EmptyState } from '@/components/ui/EmptyState';
import { Skeleton } from '@/components/ui/Skeleton';
import { size } from '@/design-system';
import { usageApi, isUsageDisabledError } from '@/services/api/usage';
import { useUsageQuery } from '@/features/usage/hooks/useUsageQuery';
import { breakdownFilterPatch } from '@/features/usage/logic/breakdown';
import { useUsageFilterStore } from '@/stores/useUsageFilterStore';
import { formatDateTimeValue } from '@/utils/format';
import type { UsageFailureEvent, UsageFailureGroup } from '@/types/usageStats';
import type { UsageSectionProps } from './usageSection';
import { UsageEventDrawer } from './UsageEventDrawer';
import styles from '../UsagePage.module.scss';

export interface UsageFailuresViewProps {
  groups: readonly UsageFailureGroup[];
  recent: readonly UsageFailureEvent[];
  onApplyGroup: (group: UsageFailureGroup) => void;
  onOpenEvent: (id: number) => void;
}

/** Presentational groups and recent list; tests render this without the network. */
export function UsageFailuresView({
  groups,
  recent,
  onApplyGroup,
  onOpenEvent,
}: UsageFailuresViewProps) {
  const { t, i18n } = useTranslation();
  return (
    <>
      {groups.length > 0 ? (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('usage_page.col_code')}</TableHead>
              <TableHead>{t('usage_page.col_class')}</TableHead>
              <TableHead>{t('usage_page.col_provider')}</TableHead>
              <TableHead>{t('usage_page.col_credential')}</TableHead>
              <TableHead>{t('usage_page.col_model')}</TableHead>
              <TableHead alignRight>{t('usage_page.col_count')}</TableHead>
              <TableHead>{t('usage_page.col_first')}</TableHead>
              <TableHead>{t('usage_page.col_last')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {groups.map((group, index) => (
              <TableRow
                key={`${group.fail_code}|${group.status_class}|${group.provider}|${group.auth_id}|${group.model}|${index}`}
              >
                <TableCell>
                  <button
                    type="button"
                    className={styles.rowFilter}
                    onClick={() => onApplyGroup(group)}
                    title={t('usage_page.row_filter_hint')}
                  >
                    {group.fail_code || '—'}
                  </button>
                </TableCell>
                <TableCell>{group.status_class || '—'}</TableCell>
                <TableCell>{group.provider || '—'}</TableCell>
                <TableCell>{group.auth_label || group.auth_id || '—'}</TableCell>
                <TableCell>{group.model || '—'}</TableCell>
                <TableCell alignRight>{group.count.toLocaleString(i18n.language)}</TableCell>
                <TableCell>{formatDateTimeValue(group.first_ms, i18n.language)}</TableCell>
                <TableCell>{formatDateTimeValue(group.last_ms, i18n.language)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      ) : (
        <EmptyState title={t('usage_page.groups_empty')} />
      )}
      <h3 className={styles.subTitle}>{t('usage_page.recent_title')}</h3>
      {recent.length > 0 ? (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('usage_page.col_time')}</TableHead>
              <TableHead>{t('usage_page.col_model')}</TableHead>
              <TableHead>{t('usage_page.col_provider')}</TableHead>
              <TableHead>{t('usage_page.col_code')}</TableHead>
              <TableHead>{t('usage_page.detail_fail_summary')}</TableHead>
              <TableHead>
                <span className={styles.srOnly}>{t('usage_page.open_event')}</span>
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {recent.map((failure) => (
              <TableRow key={failure.id}>
                <TableCell>{formatDateTimeValue(failure.ts_ms, i18n.language)}</TableCell>
                <TableCell>{failure.model || '—'}</TableCell>
                <TableCell>{failure.provider || '—'}</TableCell>
                <TableCell>{failure.fail_code || '—'}</TableCell>
                <TableCell>{failure.fail_summary || '—'}</TableCell>
                <TableCell>
                  <button
                    type="button"
                    className={styles.rowFilter}
                    onClick={() => onOpenEvent(failure.id)}
                  >
                    {t('usage_page.open_event')}
                  </button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      ) : (
        <EmptyState title={t('usage_page.recent_empty')} />
      )}
    </>
  );
}

export function UsageFailures({ query, queryKey, enabled }: UsageSectionProps) {
  const { t } = useTranslation();
  const store = useUsageFilterStore();
  const [eventId, setEventId] = useState<number | null>(null);

  const result = useUsageQuery({
    queryKey: `usage-failures|${queryKey}`,
    enabled,
    load: (signal) => usageApi.getFailures({ ...query }, { signal }),
  });

  const disabled = result.error ? isUsageDisabledError(result.error) : false;

  const applyGroup = (group: UsageFailureGroup) => {
    const values = useUsageFilterStore.getState();
    if (group.provider) {
      const patch = breakdownFilterPatch('providers', group.provider, values);
      if ('provider' in patch) store.setMultiFilter('provider', patch.provider);
    }
    if (group.auth_id) {
      const patch = breakdownFilterPatch('credentials', group.auth_id, values);
      if ('credential' in patch) store.setMultiFilter('credential', patch.credential);
    }
    if (group.model) {
      const patch = breakdownFilterPatch('models', group.model, values);
      if ('model' in patch) store.setMultiFilter('model', patch.model);
    }
    const status = breakdownFilterPatch('status', group.status_class, values);
    if ('status' in status) store.setStatus(status.status);
  };

  let body: ReactNode;
  if (result.loading && !result.data) {
    body = (
      <Skeleton height={size['240']} role="status" aria-label={t('usage_page.state_loading')} />
    );
  } else if (result.error && !disabled) {
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
  } else if (!result.data || (result.data.groups.length === 0 && result.data.recent.length === 0)) {
    // A disabled database blanks every section; the shell already names it.
    body = disabled ? null : <EmptyState title={t('usage_page.groups_empty')} />;
  } else {
    body = (
      <>
        <UsageFailuresView
          groups={result.data.groups}
          recent={result.data.recent}
          onApplyGroup={applyGroup}
          onOpenEvent={setEventId}
        />
        <UsageEventDrawer eventId={eventId} onClose={() => setEventId(null)} />
      </>
    );
  }

  return (
    <section className={styles.section} aria-labelledby="usage-failures-title">
      <div className={styles.sectionHead}>
        <h2 className={styles.sectionTitle} id="usage-failures-title">
          {t('usage_page.failures_title')}
        </h2>
      </div>
      {body}
    </section>
  );
}
