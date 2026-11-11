import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { IconChevronDown, IconPlus, IconX } from '@/components/ui/icons';
import type { ModelEntryInput, ProviderBrand } from '../../types';
import { ModelAdvancedFields } from './ModelAdvancedFields';
import styles from './sharedForm.module.scss';
import { iconSize } from '@/design-system';

const COLLAPSED_LIMIT = 10;

interface ModelEntriesEditorProps {
  models: ModelEntryInput[];
  /** Restrict the rows to the credential alias contract (no image/thinking controls). */
  oauthAliasOnly?: boolean;
  providerBrand?: ProviderBrand;
  /** Only OpenAI-compatible entries can expose the image-generation capability. */
  supportsImage: boolean;
  /** Every backend provider model can override its thinking capability. */
  supportsThinking: boolean;
  mutating: boolean;
  removeDisabled: boolean;
  onUpdate: (idx: number, patch: Partial<ModelEntryInput>) => void;
  onAdd: () => void;
  onRemove: (idx: number) => void;
}

export function ModelEntriesEditor({
  models,
  oauthAliasOnly = false,
  providerBrand = 'gemini',
  supportsImage,
  supportsThinking,
  mutating,
  removeDisabled,
  onUpdate,
  onAdd,
  onRemove,
}: ModelEntriesEditorProps) {
  const { t } = useTranslation();
  const [expandedIdx, setExpandedIdx] = useState<number | null>(null);
  const [showAll, setShowAll] = useState(false);

  const handleAdd = () => {
    // New rows are appended; make sure the truncated list doesn't hide them.
    if (!showAll && models.length >= COLLAPSED_LIMIT) {
      setShowAll(true);
    }
    onAdd();
  };

  const handleRemove = (removeIdx: number) => {
    setExpandedIdx((prev) => {
      if (prev === null || prev === removeIdx) return null;
      return prev > removeIdx ? prev - 1 : prev;
    });
    onRemove(removeIdx);
  };

  const visible = showAll ? models : models.slice(0, COLLAPSED_LIMIT);

  return (
    <>
      {visible.map((entry, idx) => {
        const hasExtendedOptions = supportsImage || supportsThinking || oauthAliasOnly;
        const expanded = hasExtendedOptions && expandedIdx === idx;
        const hasThinking = entry.thinkingEnabled ?? Boolean(entry.thinkingJson?.trim());
        return (
          <div key={idx} className={styles.modelEntry}>
            <div className={styles.modelAliasRow}>
              <input
                className={styles.input}
                placeholder="model-name"
                value={entry.name}
                onChange={(e) => onUpdate(idx, { name: e.target.value })}
                disabled={mutating}
              />
              <input
                className={styles.input}
                placeholder={oauthAliasOnly ? t('auth_files.policy_alias_name') : 'alias (optional)'}
                aria-label={
                  oauthAliasOnly ? t('auth_files.policy_alias_name') : 'alias (optional)'
                }
                required={oauthAliasOnly}
                value={entry.alias ?? ''}
                onChange={(e) => onUpdate(idx, { alias: e.target.value })}
                disabled={mutating}
              />
              <div className={styles.modelEntryActions}>
                {!oauthAliasOnly && supportsImage && !expanded && entry.image === true ? (
                  <span className={styles.entryBadge}>
                    {t('providersPage.form.modelBadgeImage')}
                  </span>
                ) : null}
                {!oauthAliasOnly && supportsThinking && !expanded && hasThinking ? (
                  <span className={styles.entryBadge}>
                    {t('providersPage.form.modelBadgeThinking')}
                  </span>
                ) : null}
                {hasExtendedOptions ? (
                  <button
                    type="button"
                    className={styles.entryCardIconBtn}
                    onClick={() => setExpandedIdx(expanded ? null : idx)}
                    title={expanded ? t('common.collapse') : t('common.expand')}
                    aria-label={expanded ? t('common.collapse') : t('common.expand')}
                    aria-expanded={expanded}
                    disabled={mutating}
                  >
                    <IconChevronDown
                      className={[
                        styles.entryCardChevron,
                        expanded ? styles.entryCardChevronOpen : '',
                      ]
                        .filter(Boolean)
                        .join(' ')}
                      size={iconSize['14']}
                    />
                  </button>
                ) : null}
                <button
                  type="button"
                  className={styles.removeBtn}
                  disabled={mutating || removeDisabled}
                  onClick={() => handleRemove(idx)}
                >
                  <IconX size={iconSize['12']} />
                </button>
              </div>
            </div>
            {expanded ? (
              <div className={styles.modelEntryDetails}>
                {!oauthAliasOnly && supportsImage ? (
                  <label className={styles.checkboxRow}>
                    <input
                      type="checkbox"
                      className={styles.checkboxBox}
                      checked={entry.image === true}
                      disabled={mutating}
                      onChange={(e) => onUpdate(idx, { image: e.target.checked })}
                    />
                    <span className={styles.checkboxText}>
                      <span>{t('providersPage.form.modelImage')}</span>
                      <small>{t('providersPage.form.modelImageHint')}</small>
                    </span>
                  </label>
                ) : null}
                {oauthAliasOnly ? (
                  <>
                    <div className={styles.field}>
                      <label className={styles.label}>
                        {t('auth_files.policy_alias_display_name')}
                        <input
                          className={styles.input}
                          value={entry.displayName ?? ''}
                          disabled={mutating}
                          onChange={(event) => onUpdate(idx, { displayName: event.target.value })}
                        />
                      </label>
                    </div>
                    {(['fork', 'forceMapping'] as const).map((field) => (
                      <label key={field} className={styles.checkboxRow}>
                        <input
                          type="checkbox"
                          className={styles.checkboxBox}
                          checked={entry[field] === true}
                          disabled={mutating}
                          onChange={(event) => onUpdate(idx, { [field]: event.target.checked })}
                        />
                        <span className={styles.checkboxText}>
                          {t(
                            field === 'fork'
                              ? 'auth_files.policy_alias_fork'
                              : 'auth_files.policy_alias_force_mapping'
                          )}
                        </span>
                      </label>
                    ))}
                  </>
                ) : (
                  <ModelAdvancedFields
                    entry={entry}
                    providerBrand={providerBrand}
                    disabled={mutating}
                    supportsThinking={supportsThinking}
                    onUpdate={(patch) => onUpdate(idx, patch)}
                  />
                )}
              </div>
            ) : null}
          </div>
        );
      })}
      {models.length > COLLAPSED_LIMIT ? (
        <button type="button" className={styles.showMoreBtn} onClick={() => setShowAll((v) => !v)}>
          {showAll
            ? t('providersPage.form.showFewerEntries')
            : t('providersPage.form.showAllEntries', { count: models.length })}
        </button>
      ) : null}
      <button type="button" className={styles.addBtn} disabled={mutating} onClick={handleAdd}>
        <IconPlus size={iconSize['12']} />
        <span>{t('providersPage.form.addModel')}</span>
      </button>
    </>
  );
}
