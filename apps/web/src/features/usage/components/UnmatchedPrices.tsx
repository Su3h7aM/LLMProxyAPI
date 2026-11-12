import { useTranslation } from 'react-i18next';
import { Card } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import type { UsageUnmatchedPriceModel } from '@/types/usageStats';
import styles from './pricePanels.module.scss';

interface UnmatchedPricesProps {
  models: UsageUnmatchedPriceModel[];
  onAdd: (provider: string, model: string) => void;
}

export function UnmatchedPrices({ models, onAdd }: UnmatchedPricesProps) {
  const { t } = useTranslation();

  return (
    <Card className={styles.panel} title={t('usage_prices.unmatched.title')}>
      <p className={styles.hint}>{t('usage_prices.unmatched.description')}</p>
      {models.length === 0 ? (
        <p className={styles.hint}>{t('usage_prices.unmatched.empty')}</p>
      ) : (
        <div className={styles.list}>
          {models.map((item) => (
            <div className={styles.listRow} key={`${item.provider}:${item.model}`}>
              <div className={styles.listPrimary}>
                <span className={styles.listModel}>{item.model}</span>
                <span className={styles.listMeta}>
                  {item.provider}
                  {' · '}
                  {t('usage_prices.unmatched.events', { count: item.events })}
                </span>
              </div>
              <Button
                variant="secondary"
                size="sm"
                onClick={() => onAdd(item.provider, item.model)}
              >
                {t('usage_prices.unmatched.add')}
              </Button>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}
