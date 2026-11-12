/**
 * Breakdown tabs: models, providers, credentials, API keys, status.
 *
 * Rows carry the share bar, counters, and cost; sorting and paging follow
 * the contract keys. A row click adds the group as a filter. The API key
 * tab edits key labels inline with an optimistic update and a rollback
 * on failure. The raw key is never shown: the API only sends the id and
 * a label, falling back to `key-<first 6>`.
 */

import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { SegmentedControl } from '@/components/ui/SegmentedControl';
import { Select } from '@/components/ui/Select';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/Table';
import { EmptyState } from '@/components/ui/EmptyState';
import { Skeleton } from '@/components/ui/Skeleton';
import { IconCheck, IconPencil, IconX } from '@/components/ui/icons';
import { iconSize, size } from '@/design-system';
import { usageApi } from '@/services/api/usage';
import { useUsageQuery } from '@/features/usage/hooks/useUsageQuery';
import {
  formatDurationMs,
  formatMicroUsd,
  formatMicroUsdExact,
  formatTokenCount,
  formatUsageRate,
} from '@/features/usage/logic/format';
import {
  BREAKDOWN_PAGE_LIMIT,
  BREAKDOWN_TABS,
  breakdownFilterPatch,
  breakdownQueryFor,
  saveUsageKeyLabel,
  type BreakdownTab,
} from '@/features/usage/logic/breakdown';
import { useUsageFilterStore } from '@/stores/useUsageFilterStore';
import type {
  UsageBreakdownItem,
  UsageBreakdownSort,
  UsageSortOrder,
} from '@/types/usageStats';
import type { UsageSectionProps } from './usageSection';
import styles from '../UsagePage.module.scss';

const TAB_KEYS: Record<BreakdownTab, string> = {
  models: 'usage_page.tab_models',
  providers: 'usage_page.tab_providers',
  credentials: 'usage_page.tab_credentials',
  api_keys: 'usage_page.tab_api_keys',
  status: 'usage_page.tab_status',
};

const SORTS: readonly UsageBreakdownSort[] = ['requests', 'tokens', 'cost', 'failures', 'latency'];

const SORT_KEYS: Record<UsageBreakdownSort, string> = {
  requests: 'usage_page.sort_requests',
  tokens: 'usage_page.sort_tokens',
  cost: 'usage_page.sort_cost',
  failures: 'usage_page.sort_failures',
  latency: 'usage_page.sort_latency',
};

function ShareBar({ share, label }: { share: number; label: string }) {
  const percent = Math.max(0, Math.min(1, share)) * 100;
  return (
    <span className={styles.share} role="img" aria-label={label}>
      <span className={styles.shareFill} style={{ width: `${percent}%` }} />
    </span>
  );
}

function KeyLabelCell({
  item,
  override,
  editing,
  draft,
  saving,
  onEdit,
  onFilter,
  onDraft,
  onSave,
  onCancel,
}: {
  item: UsageBreakdownItem;
  override: string | undefined;
  editing: boolean;
  draft: string;
  saving: boolean;
  onEdit: () => void;
  onFilter: () => void;
  onDraft: (value: string) => void;
  onSave: () => void;
  onCancel: () => void;
}) {
  const { t } = useTranslation();
  const display = override ?? item.label;
  if (!editing) {
    return (
      <span className={styles.labelCell}>
        <button
          type="button"
          className={styles.rowFilter}
          onClick={onFilter}
          title={t('usage_page.row_filter_hint')}
        >
          <span className={styles.labelText} title={display}>
            {display}
          </span>
        </button>
        <button
          type="button"
          className={styles.iconButton}
          onClick={onEdit}
          title={t('usage_page.label_edit')}
          aria-label={`${t('usage_page.label_edit')}: ${display}`}
        >
          <IconPencil size={iconSize['12']} aria-hidden="true" />
        </button>
      </span>
    );
  }
  return (
    <span className={styles.labelCell}>
      <input
        className={styles.customInput}
        value={draft}
        onChange={(event) => onDraft(event.target.value)}
        placeholder={t('usage_page.label_placeholder')}
        aria-label={t('usage_page.label_edit')}
        disabled={saving}
      />
      <button
        type="button"
        className={styles.iconButton}
        onClick={onSave}
        title={t('usage_page.label_save')}
        aria-label={t('usage_page.label_save')}
        disabled={saving}
      >
        <IconCheck size={iconSize['12']} aria-hidden="true" />
      </button>
      <button
        type="button"
        className={styles.iconButton}
        onClick={onCancel}
        title={t('usage_page.label_cancel')}
        aria-label={t('usage_page.label_cancel')}
        disabled={saving}
      >
        <IconX size={iconSize['12']} aria-hidden="true" />
      </button>
    </span>
  );
}

export interface UsageBreakdownTableProps {
  tab: BreakdownTab;
  items: readonly UsageBreakdownItem[];
  total: number;
  offset: number;
  overrides: Record<string, string>;
  editingId: string | null;
  draft: string;
  saving: boolean;
  onAddFilter: (item: UsageBreakdownItem) => void;
  onEdit: (item: UsageBreakdownItem) => void;
  onDraft: (value: string) => void;
  onSave: (item: UsageBreakdownItem) => void;
  onCancel: () => void;
  onPrev: () => void;
  onNext: () => void;
}

/** Presentational table with paging; tests render this without the network. */
export function UsageBreakdownTable({
  tab,
  items,
  total,
  offset,
  overrides,
  editingId,
  draft,
  saving,
  onAddFilter,
  onEdit,
  onDraft,
  onSave,
  onCancel,
  onPrev,
  onNext,
}: UsageBreakdownTableProps) {
  const { t, i18n } = useTranslation();
  return (
    <>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('usage_page.col_label')}</TableHead>
            <TableHead>{t('usage_page.col_share')}</TableHead>
            <TableHead alignRight>{t('usage_page.col_requests')}</TableHead>
            <TableHead alignRight>{t('usage_page.col_tokens')}</TableHead>
            <TableHead alignRight>{t('usage_page.col_cost')}</TableHead>
            <TableHead alignRight>{t('usage_page.col_failure_rate')}</TableHead>
            <TableHead alignRight>{t('usage_page.col_avg_latency')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.map((item) => (
            <TableRow key={item.key}>
              <TableCell>
                {tab === 'api_keys' ? (
                  <KeyLabelCell
                    item={item}
                    override={overrides[item.key]}
                    editing={editingId === item.key}
                    draft={draft}
                    saving={saving}
                    onEdit={() => onEdit(item)}
                    onFilter={() => onAddFilter(item)}
                    onDraft={onDraft}
                    onSave={() => onSave(item)}
                    onCancel={onCancel}
                  />
                ) : (
                  <button
                    type="button"
                    className={styles.rowFilter}
                    onClick={() => onAddFilter(item)}
                    title={t('usage_page.row_filter_hint')}
                  >
                    {item.label}
                  </button>
                )}
              </TableCell>
              <TableCell>
                <span className={styles.shareCell}>
                  <ShareBar share={item.share} label={formatUsageRate(item.share, i18n.language)} />
                  <span>{formatUsageRate(item.share, i18n.language)}</span>
                </span>
              </TableCell>
              <TableCell alignRight>
                {item.counters.requests.toLocaleString(i18n.language)}
              </TableCell>
              <TableCell alignRight>
                <span title={item.counters.total_tokens.toLocaleString(i18n.language)}>
                  {formatTokenCount(item.counters.total_tokens, i18n.language)}
                </span>
              </TableCell>
              <TableCell alignRight>
                <span title={formatMicroUsdExact(item.counters.cost_micro_usd, i18n.language)}>
                  {formatMicroUsd(item.counters.cost_micro_usd, i18n.language)}
                </span>
              </TableCell>
              <TableCell alignRight>
                {formatUsageRate(item.counters.failure_rate, i18n.language)}
              </TableCell>
              <TableCell alignRight>{formatDurationMs(item.counters.latency_avg_ms)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <div className={styles.controls}>
        <button
          type="button"
          className={styles.resetButton}
          onClick={onPrev}
          disabled={offset === 0}
        >
          {t('usage_page.page_prev')}
        </button>
        <span className={styles.sectionNote} role="status">
          {t('usage_page.page_status', { shown: items.length, total })}
        </span>
        <button
          type="button"
          className={styles.resetButton}
          onClick={onNext}
          disabled={offset + items.length >= total}
        >
          {t('usage_page.page_next')}
        </button>
      </div>
    </>
  );
}

export function UsageBreakdown({ query, queryKey, enabled }: UsageSectionProps) {
  const { t } = useTranslation();
  const [tab, setTab] = useState<BreakdownTab>('models');
  const [sort, setSort] = useState<UsageBreakdownSort>('requests');
  const [order, setOrder] = useState<UsageSortOrder>('desc');
  const [offset, setOffset] = useState(0);
  const [overrides, setOverrides] = useState<Record<string, string>>({});
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draft, setDraft] = useState('');
  const [saving, setSaving] = useState(false);
  const [labelError, setLabelError] = useState(false);

  const store = useUsageFilterStore();
  // Mirror of the overrides for the label saver; every mutation of the
  // overrides goes through the saver's `write`, so the ref stays current.
  const overridesRef = useRef(overrides);

  const breakdownQuery = useMemo(
    () => breakdownQueryFor(query, tab, sort, order, offset),
    [query, tab, sort, order, offset]
  );
  const result = useUsageQuery({
    queryKey: `usage-breakdown|${tab}|${sort}|${order}|${offset}|${queryKey}`,
    enabled,
    load: (signal) => usageApi.getBreakdown(breakdownQuery, { signal }),
  });

  const total = result.data?.total ?? 0;
  useEffect(() => {
    if (result.data && result.data.items.length === 0 && offset > 0 && total > 0) {
      setOffset(0);
    }
  }, [result.data, offset, total]);

  const addFilter = (item: UsageBreakdownItem) => {
    const patch = breakdownFilterPatch(tab, item.key, store);
    if ('status' in patch) {
      store.setStatus(patch.status);
      return;
    }
    if ('model' in patch) store.setMultiFilter('model', patch.model);
    else if ('provider' in patch) store.setMultiFilter('provider', patch.provider);
    else if ('credential' in patch) store.setMultiFilter('credential', patch.credential);
    else store.setMultiFilter('apiKey', patch.apiKey);
  };

  const saveLabel = (item: UsageBreakdownItem) => {
    // The helper reads through a ref so a retry after a refetch still
    // rolls back to the label the user saw, not a stale closure.
    overridesRef.current = overrides;
    setSaving(true);
    setLabelError(false);
    void saveUsageKeyLabel(
      {
        read: (key) => overridesRef.current[key],
        write: (key, label) =>
          setOverrides((current) => {
            overridesRef.current = { ...current, [key]: label };
            return overridesRef.current;
          }),
        putKeyLabel: (id, label) => usageApi.putKeyLabel(id, label),
      },
      item,
      draft
    )
      .then((saved) => {
        if (saved) setEditingId(null);
        else setLabelError(true);
      })
      .finally(() => {
        setSaving(false);
      });
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
  } else if (!result.data || result.data.items.length === 0) {
    body = <EmptyState title={t('usage_page.chart_empty')} />;
  } else {
    body = (
      <UsageBreakdownTable
        tab={tab}
        items={result.data.items}
        total={result.data.total}
        offset={offset}
        overrides={overrides}
        editingId={editingId}
        draft={draft}
        saving={saving}
        onAddFilter={addFilter}
        onEdit={(item) => {
          setEditingId(item.key);
          setDraft(overrides[item.key] ?? item.label);
          setLabelError(false);
        }}
        onDraft={setDraft}
        onSave={saveLabel}
        onCancel={() => setEditingId(null)}
        onPrev={() => setOffset((value) => Math.max(0, value - BREAKDOWN_PAGE_LIMIT))}
        onNext={() => setOffset((value) => value + BREAKDOWN_PAGE_LIMIT)}
      />
    );
  }

  return (
    <section className={styles.section} aria-labelledby="usage-breakdown-title">
      <div className={styles.sectionHead}>
        <h2 className={styles.sectionTitle} id="usage-breakdown-title">
          {t('usage_page.breakdown_title')}
        </h2>
        <div className={styles.controls}>
          <span className={styles.fieldLabel} id="usage-breakdown-sort">
            {t('usage_page.sort_label')}
          </span>
          <Select
            value={sort}
            options={SORTS.map((option) => ({ value: option, label: t(SORT_KEYS[option]) }))}
            onChange={(value) => {
              setSort(value as UsageBreakdownSort);
              setOffset(0);
            }}
            ariaLabelledBy="usage-breakdown-sort"
          />
          <button
            type="button"
            className={styles.resetButton}
            onClick={() => {
              setOrder((value) => (value === 'desc' ? 'asc' : 'desc'));
              setOffset(0);
            }}
            aria-label={order === 'desc' ? t('usage_page.order_desc') : t('usage_page.order_asc')}
          >
            {order === 'desc' ? '↓' : '↑'}
          </button>
        </div>
      </div>
      <SegmentedControl<BreakdownTab>
        options={BREAKDOWN_TABS.map((option) => ({ value: option, label: t(TAB_KEYS[option]) }))}
        value={tab}
        onChange={(value) => {
          setTab(value);
          setOffset(0);
          setEditingId(null);
        }}
        ariaLabel={t('usage_page.breakdown_title')}
      />
      {tab === 'api_keys' && labelError && (
        <p className={styles.rangeError} role="alert">
          {t('usage_page.label_failed')}
        </p>
      )}
      {body}
    </section>
  );
}
