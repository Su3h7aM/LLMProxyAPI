import { useTranslation } from 'react-i18next';
import type { SiwcBindingStatus } from '../siwcBinding';
import styles from './SiwcStatusBadge.module.scss';

/**
 * SIWC state pill: a dot and a short state word, in the same shape as the provider
 * region badge and the credential state chip.
 * `withLabel` drops the "SIWC" prefix where the context already names it.
 */
export function SiwcStatusBadge({
  status,
  withLabel = true,
}: {
  status: SiwcBindingStatus;
  withLabel?: boolean;
}) {
  const { t } = useTranslation();
  return (
    <span className={`${styles.badge} ${styles[status]}`}>
      <span className={styles.dot} aria-hidden="true" />
      {withLabel && <span className={styles.label}>SIWC</span>}
      <span>{t(`siwc.${status}`)}</span>
    </span>
  );
}
