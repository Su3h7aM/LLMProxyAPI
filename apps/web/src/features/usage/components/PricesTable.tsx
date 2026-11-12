import { useTranslation } from 'react-i18next';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/Table';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Select } from '@/components/ui/Select';
import { EmptyState } from '@/components/ui/EmptyState';
import {
  formatPriceRate,
  PRICE_SOURCES,
  type PriceFilters,
} from '@/features/usage/logic/prices';
import type { UsagePriceItem } from '@/types/usageStats';
import { formatDateTimeValue } from '@/utils/format';
import styles from './pricePanels.module.scss';

interface PricesTableProps {
  prices: UsagePriceItem[];
  total: number;
  page: number;
  pages: number;
  loading: boolean;
  filters: PriceFilters;
  /** The list is empty only because a filter is set. */
  filtered: boolean;
  onFiltersChange: (filters: PriceFilters) => void;
  onPageChange: (page: number) => void;
  onEdit: (price: UsagePriceItem) => void;
  onHistory: (price: UsagePriceItem) => void;
  onRemoveManual: (price: UsagePriceItem) => void;
}

export function PricesTable({
  prices,
  total,
  page,
  pages,
  loading,
  filters,
  filtered,
  onFiltersChange,
  onPageChange,
  onEdit,
  onHistory,
  onRemoveManual,
}: PricesTableProps) {
  const { t, i18n } = useTranslation();
  const locale = i18n.language;
  const sourceOptions = [
    { value: '', label: t('usage_prices.filters.source_all') },
    ...PRICE_SOURCES.map((source) => ({ value: source, label: source })),
  ];
  const noFilters =
    !filters.provider.trim() && !filters.source.trim() && !filters.q.trim();

  return (
    <div className={styles.panel}>
      <div className={styles.formGrid}>
        <Input
          label={t('usage_prices.filters.provider')}
          value={filters.provider}
          onChange={(event) => onFiltersChange({ ...filters, provider: event.target.value })}
          placeholder={t('usage_prices.filters.provider_placeholder')}
          autoComplete="off"
        />
        <div className="form-group">
          <label htmlFor="usage-price-source">{t('usage_prices.filters.source')}</label>
          <Select
            id="usage-price-source"
            value={filters.source}
            options={sourceOptions}
            onChange={(value) => onFiltersChange({ ...filters, source: value })}
            ariaLabel={t('usage_prices.filters.source')}
          />
        </div>
        <Input
          label={t('usage_prices.filters.search')}
          type="search"
          value={filters.q}
          onChange={(event) => onFiltersChange({ ...filters, q: event.target.value })}
          placeholder={t('usage_prices.filters.search_placeholder')}
          autoComplete="off"
        />
        <div className={styles.actionsRow}>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => onFiltersChange({ provider: '', source: '', q: '' })}
            disabled={noFilters}
          >
            {t('usage_prices.filters.reset')}
          </Button>
        </div>
      </div>

      {loading && prices.length === 0 ? (
        <p className="hint" role="status">
          {t('usage_prices.table.loading')}
        </p>
      ) : prices.length === 0 ? (
        <EmptyState
          title={
            filtered
              ? t('usage_prices.table.no_match_title')
              : t('usage_prices.table.empty_title')
          }
          description={
            filtered
              ? t('usage_prices.table.no_match_detail')
              : t('usage_prices.table.empty_detail')
          }
        />
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('usage_prices.table.provider')}</TableHead>
              <TableHead>{t('usage_prices.table.model')}</TableHead>
              <TableHead alignRight>{t('usage_prices.table.input')}</TableHead>
              <TableHead alignRight>{t('usage_prices.table.output')}</TableHead>
              <TableHead alignRight>{t('usage_prices.table.cache_read')}</TableHead>
              <TableHead alignRight>{t('usage_prices.table.cache_write')}</TableHead>
              <TableHead alignRight>{t('usage_prices.table.reasoning')}</TableHead>
              <TableHead>{t('usage_prices.table.source')}</TableHead>
              <TableHead>{t('usage_prices.table.valid_from')}</TableHead>
              <TableHead>{t('usage_prices.table.synced_at')}</TableHead>
              <TableHead alignRight>{t('usage_prices.table.actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {prices.map((price) => (
              <TableRow key={price.id}>
                <TableCell>{price.provider}</TableCell>
                <TableCell>
                  <span className={styles.listModel}>{price.model}</span>
                </TableCell>
                <TableCell alignRight className={styles.cellMain}>
                  {formatPriceRate(price.input, locale)}
                </TableCell>
                <TableCell alignRight className={styles.cellMain}>
                  {formatPriceRate(price.output, locale)}
                </TableCell>
                <TableCell alignRight className={styles.cellMain}>
                  {formatPriceRate(price.cache_read, locale)}
                </TableCell>
                <TableCell alignRight className={styles.cellMain}>
                  {formatPriceRate(price.cache_write, locale)}
                </TableCell>
                <TableCell alignRight className={styles.cellMain}>
                  {price.reasoning === null ? '—' : formatPriceRate(price.reasoning, locale)}
                </TableCell>
                <TableCell>
                  <span className="pill">{price.source}</span>
                </TableCell>
                <TableCell>
                  {price.valid_from_ms > 0
                    ? formatDateTimeValue(price.valid_from_ms, locale)
                    : t('usage_prices.table.start_of_time')}
                </TableCell>
                <TableCell>
                  {price.synced_at_ms > 0
                    ? formatDateTimeValue(price.synced_at_ms, locale)
                    : t('usage_prices.sync.never')}
                </TableCell>
                <TableCell alignRight>
                  <div className={styles.actionsRow}>
                    <Button variant="ghost" size="sm" onClick={() => onEdit(price)}>
                      {t('usage_prices.actions.edit')}
                    </Button>
                    <Button variant="ghost" size="sm" onClick={() => onHistory(price)}>
                      {t('usage_prices.actions.history')}
                    </Button>
                    {price.source === 'manual' ? (
                      <Button
                        variant="danger-outline"
                        size="sm"
                        onClick={() => onRemoveManual(price)}
                      >
                        {t('usage_prices.actions.remove_manual')}
                      </Button>
                    ) : null}
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      <div className={styles.actionsRow}>
        <span className={styles.hint}>{t('usage_prices.table.total', { count: total })}</span>
        <span className={styles.hint}>
          {t('usage_prices.table.page', { page, pages })}
        </span>
        <Button
          variant="secondary"
          size="sm"
          onClick={() => onPageChange(page - 1)}
          disabled={page <= 1 || loading}
        >
          {t('usage_prices.table.previous')}
        </Button>
        <Button
          variant="secondary"
          size="sm"
          onClick={() => onPageChange(page + 1)}
          disabled={page >= pages || loading}
        >
          {t('usage_prices.table.next')}
        </Button>
      </div>
    </div>
  );
}
