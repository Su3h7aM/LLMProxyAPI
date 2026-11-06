/**
 * OpenCode Go subscription data layer: rolling (5 hours), weekly and
 * monthly allowance windows at the fixed Go usage route, asked with the
 * workspace member's Go key through the backend api-call proxy.
 * React-free / SCSS-free.
 */

import type { TFunction } from 'i18next';
import type { AuthFileItem, OpencodeGoQuotaState, OpencodeGoQuotaRow } from '@/types';
import { apiCallApi, getApiCallErrorMessage } from '@/services/api';
import {
  OPENCODE_GO_USAGE_URL,
  OPENCODE_GO_REQUEST_HEADERS,
  createStatusError,
  isDisabledAuthFile,
  isOpencodeGoFile,
} from '@/utils/quota';
import { normalizeAuthIndex } from '@/utils/authIndex';
import type { QuotaProviderData } from '../types';

const OPENCODE_GO_REQUEST_TIMEOUT_MS = 15000;

const SESSION_BASE62 = '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz';

// Session id laid out as OpenCode lays one out: ses_, 12 hex digits of
// timestamp plus a per-millisecond count, then 14 random base62 chars. The
// gateway routes nothing that names no session.
let sessionCounter = 0;
let sessionLastMs = 0;

export function buildOpencodeGoSessionId(now = Date.now()): string {
  if (now !== sessionLastMs) {
    sessionLastMs = now;
    sessionCounter = 0;
  }
  sessionCounter += 1;
  const current = (BigInt(now) * 0x1000n + BigInt(sessionCounter)) ^ ((1n << 64n) - 1n);
  const hex = current.toString(16).padStart(12, '0').slice(-12);
  const bytes = new Uint8Array(14);
  crypto.getRandomValues(bytes);
  let tail = '';
  for (const b of bytes) tail += SESSION_BASE62[b % SESSION_BASE62.length];
  return `ses_${hex}${tail}`;
}

interface OpencodeGoWindow {
  status?: unknown;
  percent?: unknown;
  resetsAt?: unknown;
}

const toRecord = (value: unknown): Record<string, unknown> | null =>
  value && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : null;

const toNumber = (value: unknown): number | null => {
  if (typeof value === 'number' && Number.isFinite(value)) return value;
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) return parsed;
  }
  return null;
};

const toIsoString = (value: unknown): string | null => {
  if (typeof value !== 'string' || value.trim() === '') return null;
  const ms = Date.parse(value);
  return Number.isFinite(ms) ? value : null;
};

export function parseOpencodeGoUsage(body: unknown): OpencodeGoQuotaRow[] {
  const usage = toRecord(toRecord(body)?.usage);
  if (!usage) return [];
  const defs = [
    { key: 'rolling', id: 'rolling', labelKey: 'opencode_go_quota.window_rolling', periodHours: 5 },
    { key: 'weekly', id: 'weekly', labelKey: 'opencode_go_quota.window_weekly', periodHours: 24 * 7 },
    { key: 'monthly', id: 'monthly', labelKey: 'opencode_go_quota.window_monthly', periodHours: null },
  ] as const;
  const rows: OpencodeGoQuotaRow[] = [];
  for (const def of defs) {
    const window = toRecord(usage[def.key]) as OpencodeGoWindow | null;
    if (!window) continue;
    const percent = toNumber(window.percent);
    if (percent === null) continue;
    const rateLimited = String(window.status ?? '').trim().toLowerCase() === 'rate-limited';
    const used = rateLimited ? 100 : Math.max(0, Math.min(100, percent));
    const resetsAt = toIsoString(window.resetsAt);
    rows.push({
      id: def.id,
      labelKey: def.labelKey,
      used,
      limit: 100,
      // A 0% window carries a reset time nothing happens at; leave it out.
      resetAtMs: resetsAt && used > 0 ? Date.parse(resetsAt) : null,
      periodHours: def.periodHours,
    });
  }
  return rows;
}

const readBody = (result: { statusCode: number; body?: unknown; bodyText?: string }): unknown => {
  if (result.statusCode < 200 || result.statusCode >= 300) return null;
  const body = result.body ?? result.bodyText;
  if (typeof body === 'string') {
    try {
      return JSON.parse(body) as unknown;
    } catch {
      return null;
    }
  }
  return body ?? null;
};

const fetchOpencodeGoQuota = async (
  file: AuthFileItem,
  t: TFunction
): Promise<{ planName: string; rows: OpencodeGoQuotaRow[] }> => {
  const authIndex = normalizeAuthIndex(file['auth_index'] ?? file.authIndex);
  if (!authIndex) {
    throw new Error(t('opencode_go_quota.missing_auth_index'));
  }
  const session = buildOpencodeGoSessionId();
  const result = await apiCallApi.request(
    {
      authIndex,
      method: 'GET',
      url: OPENCODE_GO_USAGE_URL,
      header: {
        ...OPENCODE_GO_REQUEST_HEADERS,
        'x-opencode-session': session,
        'x-opencode-session-id': session,
      },
    },
    { timeout: OPENCODE_GO_REQUEST_TIMEOUT_MS }
  );
  if (result.statusCode < 200 || result.statusCode >= 300) {
    throw createStatusError(getApiCallErrorMessage(result), result.statusCode);
  }
  const rows = parseOpencodeGoUsage(readBody(result));
  if (rows.length === 0) {
    throw new Error(t('opencode_go_quota.empty_data'));
  }
  return { planName: 'OpenCode Go', rows };
};

export const OPENCODE_GO_CONFIG: QuotaProviderData<OpencodeGoQuotaState, { planName: string; rows: OpencodeGoQuotaRow[] }> = {
  type: 'opencode-go',
  i18nPrefix: 'opencode_go_quota',
  filterFn: (file) => isOpencodeGoFile(file) && !isDisabledAuthFile(file),
  fetchQuota: fetchOpencodeGoQuota,
  storeSelector: (state) => state.opencodeGoQuota,
  storeSetter: 'setOpencodeGoQuota',
  buildLoadingState: () => ({ status: 'loading', rows: [] }),
  buildSuccessState: (data) => ({ status: 'success', planName: data.planName, rows: data.rows }),
  buildErrorState: (message, status) => ({
    status: 'error',
    rows: [],
    error: message,
    errorStatus: status,
  }),
};
