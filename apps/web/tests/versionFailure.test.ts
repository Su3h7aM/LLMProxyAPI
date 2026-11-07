import { describe, expect, spyOn, test } from 'bun:test';
import { apiClient } from '@/services/api/client';
import { versionApi } from '@/services/api/version';

describe('latest release failures', () => {
  test.each([404, 502])('rejects a %s response instead of reporting a release', async (status) => {
    const error = Object.assign(new Error(`HTTP ${status}`), { status });
    const get = spyOn(apiClient, 'get').mockRejectedValue(error);
    try {
      await expect(versionApi.checkLatest()).rejects.toBe(error);
      expect(get).toHaveBeenCalledTimes(1);
      expect(get).toHaveBeenCalledWith('/server/latest-version');
    } finally {
      get.mockRestore();
    }
  });
});
