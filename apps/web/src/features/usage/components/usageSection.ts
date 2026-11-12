/**
 * Props shared by every Usage page section below the shell.
 *
 * `query` is the filter state mapped to the contract (never null when
 * `enabled`); `queryKey` is its stable identity plus the shell refresh
 * generation, so one refresh action reloads every section. Each section
 * owns its latest-wins query, so one failing section never blanks the
 * page.
 */

import type { UsageQueryParams } from '@/types/usageStats';

export interface UsageSectionProps {
  query: UsageQueryParams;
  queryKey: string;
  enabled: boolean;
  tz: string;
}
