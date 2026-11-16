import { useEffect, useRef, useState } from 'react';
import { errorMessage } from '../api/client';
export function useResource<T>(load: (signal: AbortSignal) => Promise<T>, revision = 0) {
  const [value, setValue] = useState<T>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [retry, setRetry] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    setError('');
    setValue(undefined);
    load(controller.signal)
      .then(v => {
        if (!controller.signal.aborted) setValue(v);
      })
      .catch(e => {
        if (!controller.signal.aborted) setError(errorMessage(e));
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [load, revision, retry]);
  return { value, loading, error, reload: () => setRetry(n => n + 1) };
}
export function useMutation() {
  const lock = useRef(false);
  const mounted = useRef(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  async function run(task: () => Promise<void>, reconcile?: () => Promise<void> | void) {
    if (lock.current) return;
    lock.current = true;
    setBusy(true);
    setError('');
    try {
      await task();
    } catch (e) {
      if (mounted.current) setError(errorMessage(e));
    } finally {
      try {
        await reconcile?.();
      } catch {
        if (mounted.current)
          setError('Не удалось проверить результат. Обновите данные перед повторной отправкой.');
      }
      lock.current = false;
      if (mounted.current) setBusy(false);
    }
  }
  return { busy, error, setError, run };
}
