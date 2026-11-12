/**
 * Hour-of-week heatmap: 7 rows (Monday first), 24 columns.
 *
 * Cells are an ARIA grid with roving focus and arrow-key movement; every
 * cell names its weekday, hour, and value. The fill uses chart tokens
 * with a computed opacity, so no raw colour or size reaches the styles.
 */

import { useMemo, useRef, useState, type KeyboardEvent, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { SegmentedControl } from '@/components/ui/SegmentedControl';
import { EmptyState } from '@/components/ui/EmptyState';
import { Skeleton } from '@/components/ui/Skeleton';
import { size } from '@/design-system';
import { usageApi } from '@/services/api/usage';
import { useUsageQuery } from '@/features/usage/hooks/useUsageQuery';
import { buildHeatmapCells, heatmapCellLabel } from '@/features/usage/logic/heatmap';
import {
  formatMicroUsd,
  formatTokenCount,
} from '@/features/usage/logic/format';
import type { UsageHeatmapMetric } from '@/types/usageStats';
import type { UsageSectionProps } from './usageSection';
import styles from '../UsagePage.module.scss';

const METRICS: readonly UsageHeatmapMetric[] = ['requests', 'tokens', 'cost', 'failures'];

const METRIC_KEYS: Record<UsageHeatmapMetric, string> = {
  requests: 'usage_page.metric_requests',
  tokens: 'usage_page.metric_tokens',
  cost: 'usage_page.metric_cost',
  failures: 'usage_page.metric_failures',
};

const METRIC_COLORS: Record<UsageHeatmapMetric, string> = {
  requests: 'var(--chart-1)',
  tokens: 'var(--chart-2)',
  cost: 'var(--chart-5)',
  failures: 'var(--chart-4)',
};

/** A Monday; weekday names come from the locale, order stays Monday-first. */
const MONDAY_MS = Date.UTC(2026, 10, 9);

export function UsageHeatmap({ query, queryKey, enabled }: UsageSectionProps) {
  const { t, i18n } = useTranslation();
  const [metric, setMetric] = useState<UsageHeatmapMetric>('requests');
  const [selected, setSelected] = useState({ day: 0, hour: 0 });
  const cellRefs = useRef(new Map<string, HTMLDivElement>());

  const result = useUsageQuery({
    queryKey: `usage-heatmap|${metric}|${queryKey}`,
    enabled,
    load: (signal) => usageApi.getHeatmap({ ...query, metric }, { signal }),
  });

  const weekdays = useMemo(() => {
    const format = new Intl.DateTimeFormat(i18n.language, { weekday: 'long', timeZone: 'UTC' });
    return Array.from({ length: 7 }, (_, day) => format.format(MONDAY_MS + day * 86_400_000));
  }, [i18n.language]);

  const formatValue = useMemo(() => {
    if (metric === 'cost') return (value: number) => formatMicroUsd(value, i18n.language);
    if (metric === 'failures') return (value: number) => value.toLocaleString(i18n.language);
    if (metric === 'tokens') return (value: number) => formatTokenCount(value, i18n.language);
    return (value: number) => value.toLocaleString(i18n.language);
  }, [metric, i18n.language]);

  const cells = useMemo(
    () => (result.data ? buildHeatmapCells(result.data.matrix) : []),
    [result.data]
  );

  const focusCell = (day: number, hour: number) => {
    setSelected({ day, hour });
    cellRefs.current.get(`${day}:${hour}`)?.focus();
  };

  const onGridKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    const { day, hour } = selected;
    const move =
      event.key === 'ArrowUp'
        ? { day: (day + 6) % 7, hour }
        : event.key === 'ArrowDown'
          ? { day: (day + 1) % 7, hour }
          : event.key === 'ArrowLeft'
            ? { day, hour: (hour + 23) % 24 }
            : event.key === 'ArrowRight'
              ? { day, hour: (hour + 1) % 24 }
              : event.key === 'Home'
                ? { day, hour: 0 }
                : event.key === 'End'
                  ? { day, hour: 23 }
                  : null;
    if (!move) return;
    event.preventDefault();
    focusCell(move.day, move.hour);
  };

  let body: ReactNode;
  if (result.loading && !result.data) {
    body = (
      <Skeleton height={size['240']} role="status" aria-label={t('usage_page.state_loading')} />
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
  } else if (!result.data || cells.every((cell) => cell.value === 0)) {
    body = <EmptyState title={t('usage_page.heatmap_empty')} />;
  } else {
    const rows = Array.from({ length: 7 }, (_, day) =>
      cells.filter((cell) => cell.day === day)
    );
    body = (
      <div
        className={styles.heatmap}
        role="grid"
        aria-label={t('usage_page.heatmap_title')}
        aria-rowcount={7}
        aria-colcount={24}
        onKeyDown={onGridKeyDown}
      >
        {rows.map((row, day) => (
          <div key={day} className={styles.heatRow} role="row">
            <div className={styles.heatDay} role="rowheader">
              {weekdays[day]}
            </div>
            {row.map((cell) => {
              const active = selected.day === cell.day && selected.hour === cell.hour;
              const label = heatmapCellLabel(
                weekdays[cell.day],
                cell.hour,
                formatValue(cell.value)
              );
              const fillOpacity = 0.08 + 0.92 * cell.intensity;
              return (
                <div
                  key={`${cell.day}:${cell.hour}`}
                  ref={(node) => {
                    if (node) cellRefs.current.set(`${cell.day}:${cell.hour}`, node);
                    else cellRefs.current.delete(`${cell.day}:${cell.hour}`);
                  }}
                  role="gridcell"
                  tabIndex={active ? 0 : -1}
                  aria-label={label}
                  title={label}
                  className={styles.heatCell}
                  onFocus={() => setSelected({ day: cell.day, hour: cell.hour })}
                >
                  <span
                    className={styles.heatFill}
                    style={{ background: METRIC_COLORS[metric], opacity: fillOpacity }}
                    aria-hidden="true"
                  />
                </div>
              );
            })}
          </div>
        ))}
      </div>
    );
  }

  return (
    <section className={styles.section} aria-labelledby="usage-heatmap-title">
      <div className={styles.sectionHead}>
        <h2 className={styles.sectionTitle} id="usage-heatmap-title">
          {t('usage_page.heatmap_title')}
        </h2>
        <SegmentedControl<UsageHeatmapMetric>
          options={METRICS.map((option) => ({ value: option, label: t(METRIC_KEYS[option]) }))}
          value={metric}
          onChange={setMetric}
          ariaLabel={t('usage_page.metric_label')}
        />
      </div>
      <p className={styles.sectionNote}>{t('usage_page.heatmap_caption')}</p>
      {body}
    </section>
  );
}
