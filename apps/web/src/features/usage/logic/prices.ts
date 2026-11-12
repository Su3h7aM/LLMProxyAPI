/**
 * Price book logic.
 *
 * The list route returns integer micro-USD per 1M tokens; the write route
 * takes USD per 1M tokens. Both conversions live here, with the same
 * half-up rounding the backend uses. The form validators return error codes:
 * the component turns each code into a localized message.
 */

import type { Config } from '@/types/config';
import type {
  UsagePriceCorrectionInput,
  UsagePriceInput,
  UsagePriceItem,
  UsagePriceListQuery,
} from '@/types/usageStats';

const MICRO_USD_PER_USD = 1_000_000;

/** One list page. The backend caps a price page at 500 rows. */
export const PRICE_PAGE_SIZE = 50;

/** The correction route rejects a range longer than 366 days. */
export const PRICE_RANGE_MAX_MS = 366 * 24 * 60 * 60 * 1000;

export const PRICE_SOURCES = ['models.dev', 'manual', 'embedded'] as const;

/** List rate (micro-USD per 1M tokens) as USD per 1M tokens. */
export function usdFromMicroRate(micro: number): number {
  return micro / MICRO_USD_PER_USD;
}

/** USD per 1M tokens as the integer micro-USD per 1M tokens of the wire. */
export function microFromUsdRate(usd: number): number {
  return Math.round(usd * MICRO_USD_PER_USD);
}

/** A micro-USD rate as USD per 1M tokens, with enough digits for cheap models. */
export function formatPriceRate(micro: number, locale = 'en'): string {
  if (!Number.isFinite(micro)) return '—';
  return new Intl.NumberFormat(locale, {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 6,
  }).format(usdFromMicroRate(micro));
}

export interface PriceFilters {
  provider: string;
  source: string;
  q: string;
}

export const DEFAULT_PRICE_FILTERS: Readonly<PriceFilters> = Object.freeze({
  provider: '',
  source: '',
  q: '',
});

/** Server query for one page of current prices. Empty filters are omitted. */
export function priceListQuery(filters: PriceFilters, page: number): UsagePriceListQuery {
  const provider = filters.provider.trim();
  const source = filters.source.trim();
  const q = filters.q.trim();
  const safePage = Number.isFinite(page) ? Math.max(1, Math.floor(page)) : 1;
  return {
    ...(provider ? { provider } : {}),
    ...(source ? { source } : {}),
    ...(q ? { q } : {}),
    limit: PRICE_PAGE_SIZE,
    offset: (safePage - 1) * PRICE_PAGE_SIZE,
  };
}

/** Stable identity of one price request; a new key aborts the old request. */
export function priceQueryKey(filters: PriceFilters, page: number): string {
  const query = priceListQuery(filters, page);
  return [query.provider ?? '', query.source ?? '', query.q ?? '', query.offset ?? 0].join('|');
}

/** Pages needed for a total; at least one, so the table always has a page. */
export function pricePageCount(total: number): number {
  if (!Number.isFinite(total) || total <= 0) return 1;
  return Math.max(1, Math.ceil(total / PRICE_PAGE_SIZE));
}

/** The history route answers oldest first; the sheet shows newest first. */
export function historyNewestFirst(versions: readonly UsagePriceItem[]): UsagePriceItem[] {
  return [...versions].sort((left, right) => right.valid_from_ms - left.valid_from_ms);
}

export interface PriceFormValues {
  provider: string;
  model: string;
  input: string;
  output: string;
  cacheRead: string;
  cacheWrite: string;
  reasoning: string;
  effectiveFrom: string;
  note: string;
}

export type PriceFormField =
  | 'provider'
  | 'model'
  | 'input'
  | 'output'
  | 'cacheRead'
  | 'cacheWrite'
  | 'reasoning'
  | 'effectiveFrom';

export type PriceFormErrorCode = 'required' | 'number' | 'negative' | 'date';

export type PriceFormErrors = Partial<Record<PriceFormField, PriceFormErrorCode>>;

export interface PriceFormResult {
  input: UsagePriceInput | null;
  errors: PriceFormErrors;
}

const pad = (value: number): string => String(value).padStart(2, '0');

/** Local `datetime-local` value for an instant. */
export function formatDateTimeInput(ms: number): string {
  const date = new Date(ms);
  if (Number.isNaN(date.getTime())) return '';
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(
    date.getHours()
  )}:${pad(date.getMinutes())}`;
}

/** Local `datetime-local` text as epoch milliseconds; null when unreadable. */
export function parseDateTimeInput(value: string): number | null {
  const text = value.trim();
  if (!text) return null;
  const ms = new Date(text).getTime();
  return Number.isFinite(ms) ? ms : null;
}

/** USD string of a list rate for a form field; whole rates have no decimals. */
export function rateInputValue(micro: number): string {
  return Number.isFinite(micro) ? String(usdFromMicroRate(micro)) : '';
}

export function emptyPriceForm(
  seed: { provider?: string; model?: string } = {},
  nowMs = Date.now()
): PriceFormValues {
  return {
    provider: seed.provider ?? '',
    model: seed.model ?? '',
    input: '',
    output: '',
    cacheRead: '',
    cacheWrite: '',
    reasoning: '',
    effectiveFrom: formatDateTimeInput(nowMs),
    note: '',
  };
}

/** Prefill the edit form from the current version. Effective from stays "now". */
export function priceToFormValues(price: UsagePriceItem, nowMs = Date.now()): PriceFormValues {
  return {
    provider: price.provider,
    model: price.model,
    input: rateInputValue(price.input),
    output: rateInputValue(price.output),
    cacheRead: rateInputValue(price.cache_read),
    cacheWrite: rateInputValue(price.cache_write),
    reasoning: price.reasoning === null ? '' : rateInputValue(price.reasoning),
    effectiveFrom: formatDateTimeInput(nowMs),
    note: price.note,
  };
}

const rateValue = (
  value: string,
  field: PriceFormField,
  errors: PriceFormErrors,
  required: boolean
): number | undefined => {
  const text = value.trim();
  if (!text) {
    if (required) errors[field] = 'required';
    return undefined;
  }
  const parsed = Number(text);
  if (!Number.isFinite(parsed)) {
    errors[field] = 'number';
    return undefined;
  }
  if (parsed < 0) {
    errors[field] = 'negative';
    return undefined;
  }
  return parsed;
};

/** Validate the form and build the PUT body (rates stay USD per 1M tokens). */
export function validatePriceForm(values: PriceFormValues, nowMs = Date.now()): PriceFormResult {
  const errors: PriceFormErrors = {};
  const provider = values.provider.trim();
  const model = values.model.trim();
  if (!provider) errors.provider = 'required';
  if (!model) errors.model = 'required';

  const input = rateValue(values.input, 'input', errors, true);
  const output = rateValue(values.output, 'output', errors, true);
  const cacheRead = rateValue(values.cacheRead, 'cacheRead', errors, false);
  const cacheWrite = rateValue(values.cacheWrite, 'cacheWrite', errors, false);
  const reasoning = rateValue(values.reasoning, 'reasoning', errors, false);

  const effectiveMs = values.effectiveFrom.trim()
    ? parseDateTimeInput(values.effectiveFrom)
    : nowMs;
  if (effectiveMs === null) errors.effectiveFrom = 'date';

  if (Object.keys(errors).length > 0 || input === undefined || output === undefined) {
    return { input: null, errors };
  }
  return {
    input: {
      provider,
      model,
      input,
      output,
      ...(cacheRead !== undefined ? { cache_read: cacheRead } : {}),
      ...(cacheWrite !== undefined ? { cache_write: cacheWrite } : {}),
      ...(reasoning !== undefined ? { reasoning } : {}),
      ...(effectiveMs !== null ? { effective_from_ms: effectiveMs } : {}),
      ...(values.note.trim() ? { note: values.note.trim() } : {}),
    },
    errors,
  };
}

export type CorrectionErrorCode =
  | 'range_required'
  | 'range_order'
  | 'range_too_long'
  | 'reason_required'
  | 'confirm_mismatch';

export interface CorrectionFormValues {
  from: string;
  to: string;
  model: string;
  reason: string;
  confirmation: string;
}

export interface CorrectionFormResult {
  input: UsagePriceCorrectionInput | null;
  errors: CorrectionErrorCode[];
}

/** Validate the audited correction. The typed word must match the locale's word. */
export function validateCorrectionForm(
  values: CorrectionFormValues,
  expectedWord: string
): CorrectionFormResult {
  const errors: CorrectionErrorCode[] = [];
  const from = parseDateTimeInput(values.from);
  const to = parseDateTimeInput(values.to);
  if (from === null || to === null) {
    errors.push('range_required');
  } else if (to <= from) {
    errors.push('range_order');
  } else if (to - from > PRICE_RANGE_MAX_MS) {
    errors.push('range_too_long');
  }
  const reason = values.reason.trim();
  if (!reason) errors.push('reason_required');
  if (values.confirmation.trim() !== expectedWord.trim()) errors.push('confirm_mismatch');
  if (errors.length > 0 || from === null || to === null) {
    return { input: null, errors };
  }
  const model = values.model.trim();
  return {
    input: { from_ms: from, to_ms: to, confirm: true, reason, ...(model ? { model } : {}) },
    errors,
  };
}

export interface UnpricedFormValues {
  from: string;
  to: string;
  model: string;
}

export type UnpricedErrorCode = 'range_pair' | 'range_order' | 'range_too_long';

export interface UnpricedFormResult {
  body: { from_ms?: number; to_ms?: number; model?: string } | null;
  errors: UnpricedErrorCode[];
}

/** Optional range: both ends or neither. Model is optional. */
export function unpricedBody(values: UnpricedFormValues): UnpricedFormResult {
  const errors: UnpricedErrorCode[] = [];
  const hasFrom = values.from.trim().length > 0;
  const hasTo = values.to.trim().length > 0;
  if (hasFrom !== hasTo) {
    errors.push('range_pair');
    return { body: null, errors };
  }
  const model = values.model.trim();
  if (!hasFrom) return { body: model ? { model } : {}, errors };
  const from = parseDateTimeInput(values.from);
  const to = parseDateTimeInput(values.to);
  if (from === null || to === null) {
    errors.push('range_pair');
    return { body: null, errors };
  }
  if (to <= from) errors.push('range_order');
  else if (to - from > PRICE_RANGE_MAX_MS) errors.push('range_too_long');
  if (errors.length > 0) return { body: null, errors };
  return { body: { from_ms: from, to_ms: to, ...(model ? { model } : {}) }, errors };
}

export interface PriceSyncConfigState {
  enabled: boolean;
  url: string;
  intervalHours: number;
}

const recordAt = (value: unknown, key: string): unknown =>
  value && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)[key]
    : undefined;

/** Read-only `observability.usage-db.price-sync` state; null when absent. */
export function readPriceSyncConfig(config: Config | null): PriceSyncConfigState | null {
  const observability = config?.raw?.observability;
  const usageDb = recordAt(observability, 'usage-db');
  const priceSync = recordAt(usageDb, 'price-sync');
  if (!priceSync || typeof priceSync !== 'object' || Array.isArray(priceSync)) return null;
  const enabled = recordAt(priceSync, 'enabled');
  const url = recordAt(priceSync, 'url');
  const intervalHours = recordAt(priceSync, 'interval-hours');
  return {
    enabled: typeof enabled === 'boolean' ? enabled : true,
    url: typeof url === 'string' ? url.trim() : '',
    intervalHours: typeof intervalHours === 'number' ? intervalHours : 24,
  };
}

/** Source badge tone; manual wins over sync, embedded is the offline fallback. */
export function priceSourceTone(source: string): 'manual' | 'synced' | 'embedded' | 'unknown' {
  if (source === 'manual') return 'manual';
  if (source === 'models.dev') return 'synced';
  if (source === 'embedded') return 'embedded';
  return 'unknown';
}
