/**
 * 认证状态管理
 * 同源：API 基址恒为页面源站加 v8 前缀，只保存管理密钥。
 */

import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { AuthState, LoginCredentials, ConnectionStatus } from '@/types';
import { STORAGE_KEY_AUTH } from '@/utils/constants';
import { AUTH_LOGIN_STATE_KEY, obfuscatedStorage } from '@/services/storage/secureStorage';
import { apiClient } from '@/services/api/client';
import { useConfigStore } from './useConfigStore';
import { useModelsStore } from './useModelsStore';
import { useQuotaStore } from './useQuotaStore';
import { getSameOriginApiBase } from '@/utils/connection';

interface AuthStoreState extends AuthState {
  connectionStatus: ConnectionStatus;

  // 操作
  login: (credentials: LoginCredentials) => Promise<void>;
  logout: () => void;
  checkAuth: () => Promise<boolean>;
  restoreSession: () => Promise<boolean>;
  /** 同步判断 restoreSession 是否会尝试自动登录（无副作用）。 */
  hasRestorableSession: () => boolean;
  updateServerVersion: (version: string | null, buildDate?: string | null) => void;
  updateServerPluginSupport: (supportsPlugin: boolean) => void;
}

let restoreSessionPromise: Promise<boolean> | null = null;

/**
 * 自动登录的前提：上次登录时勾选了记住密钥且未登出（isLoggedIn 标记），并且保存了密钥。
 * restoreSession 与 hasRestorableSession 共用此判断。
 */
export const shouldAttemptAutoLogin = (
  wasLoggedIn: boolean,
  managementKey: string
): boolean => wasLoggedIn && Boolean(managementKey);

export const useAuthStore = create<AuthStoreState>()(
  persist(
    (set, get) => ({
      // 初始状态
      isAuthenticated: false,
      apiBase: '',
      managementKey: '',
      rememberPassword: false,
      serverVersion: null,
      serverBuildDate: null,
      supportsPlugin: false,
      connectionStatus: 'disconnected',

      // 恢复会话并自动登录
      restoreSession: () => {
        if (restoreSessionPromise) return restoreSessionPromise;

        restoreSessionPromise = (async () => {
          const wasLoggedIn =
            typeof localStorage !== 'undefined' &&
            localStorage.getItem(AUTH_LOGIN_STATE_KEY) === 'true';
          const { managementKey, rememberPassword } = get();
          const resolvedBase = getSameOriginApiBase();
          const resolvedKey = managementKey || '';
          const resolvedRememberPassword = rememberPassword;

          set({
            apiBase: resolvedBase,
            managementKey: resolvedKey,
            rememberPassword: resolvedRememberPassword,
          });
          apiClient.setConfig({ managementKey: resolvedKey });

          if (shouldAttemptAutoLogin(wasLoggedIn, resolvedKey)) {
            try {
              await get().login({
                managementKey: resolvedKey,
                rememberPassword: resolvedRememberPassword,
              });
              return true;
            } catch (error) {
              console.warn('Auto login failed:', error);
              return false;
            }
          }

          return false;
        })();

        return restoreSessionPromise;
      },

      hasRestorableSession: () => {
        if (typeof localStorage === 'undefined') return false;
        const wasLoggedIn = localStorage.getItem(AUTH_LOGIN_STATE_KEY) === 'true';
        const { managementKey } = get();
        return shouldAttemptAutoLogin(wasLoggedIn, managementKey);
      },

      // 登录
      login: async (credentials) => {
        const managementKey = credentials.managementKey.trim();
        const rememberPassword = credentials.rememberPassword ?? get().rememberPassword ?? false;

        try {
          set({
            connectionStatus: 'connecting',
            serverVersion: null,
            serverBuildDate: null,
            supportsPlugin: false,
          });
          useConfigStore.getState().clearCache();
          useModelsStore.getState().clearCache();
          useQuotaStore.getState().clearQuotaCache();

          // 配置 API 客户端（同源基址由客户端自行确定）
          apiClient.setConfig({
            managementKey,
          });

          // 测试连接 - 获取配置
          await useConfigStore.getState().fetchConfig(true);

          // 登录成功
          set({
            isAuthenticated: true,
            apiBase: getSameOriginApiBase(),
            managementKey,
            rememberPassword,
            connectionStatus: 'connected',
          });
          if (typeof localStorage !== 'undefined') {
            if (rememberPassword) {
              localStorage.setItem(AUTH_LOGIN_STATE_KEY, 'true');
            } else {
              localStorage.removeItem(AUTH_LOGIN_STATE_KEY);
            }
          }
        } catch (error: unknown) {
          set({ connectionStatus: 'error' });
          throw error;
        }
      },

      // 登出
      logout: () => {
        restoreSessionPromise = null;
        apiClient.setConfig({ managementKey: '' });
        useConfigStore.getState().clearCache();
        useModelsStore.getState().clearCache();
        useQuotaStore.getState().clearQuotaCache();
        set({
          isAuthenticated: false,
          apiBase: '',
          managementKey: '',
          serverVersion: null,
          serverBuildDate: null,
          supportsPlugin: false,
          connectionStatus: 'disconnected',
        });
        if (typeof localStorage !== 'undefined') {
          localStorage.removeItem(AUTH_LOGIN_STATE_KEY);
        }
      },

      // 检查认证状态
      checkAuth: async () => {
        const { managementKey } = get();

        if (!managementKey) {
          return false;
        }

        try {
          // 重新配置客户端
          apiClient.setConfig({ managementKey });
          set({ supportsPlugin: false });

          // 验证连接
          await useConfigStore.getState().fetchConfig();

          set({
            isAuthenticated: true,
            connectionStatus: 'connected',
          });

          return true;
        } catch {
          set({
            isAuthenticated: false,
            connectionStatus: 'error',
            supportsPlugin: false,
          });
          return false;
        }
      },

      // 更新服务器版本
      updateServerVersion: (version, buildDate) => {
        set({
          serverVersion: version || null,
          serverBuildDate: buildDate || null,
        });
      },

      updateServerPluginSupport: (supportsPlugin) => {
        set({ supportsPlugin });
      },
    }),
    {
      name: STORAGE_KEY_AUTH,
      storage: createJSONStorage(() => ({
        getItem: (name) => {
          const data = obfuscatedStorage.getItem<AuthStoreState>(name);
          return data ? JSON.stringify(data) : null;
        },
        setItem: (name, value) => {
          obfuscatedStorage.setItem(name, JSON.parse(value));
        },
        removeItem: (name) => {
          obfuscatedStorage.removeItem(name);
        },
      })),
      partialize: (state) => ({
        apiBase: state.apiBase,
        ...(state.rememberPassword ? { managementKey: state.managementKey } : {}),
        rememberPassword: state.rememberPassword,
        serverVersion: state.serverVersion,
        serverBuildDate: state.serverBuildDate,
      }),
    }
  )
);

// 监听全局未授权事件
if (typeof window !== 'undefined') {
  window.addEventListener('unauthorized', () => {
    useAuthStore.getState().logout();
  });

  window.addEventListener('server-version-update', ((e: CustomEvent) => {
    const detail = e.detail || {};
    useAuthStore.getState().updateServerVersion(detail.version || null, detail.buildDate || null);
  }) as EventListener);

  window.addEventListener('server-plugin-support-update', ((e: CustomEvent) => {
    useAuthStore.getState().updateServerPluginSupport(e.detail?.supportsPlugin === true);
  }) as EventListener);
}
