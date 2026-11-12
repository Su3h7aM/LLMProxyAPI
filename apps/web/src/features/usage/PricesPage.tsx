import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { PageHeader } from '@/components/ui/PageHeader';
import { Button } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import { RefreshButton } from '@/components/ui/RefreshButton';
import { CorrectionPanel } from '@/features/usage/components/CorrectionPanel';
import { PriceFormSheet } from '@/features/usage/components/PriceFormSheet';
import { PriceHistorySheet } from '@/features/usage/components/PriceHistorySheet';
import { PriceSyncPanel } from '@/features/usage/components/PriceSyncPanel';
import { PriceUnpricedPanel } from '@/features/usage/components/PriceUnpricedPanel';
import { PricesTable } from '@/features/usage/components/PricesTable';
import { UnmatchedPrices } from '@/features/usage/components/UnmatchedPrices';
import { useUsageQuery } from '@/features/usage/hooks/useUsageQuery';
import {
  DEFAULT_PRICE_FILTERS,
  emptyPriceForm,
  priceListQuery,
  pricePageCount,
  priceQueryKey,
  priceToFormValues,
  readPriceSyncConfig,
  validatePriceForm,
  type PriceFilters,
  type PriceFormValues,
} from '@/features/usage/logic/prices';
import { useHeaderRefresh } from '@/hooks/useHeaderRefresh';
import { isUsageDisabledError, usageApi } from '@/services/api/usage';
import { useConfigStore, useNotificationStore } from '@/stores';
import type {
  UsagePriceCorrectionInput,
  UsagePriceCorrectionResponse,
  UsagePriceItem,
  UsagePriceListResponse,
  UsagePriceStatusResponse,
  UsagePriceSyncResponse,
  UsagePriceUnpricedResponse,
} from '@/types/usageStats';
import styles from './PricesPage.module.scss';

type FormSheetState =
  | { mode: 'create'; initial: PriceFormValues }
  | { mode: 'edit'; initial: PriceFormValues }
  | null;

interface HistoryState {
  provider: string;
  model: string;
}

const errorMessage = (error: unknown): string =>
  error instanceof Error ? error.message : String(error);

export function PricesPage() {
  const { t } = useTranslation();
  const showNotification = useNotificationStore((state) => state.showNotification);
  const showConfirmation = useNotificationStore((state) => state.showConfirmation);
  const config = useConfigStore((state) => state.config);
  const fetchConfig = useConfigStore((state) => state.fetchConfig);

  const [filters, setFilters] = useState<PriceFilters>({ ...DEFAULT_PRICE_FILTERS });
  const [page, setPage] = useState(1);
  const [form, setForm] = useState<FormSheetState>(null);
  const [history, setHistory] = useState<HistoryState | null>(null);
  const [syncing, setSyncing] = useState(false);
  const [syncResult, setSyncResult] = useState<UsagePriceSyncResponse | null>(null);
  const [syncError, setSyncError] = useState<string | null>(null);
  const [unpricedRunning, setUnpricedRunning] = useState(false);
  const [unpricedResult, setUnpricedResult] = useState<UsagePriceUnpricedResponse | null>(null);
  const [unpricedError, setUnpricedError] = useState<string | null>(null);
  const [correctionRunning, setCorrectionRunning] = useState(false);
  const [correctionResult, setCorrectionResult] =
    useState<UsagePriceCorrectionResponse | null>(null);
  const [correctionError, setCorrectionError] = useState<string | null>(null);

  const list = useUsageQuery<UsagePriceListResponse>({
    load: (signal) => usageApi.listPrices(priceListQuery(filters, page), { signal }),
    queryKey: priceQueryKey(filters, page),
  });
  const status = useUsageQuery<UsagePriceStatusResponse>({
    load: (signal) => usageApi.getPriceStatus({ signal }),
    queryKey: 'usage-price-status',
  });

  const disabled = isUsageDisabledError(list.error) || isUsageDisabledError(status.error);
  const total = list.data?.total ?? 0;
  const pages = pricePageCount(total);
  const currentPage = Math.min(page, pages);
  const syncConfig = useMemo(() => readPriceSyncConfig(config), [config]);

  useEffect(() => {
    fetchConfig().catch(() => {
      // The layout loads the config too; a failure only hides the read-only sync state.
    });
  }, [fetchConfig]);

  useEffect(() => {
    if (page > pages) setPage(pages);
  }, [page, pages]);

  const listRefresh = list.refresh;
  const statusRefresh = status.refresh;

  const refreshAll = useCallback(() => {
    void Promise.allSettled([listRefresh(), statusRefresh()]);
  }, [listRefresh, statusRefresh]);

  useHeaderRefresh(refreshAll);

  const refreshData = useCallback(async () => {
    await Promise.allSettled([listRefresh(), statusRefresh()]);
  }, [listRefresh, statusRefresh]);

  const handleSync = async () => {
    setSyncing(true);
    setSyncError(null);
    try {
      const result = await usageApi.syncPrices();
      setSyncResult(result);
      await refreshData();
    } catch (error: unknown) {
      setSyncError(errorMessage(error));
    } finally {
      setSyncing(false);
    }
  };

  const handlePriceUnpriced = async (body: {
    from_ms?: number;
    to_ms?: number;
    model?: string;
  }) => {
    setUnpricedRunning(true);
    setUnpricedError(null);
    try {
      const result = await usageApi.priceUnpriced(body);
      setUnpricedResult(result);
      await refreshData();
    } catch (error: unknown) {
      setUnpricedError(errorMessage(error));
    } finally {
      setUnpricedRunning(false);
    }
  };

  const handleCorrect = async (input: UsagePriceCorrectionInput) => {
    setCorrectionRunning(true);
    setCorrectionError(null);
    try {
      const result = await usageApi.correctPrices(input);
      setCorrectionResult(result);
      await refreshData();
    } catch (error: unknown) {
      setCorrectionError(errorMessage(error));
    } finally {
      setCorrectionRunning(false);
    }
  };

  const handleSave = async (values: PriceFormValues) => {
    const parsed = validatePriceForm(values);
    if (parsed.input === null) throw new Error(t('usage_prices.form.error_required'));
    const saved = await usageApi.putPrice(parsed.input);
    setForm(null);
    showNotification(t('usage_prices.form.saved', { model: saved.model }), 'success');
    await refreshData();
  };

  const handleRemoveManual = (price: UsagePriceItem) => {
    showConfirmation({
      title: t('usage_prices.remove.title'),
      message: t('usage_prices.remove.message', {
        provider: price.provider,
        model: price.model,
      }),
      variant: 'danger',
      confirmText: t('usage_prices.remove.confirm'),
      onConfirm: async () => {
        await usageApi.deleteManualPrice(price.provider, price.model);
        showNotification(t('usage_prices.remove.done'), 'success');
        await refreshData();
      },
    });
  };

  const openCreate = (seed: { provider?: string; model?: string } = {}) => {
    setForm({ mode: 'create', initial: emptyPriceForm(seed) });
  };

  const listError =
    list.error && !isUsageDisabledError(list.error) && !list.data ? list.error : null;
  const data = list.data?.prices ?? [];
  const filtered = Boolean(
    filters.provider.trim() || filters.source.trim() || filters.q.trim()
  );

  return (
    <div className={styles.container}>
      <PageHeader
        title={t('usage_prices.title')}
        description={t('usage_prices.description')}
        actions={
          <>
            <RefreshButton
              onClick={refreshAll}
              label={t('usage_prices.refresh')}
              refreshing={list.loading || status.loading}
            />
            <Button onClick={() => openCreate()} disabled={disabled}>
              {t('usage_prices.add_price')}
            </Button>
          </>
        }
      />

      {disabled ? (
        <Card className={styles.noticeCard} title={t('usage_prices.disabled_title')}>
          <p className={styles.noticeText}>{t('usage_prices.disabled_detail')}</p>
        </Card>
      ) : (
        <>
          <PriceSyncPanel
            status={status.data}
            config={syncConfig}
            result={syncResult}
            error={syncError}
            syncing={syncing}
            onSync={() => void handleSync()}
          />

          <Card className={styles.listCard} title={t('usage_prices.list_title')}>
            {listError ? (
              <div className={styles.errorBlock} role="alert">
                <p className={styles.errorText}>{t('usage_prices.load_error')}</p>
                <p className={styles.noticeText}>{listError.message}</p>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => void list.refresh().catch(() => undefined)}
                >
                  {t('usage_prices.retry')}
                </Button>
              </div>
            ) : (
              <PricesTable
                prices={data}
                total={total}
                page={currentPage}
                pages={pages}
                loading={list.loading}
                filters={filters}
                filtered={filtered}
                onFiltersChange={(next) => {
                  setFilters(next);
                  setPage(1);
                }}
                onPageChange={setPage}
                onEdit={(price) =>
                  setForm({ mode: 'edit', initial: priceToFormValues(price) })
                }
                onHistory={(price) =>
                  setHistory({ provider: price.provider, model: price.model })
                }
                onRemoveManual={handleRemoveManual}
              />
            )}
          </Card>

          <UnmatchedPrices
            models={status.data?.unmatched ?? []}
            onAdd={(provider, model) => openCreate({ provider, model })}
          />

          <PriceUnpricedPanel
            result={unpricedResult}
            error={unpricedError}
            running={unpricedRunning}
            onRun={(body) => void handlePriceUnpriced(body)}
          />

          <CorrectionPanel
            result={correctionResult}
            error={correctionError}
            running={correctionRunning}
            onRun={(input) => void handleCorrect(input)}
          />
        </>
      )}

      <PriceFormSheet
        open={form !== null}
        mode={form?.mode ?? 'create'}
        initial={form?.initial ?? emptyPriceForm()}
        onClose={() => setForm(null)}
        onSave={handleSave}
      />
      <PriceHistorySheet
        open={history !== null}
        provider={history?.provider ?? ''}
        model={history?.model ?? ''}
        onClose={() => setHistory(null)}
      />
    </div>
  );
}
