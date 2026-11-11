/**
 * Soonest-reset readout inside the routing policy column.
 *
 * Shown only when the strategy is `soonest-reset`. It names the predicted
 * next credential and the reset time of each credential, all from cached
 * quota snapshots. It never fetches: without snapshots it points at the
 * Quota page instead.
 */

import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import type { SoonestResetCredentialRank } from '../soonestReset';
import { formatInstantShort, formatRelativeInstant } from '@/utils/quota';
import styles from '../RoutingPage.module.scss';

export interface SoonestResetSectionData {
  ranks: SoonestResetCredentialRank[];
  nextKey: string | null;
}

export function SoonestResetSection({
  data,
  nowMs,
}: {
  data: SoonestResetSectionData;
  nowMs: number;
}) {
  const { t, i18n } = useTranslation();
  if (data.ranks.length === 0) return null;

  const next = data.ranks.find((rank) => rank.key === data.nextKey) ?? null;
  const hasData = data.ranks.some((rank) => rank.class !== 'unknown');
  const locale = i18n.resolvedLanguage;

  const renderReset = (resetAtMs: number) => {
    const absolute = formatInstantShort(resetAtMs);
    return (
      <span className={styles.soonestReset} title={absolute}>
        {formatRelativeInstant(resetAtMs, nowMs, locale)}
      </span>
    );
  };

  return (
    <section className={styles.policySection} aria-label={t('routing.soonest_title')}>
      <h3 className={styles.policyLabel}>{t('routing.soonest_title')}</h3>
      {next ? (
        <p className={styles.soonestNext}>
          <span className={styles.soonestNextLabel}>{t('routing.soonest_next')}</span>
          <span className={styles.soonestName}>{next.name}</span>
          {next.resetAtMs !== null ? renderReset(next.resetAtMs) : null}
        </p>
      ) : null}
      <ul className={styles.soonestList}>
        {data.ranks.map((rank) => (
          <li key={rank.key} className={styles.soonestRow}>
            <span className={styles.soonestName}>{rank.name}</span>
            {rank.class === 'known' && rank.resetAtMs !== null ? (
              renderReset(rank.resetAtMs)
            ) : rank.class === 'exhausted' ? (
              <span className={styles.soonestExhausted}>{t('routing.soonest_exhausted')}</span>
            ) : (
              <span className={styles.soonestEmpty}>{t('routing.soonest_no_reset')}</span>
            )}
          </li>
        ))}
      </ul>
      {hasData ? null : (
        <p className={styles.policyHint}>
          {t('routing.soonest_hint')}{' '}
          <Link to="/quota" className={styles.soonestLink}>
            {t('routing.soonest_hint_quota')}
          </Link>
        </p>
      )}
    </section>
  );
}
