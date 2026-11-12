/**
 * One usage query, always showing the newest request.
 *
 * The React-free controller owns the request race, cancellation, and the
 * timeout. The hook binds it to React, so a range change or an unmount aborts
 * the request in flight and a slower older response can never replace a newer
 * one. Polling is optional; `intervalMs` of zero means a single load plus
 * manual refresh.
 */

import { useCallback, useEffect, useMemo, useSyncExternalStore } from 'react';

export interface UsageQueryState<T> {
  data: T | null;
  error: Error | null;
  loading: boolean;
  /** Local receipt time of the last successful load. */
  receivedAt: number | null;
}

export const EMPTY_USAGE_QUERY_STATE: UsageQueryState<never> = Object.freeze({
  data: null,
  error: null,
  loading: false,
  receivedAt: null,
});

export class UsageQueryTimeoutError extends Error {
  constructor(timeoutMs: number) {
    super(`Request timed out after ${Math.round(timeoutMs / 1000)}s`);
    this.name = 'UsageQueryTimeoutError';
  }
}

export interface UsageQueryControllerOptions<T> {
  load: (signal: AbortSignal) => Promise<T>;
  timeoutMs?: number;
  now?: () => number;
  setTimer?: (callback: () => void, ms: number) => unknown;
  clearTimer?: (timer: unknown) => void;
}

export interface UsageQueryController<T> {
  /** Replace the loader, e.g. when the caller's closure changes. */
  setLoad(load: (signal: AbortSignal) => Promise<T>): void;
  /** Start a request. Aborts the previous one; only the newest may commit. */
  run(): Promise<T>;
  /** Abort the request in flight and ignore anything still settling. */
  cancel(): void;
  subscribe(listener: () => void): () => void;
  getSnapshot(): UsageQueryState<T>;
}

const DEFAULT_TIMEOUT_MS = 30 * 1000;

const toError = (error: unknown): Error =>
  error instanceof Error ? error : new Error(String(error || 'Request failed'));

export function createUsageQueryController<T>({
  load,
  timeoutMs = DEFAULT_TIMEOUT_MS,
  now = Date.now,
  setTimer = (callback, ms) => setTimeout(callback, ms),
  clearTimer = (timer) => clearTimeout(timer as ReturnType<typeof setTimeout>),
}: UsageQueryControllerOptions<T>): UsageQueryController<T> {
  const listeners = new Set<() => void>();
  let state: UsageQueryState<T> = EMPTY_USAGE_QUERY_STATE;
  let generation = 0;
  let currentLoad = load;
  let inFlight: { generation: number; abort: () => void } | null = null;

  const emit = (patch: Partial<UsageQueryState<T>>) => {
    state = { ...state, ...patch };
    listeners.forEach((listener) => listener());
  };

  const cancel = () => {
    generation += 1;
    if (inFlight) {
      inFlight.abort();
      inFlight = null;
    }
    if (state.loading) emit({ loading: false });
  };

  const run = (): Promise<T> => {
    if (inFlight) {
      inFlight.abort();
      inFlight = null;
    }
    const runGeneration = generation + 1;
    generation = runGeneration;
    const controller = new AbortController();
    let timeoutId: unknown = null;
    emit({ loading: true, error: null });

    const bounded = new Promise<T>((resolve, reject) => {
      timeoutId = setTimer(() => {
        controller.abort();
        reject(new UsageQueryTimeoutError(timeoutMs));
      }, timeoutMs);
      currentLoad(controller.signal).then(resolve, reject);
    });

    const abort = () => {
      if (timeoutId !== null) clearTimer(timeoutId);
      controller.abort();
    };

    const promise = bounded.then(
      (data) => {
        if (runGeneration === generation) {
          if (timeoutId !== null) clearTimer(timeoutId);
          inFlight = null;
          emit({ data, error: null, loading: false, receivedAt: now() });
        }
        return data;
      },
      (error: unknown) => {
        if (runGeneration === generation) {
          if (timeoutId !== null) clearTimer(timeoutId);
          inFlight = null;
          emit({ error: toError(error), loading: false });
        }
        throw error;
      }
    );

    inFlight = { generation: runGeneration, abort };
    return promise;
  };

  return {
    setLoad(next) {
      currentLoad = next;
    },
    run,
    cancel,
    subscribe(listener) {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
    getSnapshot() {
      return state;
    },
  };
}

export interface UseUsageQueryOptions<T> {
  load: (signal: AbortSignal) => Promise<T>;
  /**
   * Identity of the request (range, filters, zone). A new key creates a new
   * controller, so the old request is aborted and its state is dropped.
   */
  queryKey: string;
  enabled?: boolean;
  /** Poll cadence in ms; zero or absent means one load and manual refresh. */
  intervalMs?: number;
  timeoutMs?: number;
}

export interface UseUsageQueryResult<T> extends UsageQueryState<T> {
  refresh: () => Promise<T>;
}

const noopSubscribe = () => () => {};
const emptySnapshot = () => EMPTY_USAGE_QUERY_STATE;

export function useUsageQuery<T>({
  load,
  queryKey,
  enabled = true,
  intervalMs = 0,
  timeoutMs,
}: UseUsageQueryOptions<T>): UseUsageQueryResult<T> {
  const controller = useMemo(
    () => (enabled ? createUsageQueryController<T>({ load, timeoutMs }) : null),
    // `queryKey` is the identity of one request. A new key must create a new
    // controller even when the caller rebuilds `load` every render; the effect
    // below swaps the loader without restarting the request.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [enabled, queryKey, timeoutMs]
  );

  useEffect(() => {
    controller?.setLoad(load);
  }, [controller, load]);

  useEffect(() => {
    if (!controller) return undefined;
    void controller.run().catch(() => undefined);
    if (intervalMs <= 0) return () => controller.cancel();
    const timer = setInterval(() => {
      void controller.run().catch(() => undefined);
    }, intervalMs);
    return () => {
      clearInterval(timer);
      controller.cancel();
    };
  }, [controller, intervalMs]);

  const state = useSyncExternalStore(
    controller ? controller.subscribe : noopSubscribe,
    controller ? controller.getSnapshot : (emptySnapshot as () => UsageQueryState<T>),
    emptySnapshot as () => UsageQueryState<T>
  ) as UsageQueryState<T>;

  const refresh = useCallback(
    () => (controller ? controller.run() : Promise.reject(new Error('Not connected'))),
    [controller]
  );

  return useMemo(() => ({ ...state, refresh }), [state, refresh]);
}
