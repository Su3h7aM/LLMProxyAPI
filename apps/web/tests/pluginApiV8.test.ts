import { afterEach, describe, expect, spyOn, test } from 'bun:test';
import { apiClient } from '@/services/api/client';
import { pluginsApi, pluginStoreApi } from '@/services/api/plugins';

const spies: Array<{ mockRestore(): void }> = [];
const mock = (method: 'get' | 'put' | 'patch' | 'post' | 'delete', value: unknown = {}) => {
  const spy = spyOn(apiClient, method).mockResolvedValue(value as never);
  spies.push(spy);
  return spy;
};
afterEach(() => {
  spies.splice(0).forEach((spy) => spy.mockRestore());
  apiClient.setConfig({ apiBase: '', managementKey: '' });
});

describe('plugin management v8 API contracts', () => {
  test('lists plugins and deletes one through the operational route', async () => {
    const get = mock('get', {
      plugins_enabled: true,
      plugins_dir: 'plugins',
      plugins: [
        {
          id: 'example',
          configured: true,
          registered: true,
          enabled: true,
          effective_enabled: true,
          supports_quota: true,
          quota_provider: 'Example',
        },
      ],
    });
    const remove = mock('delete', { status: 'ok', id: 'example', restart_required: true });
    const response = await pluginsApi.list();
    expect(get).toHaveBeenLastCalledWith('/plugins');
    expect(response.plugins[0]).toMatchObject({
      id: 'example',
      supportsQuota: true,
      quotaProvider: 'example',
    });
    expect(await pluginsApi.deletePlugin('example')).toMatchObject({
      id: 'example',
      restartRequired: true,
    });
    expect(remove).toHaveBeenLastCalledWith('/plugins/example');
  });

  test('plugin enabled is an instance config scalar, preserving sibling config', async () => {
    const get = mock('get', { enabled: false, custom: 'value' });
    const put = mock('put');
    const post = mock('post');
    expect(await pluginsApi.getConfig('example')).toEqual({ enabled: false, custom: 'value' });
    expect(get).toHaveBeenLastCalledWith('/config/plugins/configs/example');
    await pluginsApi.patchConfig('example', { custom: 'new' });
    expect(put).toHaveBeenLastCalledWith('/config/plugins/configs/example', {
      enabled: false,
      custom: 'new',
    });
    await pluginsApi.updateEnabled('example', false);
    expect(put).toHaveBeenLastCalledWith('/config/plugins/configs/example/enabled', false);
    await pluginStoreApi.list();
    expect(get).toHaveBeenLastCalledWith('/plugins/store');
    await pluginStoreApi.install('example', { sourceId: 'official', version: '1.2.3' });
    expect(post).toHaveBeenLastCalledWith(
      '/plugins/store/example/install?source=official&version=1.2.3',
      { version: '1.2.3' }
    );
  });

  test('the store list asks the backend to refresh release versions', async () => {
    const get = mock('get', { plugins: [] });
    await pluginStoreApi.list();
    expect(get).toHaveBeenLastCalledWith('/plugins/store');
    await pluginStoreApi.list({ refresh: true });
    expect(get).toHaveBeenLastCalledWith('/plugins/store?refresh=true');
    await pluginStoreApi.list({ refresh: false });
    expect(get).toHaveBeenLastCalledWith('/plugins/store');
  });

  test('plugin form clears delete fields and object edits replace instead of deep-merging', async () => {
    mock('get', {
      enabled: true,
      untouched: { future: 'preserved' },
      cleared: 'remove-me',
      edited: { keep: 1, obsolete: 2 },
    });
    const put = mock('put');
    await pluginsApi.patchConfig('example', { cleared: null, edited: { keep: 3 } });
    expect(put).toHaveBeenCalledWith('/config/plugins/configs/example', {
      enabled: true,
      untouched: { future: 'preserved' },
      edited: { keep: 3 },
    });
  });

  test('a missing plugin config reads as empty; other failures propagate', async () => {
    const get = mock('get');
    get.mockRejectedValue({ status: 404, apiCode: 'not_found' });
    expect(await pluginsApi.getConfig('example')).toEqual({});
    for (const error of [{ status: 401 }, { status: 404 }, new Error('network')]) {
      get.mockRejectedValue(error);
      await expect(pluginsApi.getConfig('example')).rejects.toBe(error);
    }
  });

  test('the config write aborts after a connection change, even when switched back', async () => {
    const first = { apiBase: '', managementKey: 'fixture' };
    apiClient.setConfig(first);
    const get = mock('get');
    const put = mock('put');
    get.mockImplementation(async () => {
      apiClient.setConfig({ apiBase: '', managementKey: 'other-fixture' });
      apiClient.setConfig(first);
      return {};
    });
    await expect(pluginsApi.patchConfig('example', { custom: 'new' })).rejects.toMatchObject({
      name: 'AbortError',
    });
    expect(put).not.toHaveBeenCalled();
  });
});
