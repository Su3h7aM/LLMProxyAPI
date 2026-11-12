import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Sheet } from '@/components/ui/Sheet';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/Table';
import { Button } from '@/components/ui/Button';
import { formatPriceRate, historyNewestFirst } from '@/features/usage/logic/prices';
import { usageApi } from '@/services/api/usage';
import type { UsagePriceItem } from '@/types/usageStats';
import { formatDateTimeValue } from '@/utils/format';
import styles from './pricePanels.module.scss';

interface PriceHistorySheetProps {
  open: boolean;
  provider: string;
  model: string;
  onClose: () => void;
}

export function PriceHistoryTable({
  versions,
  locale,
}: {
  versions: UsagePriceItem[];
  locale: string;
}) {
  const { t } = useTranslation();

  if (versions.length === 0) {
    return <p className="hint">{t('usage_prices.history.empty')}</p>;
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('usage_prices.history.from')}</TableHead>
          <TableHead>{t('usage_prices.history.to')}</TableHead>
          <TableHead>{t('usage_prices.history.source')}</TableHead>
          <TableHead alignRight>{t('usage_prices.history.input')}</TableHead>
          <TableHead alignRight>{t('usage_prices.history.output')}</TableHead>
          <TableHead alignRight>{t('usage_prices.history.cache_read')}</TableHead>
          <TableHead alignRight>{t('usage_prices.history.cache_write')}</TableHead>
          <TableHead alignRight>{t('usage_prices.history.reasoning')}</TableHead>
          <TableHead>{t('usage_prices.history.note')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {versions.map((version) => (
          <TableRow key={version.id}>
            <TableCell>
              {version.valid_from_ms > 0
                ? formatDateTimeValue(version.valid_from_ms, locale)
                : t('usage_prices.history.start_of_time')}
            </TableCell>
            <TableCell>
              {version.valid_to_ms === null
                ? t('usage_prices.history.current')
                : formatDateTimeValue(version.valid_to_ms, locale)}
            </TableCell>
            <TableCell>
              <span className="pill">{version.source}</span>
            </TableCell>
            <TableCell alignRight>{formatPriceRate(version.input, locale)}</TableCell>
            <TableCell alignRight>{formatPriceRate(version.output, locale)}</TableCell>
            <TableCell alignRight>{formatPriceRate(version.cache_read, locale)}</TableCell>
            <TableCell alignRight>{formatPriceRate(version.cache_write, locale)}</TableCell>
            <TableCell alignRight>
              {version.reasoning === null ? '—' : formatPriceRate(version.reasoning, locale)}
            </TableCell>
            <TableCell>{version.note || '—'}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

export function PriceHistorySheet({ open, provider, model, onClose }: PriceHistorySheetProps) {
  const { t, i18n } = useTranslation();
  const locale = i18n.language;
  const [versions, setVersions] = useState<UsagePriceItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [reload, setReload] = useState(0);

  useEffect(() => {
    if (!open) return undefined;
    const controller = new AbortController();
    setLoading(true);
    setError(null);
    usageApi
      .getPriceHistory(provider, model, { signal: controller.signal })
      .then((response) => {
        if (controller.signal.aborted) return;
        setVersions(historyNewestFirst(response.versions));
      })
      .catch((cause: unknown) => {
        if (controller.signal.aborted) return;
        setError(cause instanceof Error ? cause.message : String(cause));
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [open, provider, model, reload]);

  return (
    <Sheet
      open={open}
      onClose={onClose}
      size="xl"
      eyebrow={t('usage_prices.history.eyebrow', { provider, model })}
      title={t('usage_prices.history.title')}
      description={t('usage_prices.history.description')}
    >
      {loading ? (
        <p className="hint" role="status">
          {t('usage_prices.history.loading')}
        </p>
      ) : error ? (
        <div role="alert">
          <p className={styles.errorText}>{t('usage_prices.history.error')}</p>
          <p className={styles.hint}>{error}</p>
          <Button variant="secondary" size="sm" onClick={() => setReload((value) => value + 1)}>
            {t('usage_prices.retry')}
          </Button>
        </div>
      ) : (
        <PriceHistoryTable versions={versions} locale={locale} />
      )}
    </Sheet>
  );
}
