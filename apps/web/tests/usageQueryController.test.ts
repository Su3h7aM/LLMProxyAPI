import { describe, expect, test } from 'bun:test';
import {
  createUsageQueryController,
  UsageQueryTimeoutError,
} from '@/features/usage/hooks/useUsageQuery';

const deferred = <T>() => {
  let resolve!: (value: T) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, resolve, reject };
};

const flush = () => new Promise((resolve) => setTimeout(resolve, 0));

describe('usage query controller', () => {
  test('only the newest request may commit a result', async () => {
    const first = deferred<string>();
    const second = deferred<string>();
    let calls = 0;
    const controller = createUsageQueryController<string>({
      load: () => {
        calls += 1;
        return calls === 1 ? first.promise : second.promise;
      },
      timeoutMs: 30_000,
      setTimer: () => 0,
      clearTimer: () => undefined,
    });

    const firstRun = controller.run();
    const secondRun = controller.run();
    expect(controller.getSnapshot().loading).toBe(true);

    first.resolve('old');
    await firstRun;
    await flush();
    expect(controller.getSnapshot()).toMatchObject({ data: null, loading: true });

    second.resolve('new');
    await secondRun;
    expect(controller.getSnapshot()).toMatchObject({ data: 'new', error: null, loading: false });
  });

  test('a new request aborts the one in flight', async () => {
    const signals: AbortSignal[] = [];
    const pending = deferred<string>();
    let calls = 0;
    const controller = createUsageQueryController<string>({
      load: (signal) => {
        signals.push(signal);
        calls += 1;
        return calls === 1 ? pending.promise : Promise.resolve('second');
      },
      timeoutMs: 30_000,
      setTimer: () => 0,
      clearTimer: () => undefined,
    });

    const firstRun = controller.run();
    await controller.run();
    expect(signals[0].aborted).toBe(true);
    pending.resolve('old');
    await firstRun;
    await flush();
    expect(controller.getSnapshot().data).toBe('second');
  });

  test('cancel aborts and drops a late result', async () => {
    const pending = deferred<string>();
    let signal: AbortSignal | null = null;
    const controller = createUsageQueryController<string>({
      load: (nextSignal) => {
        signal = nextSignal;
        return pending.promise;
      },
      timeoutMs: 30_000,
      setTimer: () => 0,
      clearTimer: () => undefined,
    });

    const run = controller.run();
    controller.cancel();
    expect(signal?.aborted).toBe(true);
    expect(controller.getSnapshot().loading).toBe(false);

    pending.resolve('late');
    await run;
    await flush();
    expect(controller.getSnapshot()).toMatchObject({ data: null, loading: false });
  });

  test('a failed request keeps its error until the next success', async () => {
    const failure = new Error('boom');
    let calls = 0;
    const controller = createUsageQueryController<string>({
      load: () => {
        calls += 1;
        return calls === 1 ? Promise.reject(failure) : Promise.resolve('ok');
      },
      timeoutMs: 30_000,
      setTimer: () => 0,
      clearTimer: () => undefined,
    });

    await expect(controller.run()).rejects.toBe(failure);
    expect(controller.getSnapshot()).toMatchObject({ data: null, error: failure, loading: false });

    await controller.run();
    expect(controller.getSnapshot()).toMatchObject({ data: 'ok', error: null, loading: false });
  });

  test('a timeout aborts the request and rejects with a typed error', async () => {
    const pending = deferred<string>();
    let timeout: (() => void) | null = null;
    let signal: AbortSignal | null = null;
    const controller = createUsageQueryController<string>({
      load: (nextSignal) => {
        signal = nextSignal;
        return pending.promise;
      },
      timeoutMs: 5_000,
      setTimer: (callback) => {
        timeout = callback;
        return 1;
      },
      clearTimer: () => undefined,
      now: () => 123,
    });

    const run = controller.run();
    expect(timeout).not.toBeNull();
    (timeout as () => void)();
    await expect(run).rejects.toBeInstanceOf(UsageQueryTimeoutError);
    expect(signal?.aborted).toBe(true);
    expect(controller.getSnapshot().error).toBeInstanceOf(UsageQueryTimeoutError);
  });
});
