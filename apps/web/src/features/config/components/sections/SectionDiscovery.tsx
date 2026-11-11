import { useTranslation } from 'react-i18next';
import { Collapsible } from '@/components/ui/Collapsible';
import { Input } from '@/components/ui/Input';
import type { ConfigSectionProps } from '../../types';
import { FieldAnchor, FieldGrid, FieldGroup, FieldShell, FieldStack, ToggleRow } from '../fields/FieldPrimitives';
import { StringListEditor } from '../blocks/StringListEditor';
import { getValidationMessage } from '../blocks/shared';

/** Local mDNS/DNS-SD discovery. Kept mounted inside connectivity so search can reveal every field. */
export function SectionDiscovery({
  values,
  validationErrors,
  disabled,
  onChange,
}: ConfigSectionProps) {
  const { t } = useTranslation();
  const discoveryServiceTypeError = getValidationMessage(
    t,
    validationErrors?.discoveryServiceType
  );

  return (
    <Collapsible
      label={t('config_management.visual.serverExtras.discoveryTitle')}
      hint={t('config_management.visual.serverExtras.discoveryHint')}
      defaultOpen={false}
    >
      <FieldStack>
        <FieldGrid>
          <FieldAnchor fieldId="discoveryEnabled">
            <ToggleRow
              title={t('config_management.visual.serverExtras.discoveryEnabled.label')}
              description={t('config_management.visual.serverExtras.discoveryEnabled.hint')}
              checked={values.discoveryEnabled}
              disabled={disabled}
              onChange={(discoveryEnabled) => onChange({ discoveryEnabled })}
            />
          </FieldAnchor>
          <FieldAnchor fieldId="discoveryAuthRequired">
            <ToggleRow
              title={t('config_management.visual.serverExtras.discoveryAuthRequired.label')}
              description={t('config_management.visual.serverExtras.discoveryAuthRequired.hint')}
              checked={values.discoveryAuthRequired}
              disabled={disabled}
              onChange={(discoveryAuthRequired) => onChange({ discoveryAuthRequired })}
            />
          </FieldAnchor>
          <FieldAnchor fieldId="discoveryAdvertiseManagement">
            <ToggleRow
              title={t('config_management.visual.serverExtras.discoveryAdvertiseManagement.label')}
              description={t(
                'config_management.visual.serverExtras.discoveryAdvertiseManagement.hint'
              )}
              checked={values.discoveryAdvertiseManagement}
              disabled={disabled}
              onChange={(discoveryAdvertiseManagement) =>
                onChange({ discoveryAdvertiseManagement })
              }
            />
          </FieldAnchor>
        </FieldGrid>
        <FieldGrid>
          <FieldAnchor fieldId="discoveryServiceName">
            <Input
              label={t('config_management.visual.serverExtras.discoveryServiceName.label')}
              hint={t('config_management.visual.serverExtras.discoveryServiceName.hint')}
              value={values.discoveryServiceName}
              disabled={disabled}
              onChange={(e) => onChange({ discoveryServiceName: e.target.value })}
            />
          </FieldAnchor>
          <FieldAnchor fieldId="discoveryServiceType">
            <Input
              label={t('config_management.visual.serverExtras.discoveryServiceType.label')}
              hint={t('config_management.visual.serverExtras.discoveryServiceType.hint')}
              placeholder="_ai-gateway._tcp"
              value={values.discoveryServiceType}
              disabled={disabled}
              error={discoveryServiceTypeError}
              onChange={(e) => onChange({ discoveryServiceType: e.target.value })}
            />
          </FieldAnchor>
        </FieldGrid>
        <FieldAnchor fieldId="discoverySubtypes">
          <FieldGroup
            title={t('config_management.visual.serverExtras.discoverySubtypes.label')}
            description={t('config_management.visual.serverExtras.discoverySubtypes.hint')}
          >
            <FieldShell
              label={t('config_management.visual.serverExtras.discoverySubtypes.label')}
              hint={t('config_management.visual.serverExtras.discoverySubtypes.hint')}
            >
              <StringListEditor
                value={values.discoverySubtypes}
                disabled={disabled}
                placeholder="_responses"
                inputAriaLabel={t(
                  'config_management.visual.serverExtras.discoverySubtypes.label'
                )}
                onChange={(discoverySubtypes) => onChange({ discoverySubtypes })}
              />
            </FieldShell>
          </FieldGroup>
        </FieldAnchor>
        <FieldAnchor fieldId="discoveryInterfacesInclude">
          <FieldGroup
            title={t('config_management.visual.serverExtras.discoveryInterfacesInclude.label')}
            description={t(
              'config_management.visual.serverExtras.discoveryInterfacesInclude.hint'
            )}
          >
            <FieldShell
              label={t('config_management.visual.serverExtras.discoveryInterfacesInclude.label')}
              hint={t('config_management.visual.serverExtras.discoveryInterfacesInclude.hint')}
            >
              <StringListEditor
                value={values.discoveryInterfacesInclude}
                disabled={disabled}
                placeholder="en*"
                inputAriaLabel={t(
                  'config_management.visual.serverExtras.discoveryInterfacesInclude.label'
                )}
                onChange={(discoveryInterfacesInclude) =>
                  onChange({ discoveryInterfacesInclude })
                }
              />
            </FieldShell>
          </FieldGroup>
        </FieldAnchor>
        <FieldAnchor fieldId="discoveryInterfacesExclude">
          <FieldGroup
            title={t('config_management.visual.serverExtras.discoveryInterfacesExclude.label')}
            description={t(
              'config_management.visual.serverExtras.discoveryInterfacesExclude.hint'
            )}
          >
            <FieldShell
              label={t('config_management.visual.serverExtras.discoveryInterfacesExclude.label')}
              hint={t('config_management.visual.serverExtras.discoveryInterfacesExclude.hint')}
            >
              <StringListEditor
                value={values.discoveryInterfacesExclude}
                disabled={disabled}
                placeholder="en*"
                inputAriaLabel={t(
                  'config_management.visual.serverExtras.discoveryInterfacesExclude.label'
                )}
                onChange={(discoveryInterfacesExclude) =>
                  onChange({ discoveryInterfacesExclude })
                }
              />
            </FieldShell>
          </FieldGroup>
        </FieldAnchor>
      </FieldStack>
    </Collapsible>
  );
}
