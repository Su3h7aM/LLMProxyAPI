/**
 * Event detail drawer: every field of `GET /events/{id}`, the failure
 * summary, the price version used, and the cost source. Opened from the
 * failures section and the events table.
 */

import { useState, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { Sheet } from '@/components/ui/Sheet';
import { Skeleton } from '@/components/ui/Skeleton';
import { size } from '@/design-system';
import { usageApi } from '@/services/api/usage';
import { useUsageQuery } from '@/features/usage/hooks/useUsageQuery';
import {
  formatDurationMs,
  formatMicroUsd,
  formatMicroUsdExact,
  formatTokenCount,
} from '@/features/usage/logic/format';
import { formatDateTimeValue } from '@/utils/format';
import { fallbackKeyLabel } from '@/features/usage/logic/breakdown';
import type { UsageEventDetail } from '@/types/usageStats';
import styles from '../UsagePage.module.scss';

function Field({ label, value, title }: { label: string; value: ReactNode; title?: string }) {
  return (
    <>
      <dt className={styles.defTerm}>{label}</dt>
      <dd className={styles.defValue} title={title}>
        {value}
      </dd>
    </>
  );
}

function EventDetailBody({ event }: { event: UsageEventDetail }) {
  const { t, i18n } = useTranslation();
  const cost =
    event.cost_micro_usd === null
      ? '—'
      : formatMicroUsd(event.cost_micro_usd, i18n.language);
  const costTitle =
    event.cost_micro_usd === null
      ? undefined
      : formatMicroUsdExact(event.cost_micro_usd, i18n.language);
  return (
    <dl className={styles.defs}>
      <Field label={t('usage_page.col_time')} value={formatDateTimeValue(event.ts_ms, i18n.language)} />
      <Field label={t('usage_page.col_request')} value={event.request_id} />
      <Field label={t('usage_page.detail_trace')} value={event.trace_id || '—'} />
      <Field label={t('usage_page.detail_session')} value={event.session_id || '—'} />
      <Field label={t('usage_page.col_provider')} value={event.provider || '—'} />
      <Field label={t('usage_page.detail_executor')} value={event.executor || '—'} />
      <Field label={t('usage_page.col_model')} value={event.model || '—'} />
      <Field label={t('usage_page.detail_alias')} value={event.alias || '—'} />
      <Field
        label={t('usage_page.col_credential')}
        value={event.account_label || event.auth_id || '—'}
        title={event.auth_id || undefined}
      />
      <Field
        label={t('usage_page.col_key')}
        value={event.api_key_label || (event.api_key_id ? fallbackKeyLabel(event.api_key_id) : '—')}
      />
      <Field
        label={t('usage_page.filter_stream')}
        value={t(event.stream ? 'usage_page.stream_on' : 'usage_page.stream_off')}
      />
      <Field
        label={t('usage_page.col_status')}
        value={`${event.status_class || '—'}${event.failed ? ` · ${t('usage_page.status_failed')}` : ''}`}
      />
      <Field label={t('usage_page.col_code')} value={event.fail_code || '—'} />
      <Field label={t('usage_page.col_latency')} value={formatDurationMs(event.latency_ms)} />
      <Field label={t('usage_page.col_ttft')} value={formatDurationMs(event.ttft_ms)} />
      <Field
        label={t('usage_page.col_tokens')}
        value={formatTokenCount(event.total_tokens, i18n.language)}
        title={`${t('usage_page.tokens_input')} ${event.input_tokens.toLocaleString(i18n.language)} · ${t('usage_page.tokens_output')} ${event.output_tokens.toLocaleString(i18n.language)} · ${t('usage_page.tokens_reasoning')} ${event.reasoning_tokens.toLocaleString(i18n.language)} · ${t('usage_page.tokens_cache_read')} ${event.cache_read_tokens.toLocaleString(i18n.language)} · ${t('usage_page.tokens_cache_write')} ${event.cache_write_tokens.toLocaleString(i18n.language)}`}
      />
      <Field label={t('usage_page.detail_service_tier')} value={event.service_tier || '—'} />
      <Field
        label={t('usage_page.detail_reasoning_effort')}
        value={event.reasoning_effort || '—'}
      />
      <Field label={t('usage_page.col_cost')} value={cost} title={costTitle} />
      <Field label={t('usage_page.detail_cost_source')} value={event.cost_source} />
      {event.fail_summary && (
        <Field label={t('usage_page.detail_fail_summary')} value={event.fail_summary} />
      )}
    </dl>
  );
}

function EventPrice({ event }: { event: UsageEventDetail }) {
  const { t, i18n } = useTranslation();
  if (!event.price) {
    return <p className={styles.sectionNote}>{t('usage_page.detail_no_price')}</p>;
  }
  const price = event.price;
  // Price-book rates are integer micro-USD per 1M tokens.
  const perMillion = (rate: number) => `${formatMicroUsd(rate, i18n.language)} / 1M`;
  return (
    <>
      <h3 className={styles.subTitle}>{t('usage_page.detail_price')}</h3>
      <dl className={styles.defs}>
        <Field label={t('usage_page.col_provider')} value={price.provider} />
        <Field label={t('usage_page.col_model')} value={price.model} />
        <Field
          label={t('usage_page.detail_valid_from')}
          value={formatDateTimeValue(price.valid_from_ms, i18n.language)}
        />
        <Field
          label={t('usage_page.tokens_input')}
          value={perMillion(price.input)}
        />
        <Field
          label={t('usage_page.tokens_output')}
          value={perMillion(price.output)}
        />
        <Field
          label={t('usage_page.tokens_cache_read')}
          value={perMillion(price.cache_read)}
        />
        <Field
          label={t('usage_page.tokens_cache_write')}
          value={perMillion(price.cache_write)}
        />
        <Field label={t('usage_page.detail_source')} value={price.source} />
      </dl>
    </>
  );
}

export function UsageEventDrawer({
  eventId,
  onClose,
}: {
  eventId: number | null;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);
  const detail = useUsageQuery({
    queryKey: `usage-event|${eventId ?? 'none'}`,
    enabled: eventId !== null,
    load: (signal) => usageApi.getEvent(eventId as number, { signal }),
  });

  const copyId = async () => {
    const requestId = detail.data?.request_id;
    if (!requestId) return;
    try {
      await navigator.clipboard.writeText(requestId);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      // Clipboard is unavailable; the id stays selectable text.
    }
  };

  return (
    <Sheet
      open={eventId !== null}
      onClose={onClose}
      size="lg"
      title={t('usage_page.detail_title')}
      footer={
        detail.data ? (
          <button type="button" className={styles.resetButton} onClick={() => void copyId()}>
            {copied ? t('usage_page.copied') : t('usage_page.copy_id')}
          </button>
        ) : undefined
      }
    >
      {detail.loading && !detail.data ? (
        <Skeleton height={size['240']} role="status" aria-label={t('usage_page.state_loading')} />
      ) : detail.error ? (
        <div className={styles.notice} role="alert">
          <p className={styles.noticeText}>{detail.error.message}</p>
          <button
            type="button"
            className={`${styles.resetButton} ${styles.retryButton}`}
            onClick={() => {
              void detail.refresh().catch(() => undefined);
            }}
          >
            {t('usage_page.retry')}
          </button>
        </div>
      ) : detail.data ? (
        <>
          <EventDetailBody event={detail.data} />
          <EventPrice event={detail.data} />
        </>
      ) : null}
    </Sheet>
  );
}
