import { useEffect, useId, useRef, useState, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal } from '@/components/ui/Modal';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { IconExternalLink } from '@/components/ui/icons';
import { iconSize } from '@/design-system';
import { authFilesApi } from '@/services/api/authFiles';
import { captureSiwcBindingApi, oauthApi, type SiwcBindResponse } from '@/services/api/oauth';
import { useAuthStore, useNotificationStore } from '@/stores';
import type { AuthFileItem } from '@/types';
import { copyToClipboard } from '@/utils/clipboard';
import { formatDateTimeValue } from '@/utils/format';
import { notifyAuthFilesChanged } from '@/features/authFiles/authFilesEvents';
import { deriveAuthFileIdentity } from '@/features/authFiles/identity';
import { createOAuthAttempts, type OAuthAttempt } from '@/pages/oauthAttempts';
import {
  createSiwcStart,
  getSiwcBindingStatus,
  isSiwcCallbackValid,
  listSiwcTargets,
} from '../siwcBinding';
import { SiwcStatusBadge } from './SiwcStatusBadge';
import styles from './SiwcBindDialog.module.scss';

type Phase = 'idle' | 'starting' | 'wait' | 'ok' | 'error' | 'expired';
type Feedback = 'submitted' | 'invalid_callback' | 'callback_failed';

const CALLBACK_PLACEHOLDER = 'http://127.0.0.1:1455/auth/callback?code=…&state=…';

export interface SiwcBindDialogProps {
  open: boolean;
  onClose: () => void;
  onViewAuthFiles: () => void;
}

/**
 * SIWC bind: pick a Codex credential, authorize in the browser, then paste the loopback
 * callback that the server exchanges for the binding. The server owns the session, so a
 * close cancels it through the connection that started it. Each open remounts the dialog.
 */
export function SiwcBindDialog({ open, onClose, onViewAuthFiles }: SiwcBindDialogProps) {
  const { t, i18n } = useTranslation();
  const showNotification = useNotificationStore((state) => state.showNotification);
  const [targets, setTargets] = useState<AuthFileItem[] | null>(null);
  const [loadFailed, setLoadFailed] = useState(false);
  const [name, setName] = useState('');
  const [run, setRun] = useState(0);
  const [session, setSession] = useState<SiwcBindResponse>();
  const [phase, setPhase] = useState<Phase>('idle');
  const [callback, setCallback] = useState('');
  const [feedback, setFeedback] = useState<Feedback | ''>('');
  const [submitting, setSubmitting] = useState(false);
  const attemptRef = useRef<OAuthAttempt | null>(null);
  const startRef = useRef<ReturnType<typeof createSiwcStart<SiwcBindResponse>> | null>(null);
  const closeRef = useRef(onClose);
  useEffect(() => {
    closeRef.current = onClose;
  }, [onClose]);

  // A new connection or a logout invalidates the attempt and closes the dialog; a response
  // from the old connection never reaches the new one.
  useEffect(() => {
    let active = true;
    const connection = useAuthStore.getState();
    const isSameConnection = () => {
      const current = useAuthStore.getState();
      return (
        current.isAuthenticated === connection.isAuthenticated &&
        current.apiBase === connection.apiBase &&
        current.managementKey === connection.managementKey
      );
    };
    const unsubscribe = useAuthStore.subscribe(() => {
      if (!isSameConnection()) {
        active = false;
        attemptRef.current?.invalidate();
        startRef.current?.close();
        closeRef.current();
      }
    });
    authFilesApi
      .list()
      .then((response) => {
        if (!active || !isSameConnection()) return;
        const list = listSiwcTargets(response.files);
        setTargets(list);
        if (list.length === 1) setName(list[0].name);
      })
      .catch(() => {
        if (active) setLoadFailed(true);
      });
    return () => {
      active = false;
      unsubscribe();
    };
  }, []);

  useEffect(() => {
    if (!name) return;
    const attempts = createOAuthAttempts({
      setTimeout: (fn, delay) => window.setTimeout(fn, delay),
      clearTimeout: (timer) => window.clearTimeout(timer),
    });
    const attempt = attempts.begin(name);
    attemptRef.current = attempt;
    const api = captureSiwcBindingApi();
    const startup = createSiwcStart({
      start: () => {
        setPhase('starting');
        return api.start(name);
      },
      cancel: api.cancel,
      onResult: (result) => {
        if (!attempt.isCurrent()) return;
        const url = new URL(result.url);
        if (
          result.status !== 'ok' ||
          result.flow !== 'siwc_bind' ||
          !result.state ||
          !['https:', 'http:'].includes(url.protocol) ||
          !(result.expires_in > 0)
        ) {
          setPhase('error');
          return;
        }
        setSession(result);
        setPhase('wait');
        const expiresAt = Date.now() + result.expires_in * 1000;
        attempt.poll<{ status: 'wait' | 'ok' | 'error' | 'expired' }>(
          () =>
            Date.now() >= expiresAt
              ? Promise.resolve({ status: 'expired' as const })
              : oauthApi.getAuthStatus(result.state, attempt.signal),
          (polled) => {
            if (polled.status === 'wait') return true;
            setPhase(polled.status);
            if (polled.status === 'ok') {
              startup.complete();
              notifyAuthFilesChanged();
            }
            return false;
          },
          () => setPhase('error'),
          2000
        );
      },
      onError: () => {
        if (attempt.isCurrent()) setPhase('error');
      },
    });
    startRef.current = startup;
    return () => {
      attempt.invalidate();
      startup.close();
      if (attemptRef.current === attempt) attemptRef.current = null;
      if (startRef.current === startup) startRef.current = null;
    };
  }, [name, run]);

  const close = () => {
    attemptRef.current?.invalidate();
    startRef.current?.close();
    onClose();
  };

  /** Start over in place: the effect cleanup cancels the old server session. */
  const restart = () => {
    setSession(undefined);
    setCallback('');
    setFeedback('');
    setSubmitting(false);
    setPhase('idle');
    setRun((value) => value + 1);
  };

  const submit = async () => {
    const attempt = attemptRef.current;
    if (!session || !attempt?.isCurrent() || phase !== 'wait' || submitting) return;
    if (!isSiwcCallbackValid(callback, session.state)) {
      setFeedback('invalid_callback');
      return;
    }
    setSubmitting(true);
    setFeedback('');
    try {
      await oauthApi.submitCallback('codex', callback, attempt.signal);
      if (attempt.isCurrent()) setFeedback('submitted');
    } catch {
      if (attempt.isCurrent()) setFeedback('callback_failed');
    } finally {
      if (attempt.isCurrent()) setSubmitting(false);
    }
  };

  const copyLink = async (value: string) => {
    const copied = await copyToClipboard(value);
    showNotification(
      t(copied ? 'notification.link_copied' : 'notification.copy_failed'),
      copied ? 'success' : 'error'
    );
  };

  const selected = targets?.find((file) => file.name === name);
  const view =
    phase === 'idle'
      ? 'pick'
      : phase === 'ok'
        ? 'ok'
        : phase === 'error' || phase === 'expired'
          ? 'failed'
          : 'flow';

  const footer =
    view === 'pick' ? (
      <>
        <Button variant="secondary" onClick={close}>
          {t('common.cancel')}
        </Button>
        <Button disabled={!name} onClick={() => void startRef.current?.start()}>
          {t('siwc.start')}
        </Button>
      </>
    ) : view === 'flow' ? (
      <Button variant="secondary" onClick={close}>
        {t('siwc.cancel')}
      </Button>
    ) : view === 'failed' ? (
      <>
        <Button variant="secondary" onClick={close}>
          {t('common.close')}
        </Button>
        <Button onClick={restart}>{t('siwc.retry')}</Button>
      </>
    ) : (
      <>
        <Button variant="secondary" onClick={onViewAuthFiles}>
          {t('auth_login.view_auth_files')}
        </Button>
        <Button onClick={close}>{t('siwc.done')}</Button>
      </>
    );

  return (
    <Modal open={open} onClose={close} title={t('siwc.title')} footer={footer}>
      <div className={styles.dialog}>
        {view === 'pick' && (
          <>
            <p className={styles.hint}>{t('siwc.intro')}</p>
            <TargetPicker
              targets={targets}
              loadFailed={loadFailed}
              value={name}
              onChange={setName}
              expiresOf={(file) =>
                file.siwcExpiresAt ? formatDateTimeValue(file.siwcExpiresAt, i18n.language) : ''
              }
            />
          </>
        )}

        {view === 'flow' && (
          <>
            {selected && <TargetSummary file={selected} />}
            <section className={styles.step}>
              <h3 className={styles.stepTitle}>{t('siwc.step_open')}</h3>
              <div className={styles.urlBox}>
                <span className={styles.urlText} title={session?.url}>
                  {session?.url ?? t('siwc.starting')}
                </span>
              </div>
              <div className={styles.stepActions}>
                <Button
                  variant="secondary"
                  size="sm"
                  disabled={!session?.url}
                  onClick={() => session?.url && void copyLink(session.url)}
                >
                  {t('siwc.copy')}
                </Button>
                <Button
                  variant="secondary"
                  size="sm"
                  disabled={!session?.url}
                  onClick={() =>
                    session?.url && window.open(session.url, '_blank', 'noopener,noreferrer')
                  }
                >
                  {t('siwc.open')}
                  <IconExternalLink size={iconSize['14']} aria-hidden="true" />
                </Button>
              </div>
              <p className={styles.stepState} role="status" aria-live="polite">
                {phase === 'starting' ? t('siwc.starting') : phase === 'wait' ? t('siwc.wait') : ''}
              </p>
            </section>
            <section className={styles.step}>
              <h3 className={styles.stepTitle}>{t('siwc.callback')}</h3>
              <form
                className={styles.callbackForm}
                onSubmit={(event) => {
                  event.preventDefault();
                  void submit();
                }}
              >
                <Input
                  label={t('siwc.callback')}
                  hint={t('siwc.hint')}
                  value={callback}
                  placeholder={CALLBACK_PLACEHOLDER}
                  disabled={!session || phase !== 'wait'}
                  autoComplete="off"
                  spellCheck={false}
                  aria-invalid={feedback === 'invalid_callback' || feedback === 'callback_failed'}
                  onChange={(event) => {
                    setCallback(event.target.value);
                    if (feedback && feedback !== 'submitted') setFeedback('');
                  }}
                />
                <div>
                  <Button
                    type="submit"
                    variant="secondary"
                    size="sm"
                    loading={submitting}
                    disabled={!session || phase !== 'wait' || !callback.trim()}
                  >
                    {t('siwc.submit')}
                  </Button>
                </div>
              </form>
              {feedback && (
                <p
                  className={
                    feedback === 'submitted' ? styles.feedbackSuccess : styles.feedbackError
                  }
                  role={feedback === 'submitted' ? 'status' : 'alert'}
                >
                  {t(`siwc.${feedback}`)}
                </p>
              )}
            </section>
          </>
        )}

        {view === 'failed' && (
          <>
            {selected && <TargetSummary file={selected} />}
            <p className={styles.alert} role="alert">
              {t(`siwc.${phase}`)}
            </p>
          </>
        )}

        {view === 'ok' && (
          <>
            <p className={styles.success} role="status">
              {t('siwc.ok')}
            </p>
            {selected && (
              <p className={styles.hint}>
                {t('siwc.ok_hint', { name: deriveAuthFileIdentity(selected).primary })}
              </p>
            )}
          </>
        )}
      </div>
    </Modal>
  );
}

/* ---------- Target picker: native radios, token-styled rows ---------- */

function TargetPicker({
  targets,
  loadFailed,
  value,
  onChange,
  expiresOf,
}: {
  targets: AuthFileItem[] | null;
  loadFailed: boolean;
  value: string;
  onChange: (name: string) => void;
  expiresOf: (file: AuthFileItem) => string;
}) {
  const { t } = useTranslation();
  const groupName = useId();

  let content: ReactNode;
  if (loadFailed) {
    content = (
      <p className={styles.alert} role="alert">
        {t('siwc.load_failed')}
      </p>
    );
  } else if (!targets) {
    content = (
      <p className={styles.stepState} role="status">
        {t('siwc.loading')}
      </p>
    );
  } else if (targets.length === 0) {
    content = <p className={styles.hint}>{t('siwc.no_targets')}</p>;
  } else {
    content = (
      <div className={styles.options}>
        {targets.map((file) => {
          const identity = deriveAuthFileIdentity(file);
          const expires = expiresOf(file);
          const meta = [expires && t('siwc.expires', { date: expires }), identity.secondary]
            .filter(Boolean)
            .join(' · ');
          return (
            <label key={file.name} className={styles.option}>
              <input
                type="radio"
                className={styles.radioInput}
                name={groupName}
                value={file.name}
                checked={value === file.name}
                onChange={() => onChange(file.name)}
              />
              <span className={styles.optionText}>
                <span className={styles.optionName} title={identity.fullName}>
                  {identity.primary}
                </span>
                {meta && <span className={styles.optionMeta}>{meta}</span>}
              </span>
              <SiwcStatusBadge status={getSiwcBindingStatus(file)} withLabel={false} />
            </label>
          );
        })}
      </div>
    );
  }

  return (
    <fieldset className={styles.picker}>
      <legend className={styles.legend}>{t('siwc.target')}</legend>
      {content}
    </fieldset>
  );
}

/** Target after the authorization starts: one line, no more changes. */
function TargetSummary({ file }: { file: AuthFileItem }) {
  const { t } = useTranslation();
  const identity = deriveAuthFileIdentity(file);
  return (
    <div className={styles.summary}>
      <span className={styles.summaryLabel}>{t('siwc.target')}</span>
      <span className={styles.optionName} title={identity.fullName}>
        {identity.primary}
      </span>
      <SiwcStatusBadge status={getSiwcBindingStatus(file)} withLabel={false} />
    </div>
  );
}
