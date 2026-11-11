import { useId } from 'react';
import { useTranslation } from 'react-i18next';
import { Collapsible } from '@/components/ui/Collapsible';
import { Input } from '@/components/ui/Input';
import type { CodexLiveICEServerDraft } from '@/types/visualConfig';
import type { ConfigSectionProps } from '../../types';
import { CodexLiveICEServersEditor } from '../blocks/CodexLiveICEServersEditor';
import {
  Divider,
  FieldAnchor,
  FieldGrid,
  FieldGroup,
  FieldGroupHeading,
  FieldHint,
  FieldShell,
  FieldStack,
  ToggleRow,
} from '../fields/FieldPrimitives';
import { getValidationMessage } from '../blocks/shared';
import { useCallback } from 'react';

/** OAuth/file-backed provider behavior, independent of API-key provider configuration. */
export function SectionOAuthBehavior({
  values,
  validationErrors,
  disabled,
  onChange,
}: ConfigSectionProps) {
  const { t } = useTranslation();
  const iceFieldId = useId();
  const iceHintId = `${iceFieldId}-hint`;
  const iceErrorId = `${iceFieldId}-error`;
  const iceError = getValidationMessage(t, validationErrors?.codexLiveMediaRelayICEServers);

  const handleICEServersChange = useCallback(
    (codexLiveMediaRelayICEServers: CodexLiveICEServerDraft[]) =>
      onChange({ codexLiveMediaRelayICEServers }),
    [onChange]
  );

  const claudeTimezoneError = getValidationMessage(t, validationErrors?.claudeHeaderTimezone);
  const codexBootstrapTimeoutError = getValidationMessage(
    t,
    validationErrors?.codexStreamBootstrapTimeout
  );
  const poolIdleTimeoutError = getValidationMessage(
    t,
    validationErrors?.antigravityConnectionPoolIdleTimeout
  );
  const poolMaxIdleError = getValidationMessage(
    t,
    validationErrors?.antigravityConnectionPoolMaxIdleConnsPerHost
  );
  const relayMaxSessionsError = getValidationMessage(
    t,
    validationErrors?.codexLiveMediaRelayMaxSessions
  );
  const relayPublicIPError = getValidationMessage(
    t,
    validationErrors?.codexLiveMediaRelayPublicIP
  );
  const relayPortMinError = getValidationMessage(
    t,
    validationErrors?.codexLiveMediaRelayUDPPortMin
  );
  const relayPortMaxError = getValidationMessage(
    t,
    validationErrors?.codexLiveMediaRelayUDPPortMax
  );

  return (
    <Collapsible
      label={t('config_management.visual.additions.oauthTitle')}
      hint={t('config_management.visual.additions.oauthHint')}
      defaultOpen={false}
    >
      <FieldStack>
        <FieldGroupHeading title={t('config_management.visual.additions.claudeTitle')} />
        <FieldGrid>
          <FieldAnchor fieldId="claudeHeaderTimezone">
            <Input
              label={t('config_management.visual.additions.claudeHeaderTimezone.label')}
              hint={t('config_management.visual.additions.claudeHeaderTimezone.hint')}
              value={values.claudeHeaderTimezone}
              disabled={disabled}
              error={claudeTimezoneError}
              onChange={(e) => onChange({ claudeHeaderTimezone: e.target.value })}
            />
          </FieldAnchor>
          <FieldAnchor fieldId="claudeModelLevelCooling">
            <ToggleRow
              title={t('config_management.visual.additions.claudeModelLevelCooling.label')}
              description={t('config_management.visual.additions.claudeModelLevelCooling.hint')}
              checked={values.claudeModelLevelCooling}
              disabled={disabled}
              onChange={(claudeModelLevelCooling) => onChange({ claudeModelLevelCooling })}
            />
          </FieldAnchor>
          <FieldAnchor fieldId="claudeDisableCloakMode">
            <ToggleRow
              title={t('config_management.visual.additions.claudeDisableCloakMode.label')}
              description={t('config_management.visual.additions.claudeDisableCloakMode.hint')}
              checked={values.claudeDisableCloakMode}
              disabled={disabled}
              onChange={(claudeDisableCloakMode) => onChange({ claudeDisableCloakMode })}
            />
          </FieldAnchor>
          <FieldAnchor fieldId="claudeCodeDisableCloakingModelList">
            <ToggleRow
              title={t('config_management.visual.additions.claudeCodeDisableCloakingModelList.label')}
              description={t(
                'config_management.visual.additions.claudeCodeDisableCloakingModelList.hint'
              )}
              checked={values.claudeCodeDisableCloakingModelList}
              disabled={disabled}
              onChange={(claudeCodeDisableCloakingModelList) =>
                onChange({ claudeCodeDisableCloakingModelList })
              }
            />
          </FieldAnchor>
        </FieldGrid>

        <Divider />
        <FieldGroupHeading title={t('config_management.visual.additions.codexTitle')} />
        <FieldGrid>
          <FieldAnchor fieldId="codexDisableCloaking">
            <ToggleRow
              title={t('config_management.visual.additions.codexDisableCloaking.label')}
              description={t('config_management.visual.additions.codexDisableCloaking.hint')}
              checked={values.codexDisableCloaking}
              disabled={disabled}
              onChange={(codexDisableCloaking) => onChange({ codexDisableCloaking })}
            />
          </FieldAnchor>
          <FieldAnchor fieldId="codexModelLevelCooling">
            <ToggleRow
              title={t('config_management.visual.additions.codexModelLevelCooling.label')}
              description={t('config_management.visual.additions.codexModelLevelCooling.hint')}
              checked={values.codexModelLevelCooling}
              disabled={disabled}
              onChange={(codexModelLevelCooling) => onChange({ codexModelLevelCooling })}
            />
          </FieldAnchor>
          <FieldAnchor fieldId="codexStreamBootstrapBuffering">
            <ToggleRow
              title={t('config_management.visual.additions.codexStreamBootstrapBuffering.label')}
              description={t(
                'config_management.visual.additions.codexStreamBootstrapBuffering.hint'
              )}
              checked={values.codexStreamBootstrapBuffering}
              disabled={disabled}
              onChange={(codexStreamBootstrapBuffering) =>
                onChange({ codexStreamBootstrapBuffering })
              }
            />
          </FieldAnchor>
          <FieldAnchor fieldId="codexOptimizeMultiAgentV2">
            <ToggleRow
              title={t('config_management.visual.additions.codexOptimizeMultiAgentV2.label')}
              description={t('config_management.visual.additions.codexOptimizeMultiAgentV2.hint')}
              checked={values.codexOptimizeMultiAgentV2}
              disabled={disabled}
              onChange={(codexOptimizeMultiAgentV2) => onChange({ codexOptimizeMultiAgentV2 })}
            />
          </FieldAnchor>
          <FieldAnchor fieldId="codexOrphanDelegationCompatibility">
            <ToggleRow
              title={t('config_management.visual.additions.codexOrphanDelegationCompatibility.label')}
              description={t(
                'config_management.visual.additions.codexOrphanDelegationCompatibility.hint'
              )}
              checked={values.codexOrphanDelegationCompatibility}
              disabled={disabled}
              onChange={(codexOrphanDelegationCompatibility) =>
                onChange({ codexOrphanDelegationCompatibility })
              }
            />
          </FieldAnchor>
          <FieldAnchor fieldId="codexResponseSteering">
            <ToggleRow
              title={t('config_management.visual.additions.codexResponseSteering.label')}
              description={t('config_management.visual.additions.codexResponseSteering.hint')}
              checked={values.codexResponseSteering}
              disabled={disabled}
              onChange={(codexResponseSteering) => onChange({ codexResponseSteering })}
            />
          </FieldAnchor>
          <FieldAnchor fieldId="codexStreamBootstrapTimeout">
            <Input
              label={t('config_management.visual.additions.codexStreamBootstrapTimeout.label')}
              hint={t('config_management.visual.additions.codexStreamBootstrapTimeout.hint')}
              value={values.codexStreamBootstrapTimeout}
              disabled={disabled}
              error={codexBootstrapTimeoutError}
              onChange={(e) => onChange({ codexStreamBootstrapTimeout: e.target.value })}
            />
          </FieldAnchor>
        </FieldGrid>

        <Divider />
        <FieldGroupHeading title={t('config_management.visual.additions.antigravityTitle')} />
        <FieldGrid>
          <FieldAnchor fieldId="antigravityConnectionPoolEnabled">
            <ToggleRow
              title={t('config_management.visual.additions.antigravityConnectionPoolEnabled.label')}
              description={t(
                'config_management.visual.additions.antigravityConnectionPoolEnabled.hint'
              )}
              checked={values.antigravityConnectionPoolEnabled}
              disabled={disabled}
              onChange={(antigravityConnectionPoolEnabled) =>
                onChange({ antigravityConnectionPoolEnabled })
              }
            />
          </FieldAnchor>
          <FieldAnchor fieldId="antigravityConnectionPoolIdleTimeout">
            <Input
              label={t(
                'config_management.visual.additions.antigravityConnectionPoolIdleTimeout.label'
              )}
              hint={t(
                'config_management.visual.additions.antigravityConnectionPoolIdleTimeout.hint'
              )}
              value={values.antigravityConnectionPoolIdleTimeout}
              disabled={disabled}
              error={poolIdleTimeoutError}
              onChange={(e) =>
                onChange({ antigravityConnectionPoolIdleTimeout: e.target.value })
              }
            />
          </FieldAnchor>
          <FieldAnchor fieldId="antigravityConnectionPoolMaxIdleConnsPerHost">
            <Input
              label={t(
                'config_management.visual.additions.antigravityConnectionPoolMaxIdleConnsPerHost.label'
              )}
              hint={t(
                'config_management.visual.additions.antigravityConnectionPoolMaxIdleConnsPerHost.hint'
              )}
              value={values.antigravityConnectionPoolMaxIdleConnsPerHost}
              disabled={disabled}
              error={poolMaxIdleError}
              onChange={(e) =>
                onChange({ antigravityConnectionPoolMaxIdleConnsPerHost: e.target.value })
              }
            />
          </FieldAnchor>
        </FieldGrid>

        <Divider />
        <FieldGroupHeading title={t('config_management.visual.additions.xaiTitle')} />
        <FieldGrid>
          <FieldAnchor fieldId="xaiInjectXSearch">
            <ToggleRow
              title={t('config_management.visual.additions.xaiInjectXSearch.label')}
              description={t('config_management.visual.additions.xaiInjectXSearch.hint')}
              checked={values.xaiInjectXSearch}
              disabled={disabled}
              onChange={(xaiInjectXSearch) => onChange({ xaiInjectXSearch })}
            />
          </FieldAnchor>
        </FieldGrid>

        <Divider />
        <FieldGroupHeading title={t('config_management.visual.additions.liveRelayTitle')} />
        <FieldHint>{t('config_management.visual.additions.liveRelayHint')}</FieldHint>
        <FieldGrid>
          <FieldAnchor fieldId="codexLiveMediaRelayEnabled">
            <ToggleRow
              title={t('config_management.visual.additions.codexLiveMediaRelayEnabled.label')}
              description={t(
                'config_management.visual.additions.codexLiveMediaRelayEnabled.hint'
              )}
              checked={values.codexLiveMediaRelayEnabled}
              disabled={disabled}
              onChange={(codexLiveMediaRelayEnabled) =>
                onChange({ codexLiveMediaRelayEnabled })
              }
            />
          </FieldAnchor>
          <FieldAnchor fieldId="codexLiveMediaRelayDisablePrivateRemoteIPs">
            <ToggleRow
              title={t(
                'config_management.visual.additions.codexLiveMediaRelayDisablePrivateRemoteIPs.label'
              )}
              description={t(
                'config_management.visual.additions.codexLiveMediaRelayDisablePrivateRemoteIPs.hint'
              )}
              checked={values.codexLiveMediaRelayDisablePrivateRemoteIPs}
              disabled={disabled}
              onChange={(codexLiveMediaRelayDisablePrivateRemoteIPs) =>
                onChange({ codexLiveMediaRelayDisablePrivateRemoteIPs })
              }
            />
          </FieldAnchor>
          <FieldAnchor fieldId="codexLiveMediaRelayMaxSessions">
            <Input
              label={t('config_management.visual.additions.codexLiveMediaRelayMaxSessions.label')}
              hint={t('config_management.visual.additions.codexLiveMediaRelayMaxSessions.hint')}
              value={values.codexLiveMediaRelayMaxSessions}
              disabled={disabled}
              error={relayMaxSessionsError}
              onChange={(e) => onChange({ codexLiveMediaRelayMaxSessions: e.target.value })}
            />
          </FieldAnchor>
          <FieldAnchor fieldId="codexLiveMediaRelayPublicIP">
            <Input
              label={t('config_management.visual.additions.codexLiveMediaRelayPublicIP.label')}
              hint={t('config_management.visual.additions.codexLiveMediaRelayPublicIP.hint')}
              value={values.codexLiveMediaRelayPublicIP}
              disabled={disabled}
              error={relayPublicIPError}
              onChange={(e) => onChange({ codexLiveMediaRelayPublicIP: e.target.value })}
            />
          </FieldAnchor>
          <FieldAnchor fieldId="codexLiveMediaRelayUDPPortMin">
            <Input
              label={t('config_management.visual.additions.codexLiveMediaRelayUDPPortMin.label')}
              hint={t('config_management.visual.additions.codexLiveMediaRelayUDPPortMin.hint')}
              value={values.codexLiveMediaRelayUDPPortMin}
              disabled={disabled}
              error={relayPortMinError}
              onChange={(e) => onChange({ codexLiveMediaRelayUDPPortMin: e.target.value })}
            />
          </FieldAnchor>
          <FieldAnchor fieldId="codexLiveMediaRelayUDPPortMax">
            <Input
              label={t('config_management.visual.additions.codexLiveMediaRelayUDPPortMax.label')}
              hint={t('config_management.visual.additions.codexLiveMediaRelayUDPPortMax.hint')}
              value={values.codexLiveMediaRelayUDPPortMax}
              disabled={disabled}
              error={relayPortMaxError}
              onChange={(e) => onChange({ codexLiveMediaRelayUDPPortMax: e.target.value })}
            />
          </FieldAnchor>
        </FieldGrid>
        <FieldAnchor fieldId="codexLiveMediaRelayICEServers">
          <FieldGroup
            title={t('config_management.visual.additions.codexLiveMediaRelayICEServers.label')}
            description={t(
              'config_management.visual.additions.codexLiveMediaRelayICEServers.hint'
            )}
          >
            <FieldShell
              label={t('config_management.visual.additions.codexLiveMediaRelayICEServers.label')}
              hint={t('config_management.visual.additions.codexLiveMediaRelayICEServers.hint')}
              hintId={iceHintId}
              error={iceError}
              errorId={iceErrorId}
            >
              <CodexLiveICEServersEditor
                value={values.codexLiveMediaRelayICEServers}
                disabled={disabled}
                describedBy={`${iceHintId} ${iceErrorId}`}
                invalid={Boolean(iceError)}
                onChange={handleICEServersChange}
              />
            </FieldShell>
          </FieldGroup>
        </FieldAnchor>
      </FieldStack>
    </Collapsible>
  );
}
