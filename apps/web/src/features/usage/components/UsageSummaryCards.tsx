/**
 * The six summary cards: requests, tokens, cost, failure rate, average
 * latency and p95 latency, each with the previous-period delta and the
 * cost estimate and unpriced marks.
 */

import { useTranslation } from 'react-i18next';
import {
  buildUsageSummaryCards,
  type UsageSummaryCard,
  type UsageSummaryDeltaUnit,
} from '@/features/usage/logic/summaryCards';
import type { UsageDelta } from '@/features/usage/logic/delta';
import {
  formatPercentagePoints,
  formatUsageChangePercent,
} from '@/features/usage/logic/format';
import type { UsageSummaryResponse } from '@/types/usageStats';
import styles from '../UsagePage.module.scss';

function DeltaLine({ delta, unit }: { delta: UsageDelta | null; unit: UsageSummaryDeltaUnit }) {
  const { t, i18n } = useTranslation();
  if (!delta) {
    return <span className={styles.deltaNone}>{t('usage_page.delta_no_data')}</span>;
  }
  if (delta.direction === 'flat') {
    return <span className={styles.deltaNone}>{t('usage_page.delta_flat')}</span>;
  }
  const arrow = delta.direction === 'up' ? '↑' : '↓';
  const amount =
    unit === 'points'
      ? formatPercentagePoints(delta.value, i18n.language)
      : formatUsageChangePercent(delta.percent ?? 0, i18n.language);
  const key = unit === 'points' ? 'usage_page.delta_points' : 'usage_page.delta_percent';
  return (
    <span className={delta.direction === 'up' ? styles.deltaUp : styles.deltaDown}>
      {t(key, { arrow, amount })}
    </span>
  );
}

function Card({ card }: { card: UsageSummaryCard }) {
  const { t, i18n } = useTranslation();
  return (
    <div className={styles.card}>
      <dt className={styles.cardLabel}>{card.label}</dt>
      <dd className={styles.cardValue} title={card.title}>
        {card.value}
      </dd>
      {(card.showDelta || card.estimate || card.unpriced > 0) && (
        <dd className={styles.cardMeta}>
          {card.showDelta && <DeltaLine delta={card.delta} unit={card.deltaUnit} />}
          {card.estimate && (
            <span className={styles.mark} title={t('usage_page.estimate_hint')}>
              {t('usage_page.estimate')}
            </span>
          )}
          {card.unpriced > 0 && (
            <span className={styles.mark} title={t('usage_page.unpriced_hint')}>
              {t('usage_page.unpriced', {
                count: card.unpriced,
                formatted: card.unpriced.toLocaleString(i18n.language),
              })}
            </span>
          )}
        </dd>
      )}
      {card.detail && <dd className={styles.cardDetail}>{card.detail}</dd>}
    </div>
  );
}

export function UsageSummaryCards({ summary }: { summary: UsageSummaryResponse }) {
  const { t, i18n } = useTranslation();
  const cards = buildUsageSummaryCards(summary, i18n.language, (key, vars) =>
    t(key, vars as Record<string, string>)
  );
  return (
    <section aria-label={t('usage_page.summary_title')}>
      <dl className={styles.cards}>
        {cards.map((card) => (
          <Card key={card.key} card={card} />
        ))}
      </dl>
    </section>
  );
}
