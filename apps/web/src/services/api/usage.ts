import { apiClient } from './client';

const USAGE_FLAG_TIMEOUT_MS = 15 * 1000;
const USAGE_FLAG_PATH = '/config/observability/usage/usage-statistics-enabled';

export const usageApi = {
  /**
   * Whether the server records per-request usage statistics. Null when the backend
   * does not expose the flag. The flag lives in the v8 config tree; there is no
   * flat usage endpoint on v8.
   */
  getStatisticsEnabled: async (): Promise<boolean | null> => {
    try {
      const payload = await apiClient.get<unknown>(USAGE_FLAG_PATH, {
        timeout: USAGE_FLAG_TIMEOUT_MS,
      });
      return typeof payload === 'boolean' ? payload : null;
    } catch (error: unknown) {
      const status = (error as { status?: unknown } | null)?.status;
      if (status === 404) return null;
      throw error;
    }
  },
};
