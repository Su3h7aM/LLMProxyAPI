/**
 * OpenCode Go subscription render body: plan row plus rolling (5 hours),
 * weekly and monthly allowance meters with reset labels.
 */

import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import type { OpencodeGoQuotaState } from '@/types';
import { buildResetDisplay } from '@/utils/quota';
import { useNow } from '@/hooks/useNow';
import { QuotaMeter } from '../../components/QuotaMeter';
import { QuotaResetLabel } from '../../components/QuotaResetLabel';
import { collectQuotaRowInstants, pickUrgentRowId } from '../../resetSchedule';
import type { QuotaBodyProps } from '../../types';

export function OpencodeGoQuotaBody({ quota, classes }: QuotaBodyProps<OpencodeGoQuotaState>) {
  const { t, i18n } = useTranslation();
  const now = useNow();
  const soonestRowId = useMemo(
    () => pickUrgentRowId(collectQuotaRowInstants('opencode-go', quota), now),
    [quota, now]
  );
  const rows = quota.rows ?? [];

  return (
    <>
      <div className={classes.codexPlan}>
        <span className={classes.codexPlanItem}>
          <span className={classes.codexPlanLabel}>{t('opencode_go_quota.plan_label')}</span>
          <span className={classes.codexPlanValue}>
            {quota.planName || t('opencode_go_quota.plan_unknown')}
          </span>
        </span>
      </div>
      {rows.length === 0 && (
        <div className={classes.quotaMessage}>{t('opencode_go_quota.empty_data')}</div>
      )}
      {rows.map((row, index) => {
        const remaining =
          row.limit > 0
            ? Math.max(0, Math.min(100, Math.round(((row.limit - row.used) / row.limit) * 100)))
            : row.used > 0
              ? 0
              : null;
        const percentLabel = remaining === null ? '--' : `${remaining}%`;
        const rowLabel = row.labelKey
          ? t(row.labelKey, (row.labelParams ?? {}) as Record<string, string | number>)
          : (row.label ?? '');
        const resetDisplay = buildResetDisplay(null, row.resetAtMs ?? null, now, i18n.resolvedLanguage);
        const soon = row.id === soonestRowId;

        return (
          <div
            key={row.id}
            className={classes.quotaRow}
            title={soon ? t('quota_management.soonest_row_hint') : undefined}
          >
            <div className={classes.quotaRowHeader}>
              <span className={classes.quotaModel}>{rowLabel}</span>
              <div className={classes.quotaMeta}>
                <span className={classes.quotaPercent}>{percentLabel}</span>
                {resetDisplay && (
                  <QuotaResetLabel display={resetDisplay} classes={classes} soon={soon} />
                )}
              </div>
            </div>
            <QuotaMeter percent={remaining} classes={classes} index={index} />
          </div>
        );
      })}
    </>
  );
}
