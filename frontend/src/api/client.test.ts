import { expect, test, vi } from 'vitest';
import { ApiClient, ApiError } from './client';
test('a failed mutation is never automatically retried and retains status', async () => {
  const fetcher = vi
    .fn()
    .mockResolvedValue(new Response(JSON.stringify({ error: 'conflict' }), { status: 409 }));
  vi.stubGlobal('fetch', fetcher);
  const api = new ApiClient('token', () => {});
  await expect(api.decideRequest('id', 'approve', '')).rejects.toMatchObject({ status: 409 });
  expect(fetcher).toHaveBeenCalledTimes(1);
});
test('unauthorized expires the session and malformed errors stay readable', async () => {
  const expired = vi.fn();
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('gateway', { status: 401 })));
  const api = new ApiClient('token', expired);
  await expect(api.me()).rejects.toBeInstanceOf(ApiError);
  expect(expired).toHaveBeenCalledOnce();
});

test.each([400, 403, 404, 409, 502])(
  'preserves HTTP %i without retrying the operation',
  async status => {
    const fetcher = vi
      .fn()
      .mockResolvedValue(new Response(JSON.stringify({ error: 'server response' }), { status }));
    vi.stubGlobal('fetch', fetcher);
    const api = new ApiClient('token', () => {});
    await expect(api.action('id', 'confirm')).rejects.toMatchObject({ status });
    expect(fetcher).toHaveBeenCalledOnce();
  }
);
test('session disposal aborts an in-flight read', async () => {
  let signal: AbortSignal | undefined;
  vi.stubGlobal(
    'fetch',
    vi.fn((_url: string, init: RequestInit) => {
      signal = init.signal as AbortSignal;
      return new Promise((_resolve, reject) =>
        signal?.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')))
      );
    })
  );
  const api = new ApiClient('token', () => {});
  const pending = api.me();
  api.dispose();
  await expect(pending).rejects.toMatchObject({ name: 'AbortError' });
  expect(signal?.aborted).toBe(true);
});
