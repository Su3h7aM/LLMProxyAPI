/**
 * Events table: cursor paging with "load more", debounced text search,
 * key labels (never raw keys), per-row request-id copy, and a JSONL/CSV
 * export of the current filters. A row opens the event drawer.
 */

import { useEffect, useMemo, useState, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/Table';
import { EmptyState } from '@/components/ui/EmptyState';
import { Skeleton } from '@/components/ui/Skeleton';
import { SegmentedControl } from '@/components/ui/SegmentedControl';
import { IconCheck } from '@/components/ui/icons';
import { iconSize, size } from '@/design-system';
import { usageApi, isUsageDisabledError } from '@/services/api/usage';
import { useUsageQuery } from '@/features/usage/hooks/useUsageQuery';
import { nextUsageCursor } from '@/utils/usageQuery';
import { eventsExportQuery, eventsResetKey } from '@/features/usage/logic/events';
import {
  formatDurationMs,
  formatMicroUsd,
  formatMicroUsdExact,
  formatTokenCount,
} from '@/features/usage/logic/format';
import { fallbackKeyLabel } from '@/features/usage/logic/breakdown';
import { formatDateTimeValue } from '@/utils/format';
import type {
  UsageEventListItem,
  UsageEventsResponse,
  UsageExportFormat,
} from '@/types/usageStats';
import type { UsageSectionProps } from './usageSection';
import { UsageEventDrawer } from './UsageEventDrawer';
import styles from '../UsagePage.module.scss';

const PAGE_LIMIT = 50;
const SEARCH_DEBOUNCE_MS = 300;

interface EventsPage {
  id: string;
  items: UsageEventListItem[];
  next: string | null;
  first: boolean;
}

interface EventsTotal {
  estimate: number;
  exact: boolean;
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  window.setTimeout(() => URL.revokeObjectURL(url), 5000);
}

function CostCell({ item }: { item: UsageEventListItem }) {
  const { t, i18n } = useTranslation();
  if (item.cost_micro_usd === null) return <>—</>;
  const estimated = item.cost_source === 'price-book-late';
  return (
    <span>
      <span title={formatMicroUsdExact(item.cost_micro_usd, i18n.language)}>
        {formatMicroUsd(item.cost_micro_usd, i18n.language)}
      </span>
      {estimated && (
        <span className={styles.mark} title={t('usage_page.estimate_hint')}>
          {t('usage_page.estimate')}
        </span>
      )}
    </span>
  );
}

function RequestCell({
  item,
  onOpen,
  copied,
  onCopy,
}: {
  item: UsageEventListItem;
  onOpen: () => void;
  copied: boolean;
  onCopy: () => void;
}) {
  const { t } = useTranslation();
  return (
    <span className={styles.labelCell}>
      <button
        type="button"
        className={`${styles.rowFilter} ${styles.mono}`}
        onClick={onOpen}
        title={item.request_id}
      >
        <span className={styles.labelText}>{item.request_id}</span>
      </button>
      <button
        type="button"
        className={styles.iconButton}
        onClick={onCopy}
        title={t('usage_page.copy_id')}
        aria-label={`${t('usage_page.copy_id')}: ${item.request_id}`}
      >
        <IconCheck size={iconSize['12']} aria-hidden="true" />
        {copied ? <span className={styles.copiedMark}>{t('usage_page.copied')}</span> : null}
      </button>
    </span>
  );
}

export interface UsageEventsTableProps {
  items: readonly UsageEventListItem[];
  totalText: string | null;
  hasMore: boolean;
  loadingMore: boolean;
  copiedId: number | null;
  onLoadMore: () => void;
  onOpenEvent: (id: number) => void;
  onCopyRequest: (item: UsageEventListItem) => void;
}

/** Presentational events table with paging controls; tests render this directly. */
export function UsageEventsTable({
  items,
  totalText,
  hasMore,
  loadingMore,
  copiedId,
  onLoadMore,
  onOpenEvent,
  onCopyRequest,
}: UsageEventsTableProps) {
  const { t, i18n } = useTranslation();
  return (
    <>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('usage_page.col_time')}</TableHead>
            <TableHead>{t('usage_page.col_model')}</TableHead>
            <TableHead>{t('usage_page.col_provider')}</TableHead>
            <TableHead>{t('usage_page.col_credential')}</TableHead>
            <TableHead>{t('usage_page.col_key')}</TableHead>
            <TableHead>{t('usage_page.col_status')}</TableHead>
            <TableHead alignRight>{t('usage_page.col_latency')}</TableHead>
            <TableHead alignRight>{t('usage_page.col_ttft')}</TableHead>
            <TableHead alignRight>{t('usage_page.col_tokens')}</TableHead>
            <TableHead alignRight>{t('usage_page.col_cost')}</TableHead>
            <TableHead>{t('usage_page.col_request')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.map((item) => (
            <TableRow key={item.id}>
              <TableCell>{formatDateTimeValue(item.ts_ms, i18n.language)}</TableCell>
              <TableCell>{item.model || '—'}</TableCell>
              <TableCell>{item.provider || '—'}</TableCell>
              <TableCell>{item.account_label || item.auth_id || '—'}</TableCell>
              <TableCell>
                {item.api_key_label || (item.api_key_id ? fallbackKeyLabel(item.api_key_id) : '—')}
              </TableCell>
              <TableCell>
                <span title={item.status_class || undefined}>
                  {item.failed ? t('usage_page.status_failed') : t('usage_page.status_ok')}
                </span>
              </TableCell>
              <TableCell alignRight>{formatDurationMs(item.latency_ms)}</TableCell>
              <TableCell alignRight>{formatDurationMs(item.ttft_ms)}</TableCell>
              <TableCell alignRight>
                <span title={item.total_tokens.toLocaleString(i18n.language)}>
                  {formatTokenCount(item.total_tokens, i18n.language)}
                </span>
              </TableCell>
              <TableCell alignRight>
                <CostCell item={item} />
              </TableCell>
              <TableCell>
                <RequestCell
                  item={item}
                  onOpen={() => onOpenEvent(item.id)}
                  copied={copiedId === item.id}
                  onCopy={() => onCopyRequest(item)}
                />
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <div className={styles.controls}>
        {hasMore ? (
          <button
            type="button"
            className={styles.resetButton}
            onClick={onLoadMore}
            disabled={loadingMore}
          >
            {t('usage_page.load_more')}
          </button>
        ) : null}
        {totalText ? (
          <span className={styles.sectionNote} role="status">
            {totalText}
          </span>
        ) : null}
        {loadingMore ? (
          <span className={styles.sectionNote}>{t('usage_page.state_loading')}</span>
        ) : null}
      </div>
    </>
  );
}

export function UsageEvents({ query, queryKey, enabled }: UsageSectionProps) {
  const { t, i18n } = useTranslation();
  const [draft, setDraft] = useState('');
  const [q, setQ] = useState('');
  const [cursor, setCursor] = useState<string | null>(null);
  const [pages, setPages] = useState<EventsPage[]>([]);
  const [total, setTotal] = useState<EventsTotal | null>(null);
  const [eventId, setEventId] = useState<number | null>(null);
  const [copiedId, setCopiedId] = useState<number | null>(null);
  const [exportFormat, setExportFormat] = useState<UsageExportFormat>('jsonl');
  const [exporting, setExporting] = useState(false);
  const [exportError, setExportError] = useState(false);

  useEffect(() => {
    const timer = window.setTimeout(() => setQ(draft.trim()), SEARCH_DEBOUNCE_MS);
    return () => window.clearTimeout(timer);
  }, [draft]);

  const resetKey = useMemo(() => eventsResetKey(query, q), [query, q]);

  // A new filter set or search restarts from the first page.
  useEffect(() => {
    setPages([]);
    setTotal(null);
    setCursor(null);
  }, [resetKey]);

  const pageQuery = useMemo(
    () => ({
      ...query,
      limit: PAGE_LIMIT,
      ...(q ? { q } : {}),
      ...(cursor ? { cursor } : {}),
    }),
    [query, q, cursor]
  );
  const page = useUsageQuery<UsageEventsResponse>({
    queryKey: `usage-events|${resetKey}|${cursor ?? 'first'}|${queryKey}`,
    enabled,
    load: (signal) => usageApi.getEvents(pageQuery, { signal }),
  });

  // Latest-wins: only the newest response commits, replacing or appending
  // the page it asked for.
  useEffect(() => {
    if (!page.data) return;
    const data = page.data;
    const id = `${resetKey}|${cursor ?? 'first'}`;
    setPages((previous) => {
      const index = previous.findIndex((entry) => entry.id === id);
      const entry: EventsPage = {
        id,
        items: data.items,
        next: nextUsageCursor(data),
        first: cursor === null,
      };
      if (index === -1) return [...previous, entry];
      const next = [...previous];
      next[index] = entry;
      return next;
    });
    if (cursor === null) {
      setTotal({ estimate: data.total_estimate, exact: data.total_exact !== false });
    }
  }, [page.data, resetKey, cursor]);

  const disabled = page.error ? isUsageDisabledError(page.error) : false;
  const items = pages.flatMap((entry) => entry.items);
  const last = pages.length > 0 ? pages[pages.length - 1] : null;
  const hasMore = last ? last.next !== null : false;
  const shownTotal = total
    ? t(total.exact ? 'usage_page.total_line' : 'usage_page.total_estimate', {
        shown: items.length.toLocaleString(i18n.language),
        total: total.estimate.toLocaleString(i18n.language),
      })
    : null;

  const copyRequestId = (item: UsageEventListItem) => {
    if (!item.request_id) return;
    try {
      const done = navigator.clipboard?.writeText(item.request_id);
      if (done && typeof done.then === 'function') {
        void done
          .then(() => {
            setCopiedId(item.id);
            window.setTimeout(
              () => setCopiedId((current) => (current === item.id ? null : current)),
              1500
            );
          })
          .catch(() => undefined);
      }
    } catch {
      // Clipboard is unavailable; the id stays selectable text.
    }
  };

  const runExport = () => {
    setExporting(true);
    setExportError(false);
    void usageApi
      .exportUsage(eventsExportQuery(query, q, exportFormat))
      .then((file) => downloadBlob(file.blob, file.filename))
      .catch(() => setExportError(true))
      .finally(() => setExporting(false));
  };

  let body: ReactNode;
  if (page.loading && pages.length === 0) {
    body = (
      <Skeleton height={size['240']} role="status" aria-label={t('usage_page.state_loading')} />
    );
  } else if (page.error && pages.length === 0 && !disabled) {
    body = (
      <div className={styles.notice} role="alert">
        <p className={styles.noticeText}>{page.error.message}</p>
        <button
          type="button"
          className={`${styles.resetButton} ${styles.retryButton}`}
          onClick={() => {
            void page.refresh().catch(() => undefined);
          }}
        >
          {t('usage_page.retry')}
        </button>
      </div>
    );
  } else if (items.length === 0) {
    body = disabled ? null : (
      <EmptyState
        title={t('usage_page.events_empty')}
        description={t('usage_page.events_empty_hint')}
      />
    );
  } else {
    body = (
      <>
        <UsageEventsTable
          items={items}
          totalText={shownTotal}
          hasMore={hasMore && last?.next != null}
          loadingMore={page.loading && pages.length > 0}
          copiedId={copiedId}
          onLoadMore={() => {
            if (last?.next) setCursor(last.next);
          }}
          onOpenEvent={setEventId}
          onCopyRequest={copyRequestId}
        />
        <UsageEventDrawer eventId={eventId} onClose={() => setEventId(null)} />
      </>
    );
  }

  return (
    <section className={styles.section} aria-labelledby="usage-events-title">
      <div className={styles.sectionHead}>
        <h2 className={styles.sectionTitle} id="usage-events-title">
          {t('usage_page.events_title')}
        </h2>
        <div className={styles.controls}>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>{t('usage_page.search_label')}</span>
            <input
              className={styles.customInput}
              value={draft}
              onChange={(event) => setDraft(event.target.value)}
              placeholder={t('usage_page.search_placeholder')}
              type="search"
            />
          </label>
          <div className={styles.field}>
            <span className={styles.fieldLabel} id="usage-export-format">
              {t('usage_page.export_label')}
            </span>
            <SegmentedControl<UsageExportFormat>
              options={[
                { value: 'jsonl', label: t('usage_page.export_jsonl') },
                { value: 'csv', label: t('usage_page.export_csv') },
              ]}
              value={exportFormat}
              onChange={setExportFormat}
              ariaLabelledBy="usage-export-format"
            />
          </div>
          <button
            type="button"
            className={styles.resetButton}
            onClick={runExport}
            disabled={!enabled || exporting}
          >
            {exporting ? t('usage_page.state_loading') : t('usage_page.export_label')}
          </button>
        </div>
      </div>
      {exportError && (
        <p className={styles.rangeError} role="alert">
          {t('usage_page.export_failed')}
        </p>
      )}
      {body}
    </section>
  );
}
