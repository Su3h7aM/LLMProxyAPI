import { describe, expect, test } from 'bun:test';
import {
  DEFAULT_PRICE_FILTERS,
  PRICE_RANGE_MAX_MS,
  emptyPriceForm,
  formatDateTimeInput,
  formatPriceRate,
  historyNewestFirst,
  microFromUsdRate,
  parseDateTimeInput,
  priceListQuery,
  pricePageCount,
  priceQueryKey,
  priceSourceTone,
  priceToFormValues,
  rateInputValue,
  readPriceSyncConfig,
  unpricedBody,
  usdFromMicroRate,
  validateCorrectionForm,
  validatePriceForm,
} from '@/features/usage/logic/prices';
import type { UsagePriceItem } from '@/types/usageStats';

const NOW = Date.parse('2026-11-12T12:00:00.000Z');

const price = (overrides: Partial<UsagePriceItem> = {}): UsagePriceItem => ({
  id: 1,
  provider: 'claude',
  model: 'claude-sonnet-5-5',
  valid_from_ms: 0,
  valid_to_ms: null,
  input: 3_000_000,
  output: 15_000_000,
  cache_read: 300_000,
  cache_write: 3_750_000,
  reasoning: null,
  source: 'models.dev',
  synced_at_ms: NOW,
  note: '',
  tiers: [],
  ...overrides,
});

describe('price rate conversion', () => {
  test('list micro-USD per 1M becomes USD per 1M', () => {
    expect(usdFromMicroRate(3_000_000)).toBe(3);
    expect(usdFromMicroRate(75_000)).toBe(0.075);
    expect(usdFromMicroRate(0)).toBe(0);
  });

  test('USD per 1M becomes micro-USD per 1M with half-up rounding', () => {
    expect(microFromUsdRate(3)).toBe(3_000_000);
    expect(microFromUsdRate(0.075)).toBe(75_000);
    expect(microFromUsdRate(0.0000005)).toBe(1);
    expect(microFromUsdRate(0.0000004)).toBe(0);
  });

  test('a rate keeps its small digits and its currency style', () => {
    expect(formatPriceRate(3_000_000, 'en')).toBe('$3.00');
    expect(formatPriceRate(75_000, 'en')).toBe('$0.075');
    expect(formatPriceRate(0, 'en')).toBe('$0.00');
    expect(formatPriceRate(Number.NaN, 'en')).toBe('—');
  });

  test('a list rate becomes a plain form value', () => {
    expect(rateInputValue(3_000_000)).toBe('3');
    expect(rateInputValue(75_000)).toBe('0.075');
    expect(rateInputValue(Number.NaN)).toBe('');
  });
});

describe('price filters and paging', () => {
  test('empty filters are omitted and the page becomes an offset', () => {
    expect(priceListQuery(DEFAULT_PRICE_FILTERS, 1)).toEqual({ limit: 50, offset: 0 });
    expect(priceListQuery({ provider: ' claude ', source: '', q: ' son ' }, 3)).toEqual({
      provider: 'claude',
      q: 'son',
      limit: 50,
      offset: 100,
    });
  });

  test('an unusable page falls back to the first page', () => {
    expect(priceListQuery(DEFAULT_PRICE_FILTERS, 0).offset).toBe(0);
    expect(priceListQuery(DEFAULT_PRICE_FILTERS, Number.NaN).offset).toBe(0);
  });

  test('the query key follows every filter change', () => {
    expect(priceQueryKey(DEFAULT_PRICE_FILTERS, 1)).toBe('|||0');
    expect(priceQueryKey({ provider: 'claude', source: 'manual', q: 's' }, 2)).toBe(
      'claude|manual|s|50'
    );
  });

  test('pages cover the total without an empty last page', () => {
    expect(pricePageCount(0)).toBe(1);
    expect(pricePageCount(50)).toBe(1);
    expect(pricePageCount(51)).toBe(2);
    expect(pricePageCount(Number.NaN)).toBe(1);
  });
});

describe('price history order', () => {
  test('versions are newest first without changing the source list', () => {
    const versions = [
      price({ id: 1, valid_from_ms: 10 }),
      price({ id: 3, valid_from_ms: 30 }),
      price({ id: 2, valid_from_ms: 20 }),
    ];
    expect(historyNewestFirst(versions).map((item) => item.id)).toEqual([3, 2, 1]);
    expect(versions.map((item) => item.id)).toEqual([1, 3, 2]);
  });
});

describe('price form', () => {
  test('a new form defaults the effective time to now', () => {
    const values = emptyPriceForm({ provider: 'claude', model: 'sonnet' }, NOW);
    expect(values.provider).toBe('claude');
    expect(values.model).toBe('sonnet');
    expect(parseDateTimeInput(values.effectiveFrom)).toBe(
      Date.parse(formatDateTimeInput(NOW))
    );
  });

  test('edit prefill uses USD rates and clears an absent reasoning rate', () => {
    const values = priceToFormValues(price(), NOW);
    expect(values.input).toBe('3');
    expect(values.output).toBe('15');
    expect(values.cacheRead).toBe('0.3');
    expect(values.cacheWrite).toBe('3.75');
    expect(values.reasoning).toBe('');
    expect(parseDateTimeInput(values.effectiveFrom)).toBe(
      Date.parse(formatDateTimeInput(NOW))
    );
  });

  test('input and output are required numbers that must not be negative', () => {
    const missing = validatePriceForm(emptyPriceForm({}, NOW), NOW);
    expect(missing.input).toBeNull();
    expect(missing.errors).toMatchObject({
      provider: 'required',
      model: 'required',
      input: 'required',
      output: 'required',
    });

    const negative = validatePriceForm(
      { ...emptyPriceForm({ provider: 'claude', model: 'm' }, NOW), input: '-1', output: '2' },
      NOW
    );
    expect(negative.errors.input).toBe('negative');

    const text = validatePriceForm(
      { ...emptyPriceForm({ provider: 'claude', model: 'm' }, NOW), input: 'abc', output: '2' },
      NOW
    );
    expect(text.errors.input).toBe('number');
  });

  test('a valid form keeps rates in USD per 1M and omits empty optional fields', () => {
    const result = validatePriceForm(
      {
        ...emptyPriceForm({ provider: ' claude ', model: ' sonnet ' }, NOW),
        input: '3',
        output: '15',
        cacheRead: '0.3',
        reasoning: '',
        note: ' negotiated ',
      },
      NOW
    );
    expect(result.errors).toEqual({});
    expect(result.input).toEqual({
      provider: 'claude',
      model: 'sonnet',
      input: 3,
      output: 15,
      cache_read: 0.3,
      effective_from_ms: NOW,
      note: 'negotiated',
    });
  });

  test('an unreadable effective time is a date error; an empty one means now', () => {
    const base = {
      ...emptyPriceForm({ provider: 'claude', model: 'm' }, NOW),
      input: '1',
      output: '2',
    };
    expect(validatePriceForm({ ...base, effectiveFrom: 'not a date' }, NOW).errors.effectiveFrom).toBe(
      'date'
    );
    expect(validatePriceForm({ ...base, effectiveFrom: '' }, NOW).input?.effective_from_ms).toBe(
      NOW
    );
  });
});

describe('audited correction form', () => {
  const word = 'correct';
  const base = {
    from: '2026-01-01T00:00',
    to: '2026-01-02T00:00',
    model: 'claude-sonnet-5-5',
    reason: 'wrong price',
    confirmation: word,
  };

  test('a valid form asks for confirmation and the reason', () => {
    const result = validateCorrectionForm(base, word);
    expect(result.errors).toEqual([]);
    expect(result.input).toMatchObject({
      confirm: true,
      reason: 'wrong price',
      model: 'claude-sonnet-5-5',
    });
    expect(result.input?.from_ms).toBe(parseDateTimeInput(base.from));
  });

  test('the typed word must match the locale word', () => {
    const result = validateCorrectionForm({ ...base, confirmation: 'Correct ' }, word);
    expect(result.errors).toEqual(['confirm_mismatch']);
    expect(result.input).toBeNull();
  });

  test('the range is required, ordered, and capped at 366 days', () => {
    expect(validateCorrectionForm({ ...base, from: '' }, word).errors).toEqual(
      expect.arrayContaining(['range_required'])
    );
    expect(
      validateCorrectionForm({ ...base, from: '2026-01-02T00:00', to: '2026-01-01T00:00' }, word)
        .errors
    ).toContain('range_order');
    const from = Date.parse('2026-01-01T00:00');
    const tooLong = {
      ...base,
      from: formatDateTimeInput(from),
      to: formatDateTimeInput(from + PRICE_RANGE_MAX_MS + 60_000),
    };
    expect(validateCorrectionForm(tooLong, word).errors).toContain('range_too_long');
    expect(
      validateCorrectionForm({ ...base, reason: '  ' }, word).errors
    ).toContain('reason_required');
  });
});

describe('price unpriced form', () => {
  test('no range and no model sends an empty body', () => {
    expect(unpricedBody({ from: '', to: '', model: '' })).toEqual({ body: {}, errors: [] });
  });

  test('model alone is allowed', () => {
    expect(unpricedBody({ from: '', to: '', model: ' sonnet ' })).toEqual({
      body: { model: 'sonnet' },
      errors: [],
    });
  });

  test('one range end is an error; both must be ordered and within the cap', () => {
    expect(unpricedBody({ from: '2026-01-01T00:00', to: '', model: '' }).errors).toEqual([
      'range_pair',
    ]);
    const from = Date.parse('2026-01-01T00:00');
    expect(
      unpricedBody({
        from: formatDateTimeInput(from),
        to: formatDateTimeInput(from),
        model: '',
      }).errors
    ).toEqual(['range_order']);
    expect(
      unpricedBody({
        from: formatDateTimeInput(from),
        to: formatDateTimeInput(from + PRICE_RANGE_MAX_MS + 60_000),
        model: '',
      }).errors
    ).toEqual(['range_too_long']);
  });
});

describe('price sync config', () => {
  test('reads the read-only price-sync node and fills its defaults', () => {
    const config = {
      raw: {
        observability: {
          'usage-db': {
            'price-sync': { enabled: false, url: ' http://localhost/prices ', 'interval-hours': 6 },
          },
        },
      },
    } as never;
    expect(readPriceSyncConfig(config)).toEqual({
      enabled: false,
      url: 'http://localhost/prices',
      intervalHours: 6,
    });
    expect(
      readPriceSyncConfig({ raw: { observability: { 'usage-db': { 'price-sync': {} } } } } as never)
    ).toEqual({ enabled: true, url: '', intervalHours: 24 });
    expect(readPriceSyncConfig(null)).toBeNull();
  });

  test('a source badge tone separates manual, synced, embedded, and unknown', () => {
    expect(priceSourceTone('manual')).toBe('manual');
    expect(priceSourceTone('models.dev')).toBe('synced');
    expect(priceSourceTone('embedded')).toBe('embedded');
    expect(priceSourceTone('other')).toBe('unknown');
  });
});
