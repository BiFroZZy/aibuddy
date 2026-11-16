import type {
  Action,
  ChatResponse,
  Confirmation,
  Identity,
  Message,
  Metrics,
  Progress,
  Request,
  StepStatus,
} from './types';
export class ApiError extends Error {
  constructor(
    public status: number,
    message: string
  ) {
    super(message);
  }
}
export function errorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 401) return 'Сессия завершена. Выберите демо-профиль.';
    if (error.status === 403) return 'У вашей роли нет доступа к этой операции.';
    if (error.status === 404) return 'Объект не найден или уже обработан. Данные обновлены.';
    if (error.status === 409) return 'Состояние уже изменилось. Проверьте обновлённые данные.';
    if (error.status === 502)
      return 'Сервис временно недоступен. Проверьте состояние перед повторной отправкой.';
    if (error.status === 400) return error.message;
  }
  return 'Не удалось получить ответ. Проверьте соединение и обновите данные. Операция могла быть выполнена.';
}
export class ApiClient {
  private lifetime = new AbortController();
  constructor(
    private token: string,
    private expired: () => void,
    private base = import.meta.env.VITE_API_BASE_URL || ''
  ) {}
  dispose() {
    this.lifetime.abort();
  }
  async request<T>(path: string, body?: object, signal?: AbortSignal): Promise<T> {
    const controller = new AbortController();
    const abort = () => controller.abort();
    const sources = [this.lifetime.signal, signal].filter((s): s is AbortSignal => !!s);
    sources.forEach(s => s.addEventListener('abort', abort, { once: true }));
    if (sources.some(s => s.aborted)) controller.abort();
    const timeout = setTimeout(abort, 150000);
    try {
      const response = await fetch(this.base.replace(/\/$/, '') + path, {
        method: body === undefined ? 'GET' : 'POST',
        signal: controller.signal,
        headers: {
          Authorization: `Bearer ${this.token}`,
          ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
        },
        body: body === undefined ? undefined : JSON.stringify(body),
      });
      if (response.status === 401) this.expired();
      let value: unknown;
      try {
        value = await response.json();
      } catch {
        throw new ApiError(response.status || 502, 'Некорректный ответ сервера');
      }
      if (!response.ok)
        throw new ApiError(
          response.status,
          typeof value === 'object' && value && 'error' in value
            ? String(value.error)
            : 'Ошибка сервера'
        );
      return value as T;
    } finally {
      clearTimeout(timeout);
      sources.forEach(s => s.removeEventListener('abort', abort));
    }
  }
  me(signal?: AbortSignal) {
    return this.request<Identity>('/api/me', undefined, signal);
  }
  progress(signal?: AbortSignal) {
    return this.request<Progress>('/api/progress', undefined, signal);
  }
  setStep(id: string, status: StepStatus) {
    return this.request<Progress>(`/api/progress/steps/${id}/status`, { status });
  }
  actions(signal?: AbortSignal) {
    return this.request<{ actions: Action[] }>('/api/actions', undefined, signal);
  }
  requests(signal?: AbortSignal) {
    return this.request<{ requests: Request[] }>('/api/requests', undefined, signal);
  }
  action(id: string, decision: 'confirm' | 'cancel') {
    return this.request<Action>(`/api/actions/${id}/${decision}`, {});
  }
  chat(message: string, conversation_id?: string) {
    return this.request<ChatResponse>('/api/chat', {
      message,
      request_id: crypto.randomUUID(),
      ...(conversation_id ? { conversation_id } : {}),
    });
  }
  history(id: string, signal?: AbortSignal) {
    return this.request<{ messages: Message[] }>(
      `/api/conversations/${id}/messages`,
      undefined,
      signal
    );
  }
  confirmations(signal?: AbortSignal) {
    return this.request<{ confirmations: Confirmation[] }>(
      '/api/mentor/confirmations',
      undefined,
      signal
    );
  }
  decideStep(intern: string, step: string, decision: 'approve' | 'reject') {
    return this.request<{ status: string }>(
      `/api/mentor/interns/${intern}/steps/${step}/decision`,
      { decision }
    );
  }
  approvals(signal?: AbortSignal) {
    return this.request<{ requests: Request[] }>('/api/manager/requests', undefined, signal);
  }
  decideRequest(id: string, decision: 'approve' | 'reject', comment: string) {
    return this.request<Request>(`/api/manager/requests/${id}/decision`, { decision, comment });
  }
  metrics(days: number, signal?: AbortSignal) {
    return this.request<Metrics>(`/api/metrics?days=${days}`, undefined, signal);
  }
}
