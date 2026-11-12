import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Card } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import {
  unpricedBody,
  type UnpricedErrorCode,
} from '@/features/usage/logic/prices';
import type { UsagePriceUnpricedResponse } from '@/types/usageStats';
import styles from './pricePanels.module.scss';

interface PriceUnpricedPanelProps {
  result: UsagePriceUnpricedResponse | null;
  error: string | null;
  running: boolean;
  onRun: (body: { from_ms?: number; to_ms?: number; model?: string }) => void;
}

export function PriceUnpricedPanel({
  result,
  error,
  running,
  onRun,
}: PriceUnpricedPanelProps) {
  const { t } = useTranslation();
  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');
  const [model, setModel] = useState('');
  const [errors, setErrors] = useState<UnpricedErrorCode[]>([]);

  const errorText = (code: UnpricedErrorCode): string => {
    if (code === 'range_pair') return t('usage_prices.unpriced.error_range_pair');
    if (code === 'range_order') return t('usage_prices.unpriced.error_range_order');
    return t('usage_prices.unpriced.error_range_too_long');
  };

  return (
    <Card className={styles.panel} title={t('usage_prices.unpriced.title')}>
      <p className={styles.hint}>{t('usage_prices.unpriced.description')}</p>
      <div className={styles.formGrid}>
        <Input
          label={t('usage_prices.unpriced.from')}
          type="datetime-local"
          value={from}
          onChange={(event) => setFrom(event.target.value)}
        />
        <Input
          label={t('usage_prices.unpriced.to')}
          type="datetime-local"
          value={to}
          onChange={(event) => setTo(event.target.value)}
        />
        <Input
          label={t('usage_prices.unpriced.model')}
          value={model}
          onChange={(event) => setModel(event.target.value)}
          autoComplete="off"
        />
        <div className={styles.actionsRow}>
          <Button
            variant="secondary"
            loading={running}
            onClick={() => {
              const parsed = unpricedBody({ from, to, model });
              setErrors(parsed.errors);
              if (parsed.body !== null) onRun(parsed.body);
            }}
          >
            {running ? t('usage_prices.unpriced.running') : t('usage_prices.unpriced.button')}
          </Button>
        </div>
      </div>
      {errors.map((code) => (
        <p className={styles.errorText} role="alert" key={code}>
          {errorText(code)}
        </p>
      ))}
      {result ? (
        <p className={styles.hint} role="status">
          {t('usage_prices.unpriced.result', {
            priced: result.priced,
            skipped: result.skipped_no_price,
          })}
        </p>
      ) : null}
      {error ? (
        <p className={styles.errorText} role="alert">
          {t('usage_prices.unpriced.failed', { error })}
        </p>
      ) : null}
    </Card>
  );
}
