/**
 * Shared usage filters: range presets, one-value selects that add chips for
 * provider, model, credential and API key, status and stream segments, and a
 * custom date-time range. The state lives in `useUsageFilterStore`, so the
 * dashboard section and this page read the same values.
 */

import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { SegmentedControl } from '@/components/ui/SegmentedControl';
import { Select } from '@/components/ui/Select';
import { IconX } from '@/components/ui/icons';
import { iconSize } from '@/design-system';
import { usageApi } from '@/services/api/usage';
import { useUsageQuery } from '@/features/usage/hooks/useUsageQuery';
import { USAGE_RANGE_PRESETS, type UsageRangePreset } from '@/features/usage/logic/range';
import {
  buildUsageFilterChips,
  OPTION_DIMENSIONS,
  optionBaseQuery,
  PRESET_KEYS,
  resolveCustomDraft,
  toDateTimeLocal,
  type OptionDimension,
} from '@/features/usage/logic/filterBar';
import {
  useUsageFilterStore,
  type UsageStatusChoice,
  type UsageStreamChoice,
} from '@/stores/useUsageFilterStore';
import type { UsageQueryParams } from '@/types/usageStats';
import styles from '../UsagePage.module.scss';

function DimensionSelect({
  dimension,
  baseQuery,
  queryKey,
  enabled,
}: {
  dimension: OptionDimension;
  baseQuery: UsageQueryParams | null;
  queryKey: string;
  enabled: boolean;
}) {
  const { t } = useTranslation();
  const selected = useUsageFilterStore((state) => state[dimension.key]);
  const setMultiFilter = useUsageFilterStore((state) => state.setMultiFilter);
  const options = useUsageQuery({
    queryKey: `usage-options|${dimension.groupBy}|${queryKey}`,
    enabled: enabled && baseQuery !== null,
    load: (signal) =>
      usageApi.getBreakdown(
        { ...(baseQuery as UsageQueryParams), group_by: dimension.groupBy, limit: 200 },
        { signal }
      ),
  });
  const items = options.data?.items ?? [];
  return (
    <div className={styles.field}>
      <span className={styles.fieldLabel} id={`usage-filter-${dimension.key}`}>
        {t(dimension.labelKey)}
      </span>
      <Select
        value=""
        options={items.map((item) => ({ value: item.key, label: item.label }))}
        onChange={(value) => {
          if (value && !selected.includes(value)) {
            setMultiFilter(dimension.key, [...selected, value]);
          }
        }}
        placeholder={t('usage_page.all_option')}
        ariaLabelledBy={`usage-filter-${dimension.key}`}
        disabled={!enabled || options.loading}
      />
    </div>
  );
}

export function UsageFilterBar({ enabled, nowMs }: { enabled: boolean; nowMs: number }) {
  const { t } = useTranslation();
  const values = useUsageFilterStore();
  const tz = useMemo(() => {
    try {
      const zone = new Intl.DateTimeFormat().resolvedOptions().timeZone;
      return zone && zone.trim() ? zone.trim() : 'UTC';
    } catch {
      return 'UTC';
    }
  }, []);

  const baseQuery = optionBaseQuery(values, nowMs, tz);
  const optionsKey = baseQuery ? JSON.stringify(baseQuery) : 'invalid';

  const [draftFrom, setDraftFrom] = useState(() =>
    toDateTimeLocal(values.customFromMs ?? nowMs - 24 * 60 * 60 * 1000)
  );
  const [draftTo, setDraftTo] = useState(() => toDateTimeLocal(values.customToMs ?? nowMs));
  const [rangeError, setRangeError] = useState(false);

  const applyCustom = () => {
    const check = resolveCustomDraft(draftFrom, draftTo, values, nowMs, tz);
    if (!check) {
      setRangeError(true);
      return;
    }
    values.setCustomRange(check.from_ms ?? null, check.to_ms ?? null);
    setRangeError(false);
    values.setRangePreset('custom');
  };

  const chipList = buildUsageFilterChips(values, t, values);

  return (
    <section className={styles.section} aria-label={t('usage_page.active_filters')}>
      <div className={styles.filters}>
        <SegmentedControl<UsageRangePreset>
          options={USAGE_RANGE_PRESETS.map((preset) => ({
            value: preset,
            label: preset === 'custom' ? t('usage_page.range_custom') : t(PRESET_KEYS[preset]),
          }))}
          value={values.rangePreset}
          onChange={(preset) => {
            setRangeError(false);
            values.setRangePreset(preset);
          }}
          ariaLabel={t('usage_page.range_label')}
          disabled={!enabled}
        />

        {values.rangePreset === 'custom' && (
          <div className={styles.customRow}>
            <label className={styles.field}>
              <span className={styles.fieldLabel}>{t('usage_page.custom_from')}</span>
              <input
                type="datetime-local"
                className={styles.customInput}
                value={draftFrom}
                onChange={(event) => setDraftFrom(event.target.value)}
                disabled={!enabled}
              />
            </label>
            <label className={styles.field}>
              <span className={styles.fieldLabel}>{t('usage_page.custom_to')}</span>
              <input
                type="datetime-local"
                className={styles.customInput}
                value={draftTo}
                onChange={(event) => setDraftTo(event.target.value)}
                disabled={!enabled}
              />
            </label>
            <button
              type="button"
              className={styles.resetButton}
              onClick={applyCustom}
              disabled={!enabled}
            >
              {t('usage_page.custom_apply')}
            </button>
            {rangeError && (
              <p className={styles.rangeError} role="alert">
                {t('usage_page.range_error')}
              </p>
            )}
          </div>
        )}

        <div className={styles.filterGrid}>
          {OPTION_DIMENSIONS.map((dimension) => (
            <DimensionSelect
              key={dimension.key}
              dimension={dimension}
              baseQuery={baseQuery}
              queryKey={optionsKey}
              enabled={enabled}
            />
          ))}
          <div className={styles.field}>
            <span className={styles.fieldLabel} id="usage-filter-status">
              {t('usage_page.filter_status')}
            </span>
            <SegmentedControl<UsageStatusChoice>
              options={[
                { value: 'all', label: t('usage_page.status_all') },
                { value: 'ok', label: t('usage_page.status_ok') },
                { value: 'failed', label: t('usage_page.status_failed') },
              ]}
              value={values.status}
              onChange={values.setStatus}
              ariaLabelledBy="usage-filter-status"
              disabled={!enabled}
            />
          </div>
          <div className={styles.field}>
            <span className={styles.fieldLabel} id="usage-filter-stream">
              {t('usage_page.filter_stream')}
            </span>
            <SegmentedControl<UsageStreamChoice>
              options={[
                { value: 'all', label: t('usage_page.stream_all') },
                { value: 'true', label: t('usage_page.stream_on') },
                { value: 'false', label: t('usage_page.stream_off') },
              ]}
              value={values.stream}
              onChange={values.setStream}
              ariaLabelledBy="usage-filter-stream"
              disabled={!enabled}
            />
          </div>
        </div>

        <div className={styles.controls}>
          {chipList.length > 0 && (
            <ul className={styles.chips} aria-label={t('usage_page.active_filters')}>
              {chipList.map((chip) => (
                <li key={chip.id} className={styles.chip}>
                  <span className={styles.chipText}>{chip.text}</span>
                  <button
                    type="button"
                    className={styles.chipRemove}
                    onClick={chip.onRemove}
                    aria-label={`${t('usage_page.clear_all')}: ${chip.text}`}
                  >
                    <IconX size={iconSize['12']} aria-hidden="true" />
                  </button>
                </li>
              ))}
            </ul>
          )}
          <button type="button" className={styles.resetButton} onClick={() => values.reset()}>
            {t('usage_page.reset')}
          </button>
        </div>
      </div>
    </section>
  );
}
