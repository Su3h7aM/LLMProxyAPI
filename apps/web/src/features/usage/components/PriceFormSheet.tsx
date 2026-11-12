import { useEffect, useId, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Sheet } from '@/components/ui/Sheet';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import {
  validatePriceForm,
  type PriceFormErrorCode,
  type PriceFormErrors,
  type PriceFormValues,
} from '@/features/usage/logic/prices';
import styles from './pricePanels.module.scss';

interface PriceFormSheetProps {
  open: boolean;
  mode: 'create' | 'edit';
  initial: PriceFormValues;
  onClose: () => void;
  /** Save the validated body; a rejection keeps the sheet open with the error. */
  onSave: (values: PriceFormValues) => Promise<void>;
}

export function PriceFormSheet({ open, mode, initial, onClose, onSave }: PriceFormSheetProps) {
  const { t } = useTranslation();
  const formId = useId();
  const [values, setValues] = useState<PriceFormValues>(initial);
  const [errors, setErrors] = useState<PriceFormErrors>({});
  const [saving, setSaving] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      setValues(initial);
      setErrors({});
      setSubmitError(null);
      setSaving(false);
    }
  }, [initial, open]);

  const errorText = (code: PriceFormErrorCode | undefined): string | undefined => {
    if (!code) return undefined;
    if (code === 'required') return t('usage_prices.form.error_required');
    if (code === 'number') return t('usage_prices.form.error_number');
    if (code === 'negative') return t('usage_prices.form.error_negative');
    return t('usage_prices.form.error_date');
  };

  const update = (field: keyof PriceFormValues, value: string) => {
    setValues((current) => ({ ...current, [field]: value }));
  };

  const submit = async () => {
    const result = validatePriceForm(values);
    setErrors(result.errors);
    if (result.input === null) return;
    setSaving(true);
    setSubmitError(null);
    try {
      await onSave(values);
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : String(error);
      setSubmitError(message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Sheet
      open={open}
      onClose={onClose}
      size="lg"
      closeDisabled={saving}
      eyebrow={
        mode === 'create'
          ? t('usage_prices.form.create_eyebrow')
          : t('usage_prices.form.edit_eyebrow')
      }
      title={
        mode === 'create'
          ? t('usage_prices.form.create_title')
          : t('usage_prices.form.edit_title', { model: values.model })
      }
      description={t('usage_prices.form.description')}
      focusKey={`${mode}:${initial.provider}:${initial.model}`}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={saving}>
            {t('usage_prices.form.cancel')}
          </Button>
          <Button type="submit" form={formId} loading={saving}>
            {saving ? t('usage_prices.form.saving') : t('usage_prices.form.save')}
          </Button>
        </>
      }
    >
      <form
        id={formId}
        className={styles.formGrid}
        onSubmit={(event) => {
          event.preventDefault();
          void submit();
        }}
        noValidate
      >
        <Input
          label={t('usage_prices.form.provider')}
          value={values.provider}
          onChange={(event) => update('provider', event.target.value)}
          placeholder={t('usage_prices.form.provider_placeholder')}
          readOnly={mode === 'edit'}
          error={errorText(errors.provider)}
          autoComplete="off"
        />
        <Input
          label={t('usage_prices.form.model')}
          value={values.model}
          onChange={(event) => update('model', event.target.value)}
          placeholder={t('usage_prices.form.model_placeholder')}
          readOnly={mode === 'edit'}
          error={errorText(errors.model)}
          autoComplete="off"
        />
        <Input
          label={t('usage_prices.form.input')}
          type="number"
          min={0}
          step="any"
          inputMode="decimal"
          value={values.input}
          onChange={(event) => update('input', event.target.value)}
          error={errorText(errors.input)}
        />
        <Input
          label={t('usage_prices.form.output')}
          type="number"
          min={0}
          step="any"
          inputMode="decimal"
          value={values.output}
          onChange={(event) => update('output', event.target.value)}
          error={errorText(errors.output)}
        />
        <Input
          label={t('usage_prices.form.cache_read')}
          type="number"
          min={0}
          step="any"
          inputMode="decimal"
          value={values.cacheRead}
          onChange={(event) => update('cacheRead', event.target.value)}
          error={errorText(errors.cacheRead)}
        />
        <Input
          label={t('usage_prices.form.cache_write')}
          type="number"
          min={0}
          step="any"
          inputMode="decimal"
          value={values.cacheWrite}
          onChange={(event) => update('cacheWrite', event.target.value)}
          error={errorText(errors.cacheWrite)}
        />
        <Input
          label={t('usage_prices.form.reasoning')}
          type="number"
          min={0}
          step="any"
          inputMode="decimal"
          value={values.reasoning}
          onChange={(event) => update('reasoning', event.target.value)}
          hint={t('usage_prices.form.rate_hint')}
          error={errorText(errors.reasoning)}
        />
        <Input
          label={t('usage_prices.form.effective_from')}
          type="datetime-local"
          value={values.effectiveFrom}
          onChange={(event) => update('effectiveFrom', event.target.value)}
          hint={t('usage_prices.form.effective_from_hint')}
          error={errorText(errors.effectiveFrom)}
        />
        <div className={styles.fullWidth}>
          <Input
            label={t('usage_prices.form.note')}
            value={values.note}
            onChange={(event) => update('note', event.target.value)}
            placeholder={t('usage_prices.form.note_placeholder')}
          />
        </div>
        {submitError ? (
          <p className={`${styles.errorText} ${styles.fullWidth}`} role="alert">
            {submitError}
          </p>
        ) : null}
      </form>
    </Sheet>
  );
}
