import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Card } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { Collapsible } from '@/components/ui/Collapsible/Collapsible';
import { Input } from '@/components/ui/Input';
import {
  validateCorrectionForm,
  type CorrectionErrorCode,
  type CorrectionFormValues,
} from '@/features/usage/logic/prices';
import { formatMicroUsd } from '@/features/usage/logic/format';
import type { UsagePriceCorrectionInput, UsagePriceCorrectionResponse } from '@/types/usageStats';
import styles from './pricePanels.module.scss';

interface CorrectionPanelProps {
  result: UsagePriceCorrectionResponse | null;
  error: string | null;
  running: boolean;
  onRun: (input: UsagePriceCorrectionInput) => void;
}

const EMPTY: CorrectionFormValues = {
  from: '',
  to: '',
  model: '',
  reason: '',
  confirmation: '',
};

export function CorrectionPanel({ result, error, running, onRun }: CorrectionPanelProps) {
  const { t, i18n } = useTranslation();
  const [values, setValues] = useState<CorrectionFormValues>(EMPTY);
  const [errors, setErrors] = useState<CorrectionErrorCode[]>([]);
  const word = t('usage_prices.correction.confirm_word');

  const update = (field: keyof CorrectionFormValues, value: string) => {
    setValues((current) => ({ ...current, [field]: value }));
  };

  const errorText = (code: CorrectionErrorCode): string => {
    if (code === 'range_required') return t('usage_prices.correction.error_range_required');
    if (code === 'range_order') return t('usage_prices.correction.error_range_order');
    if (code === 'range_too_long') return t('usage_prices.correction.error_range_too_long');
    if (code === 'reason_required') return t('usage_prices.correction.error_reason_required');
    return t('usage_prices.correction.error_confirm_mismatch');
  };

  return (
    <Card className={styles.panel} title={t('usage_prices.correction.title')}>
      <p className={styles.hint}>{t('usage_prices.correction.description')}</p>
      <div className="status-badge warning">{t('usage_prices.correction.warning')}</div>
      <Collapsible label={t('usage_prices.correction.form_label')}>
        <div className={styles.formGrid}>
          <Input
            label={t('usage_prices.correction.from')}
            type="datetime-local"
            value={values.from}
            onChange={(event) => update('from', event.target.value)}
          />
          <Input
            label={t('usage_prices.correction.to')}
            type="datetime-local"
            value={values.to}
            onChange={(event) => update('to', event.target.value)}
          />
          <Input
            label={t('usage_prices.correction.model')}
            value={values.model}
            onChange={(event) => update('model', event.target.value)}
            autoComplete="off"
          />
          <div className={styles.fullWidth}>
            <Input
              label={t('usage_prices.correction.reason')}
              value={values.reason}
              onChange={(event) => update('reason', event.target.value)}
              placeholder={t('usage_prices.correction.reason_placeholder')}
            />
          </div>
          <Input
            label={t('usage_prices.correction.confirm_label', { word })}
            value={values.confirmation}
            onChange={(event) => update('confirmation', event.target.value)}
            autoComplete="off"
          />
          <div className={styles.actionsRow}>
            <Button
              variant="danger-outline"
              loading={running}
              onClick={() => {
                const parsed = validateCorrectionForm(values, word);
                setErrors(parsed.errors);
                if (parsed.input !== null) onRun(parsed.input);
              }}
            >
              {running ? t('usage_prices.correction.running') : t('usage_prices.correction.button')}
            </Button>
          </div>
        </div>
        {errors.map((code) => (
          <p className={styles.errorText} role="alert" key={code}>
            {errorText(code)}
          </p>
        ))}
      </Collapsible>
      {result ? (
        <p className={styles.hint} role="status">
          {t('usage_prices.correction.result', {
            id: result.correction_id,
            count: result.events_updated,
            old: formatMicroUsd(result.old_cost_micro_usd, i18n.language),
            new: formatMicroUsd(result.new_cost_micro_usd, i18n.language),
          })}
        </p>
      ) : null}
      {error ? (
        <p className={styles.errorText} role="alert">
          {t('usage_prices.correction.failed', { error })}
        </p>
      ) : null}
    </Card>
  );
}
