import { useAppState } from '@/state/useAppState';
import { ChangeEvent } from 'react';
import { Button } from '@/components/ui/Button';
import { EmptyState } from '@/components/ui/EmptyState';

export function RequestsView() {
  const { requests, serviceOffline, setServiceOffline, updateRequest } = useAppState();

  const handleServiceOfflineChange = (e: ChangeEvent<HTMLInputElement>) => {
    setServiceOffline(e.target.checked);
  };

  const handleRetry = (requestKey: string) => {
    updateRequest(requestKey, { status: 'Отправка', error: undefined, busy: true });
    // Simulate retry
    setTimeout(() => {
      if (serviceOffline) {
        updateRequest(requestKey, { status: 'Ошибка', error: 'Сервис недоступен. Заявка не отправлена. Включи сервис и повтори попытку.', busy: false });
      } else {
        const idx = requests.findIndex((r) => r.key === requestKey);
        const newId = `DEMO-${String(idx + 1).padStart(3, '0')}`;
        updateRequest(requestKey, { status: 'На согласовании', id: newId, error: '', busy: false });
      }
    }, 700);
  };

  return (
    <section className="view secondary" id="requests" style={{ width: '100%', maxWidth: '720px', margin: '0 auto', display: 'flex', flexDirection: 'column', gap: 'var(--space-6)' }}>
      <div style={{ textAlign: 'center' }}>
        <p className="eyebrow" style={{ display: 'block', marginBottom: 'var(--space-4)', fontSize: 'var(--text-sm)', fontWeight: 400, color: 'var(--muted)' }}>
          История действий
        </p>
        <h1 style={{ fontSize: 'clamp(28px, 3vw, 40px)', letterSpacing: '-1.2px', lineHeight: '1.2', fontWeight: 600, marginBottom: 'var(--space-4)' }}>
          Вопрос → действие.
        </h1>
        <p className="intro" style={{ fontSize: 'var(--text-base)', lineHeight: 'var(--leading-body)', color: 'var(--fg-2)', maxWidth: '36ch', margin: '0 auto' }}>
          Здесь сохраняются результаты демонстрационных обращений. Настоящие корпоративные сервисы не подключены.
        </p>
      </div>

      <details className="doc" style={{ marginBottom: 'var(--space-6)', border: '1px solid var(--border)', borderRadius: 'var(--radius-md)', background: 'var(--surface-warm)' }}>
        <summary style={{ padding: 'var(--space-3) var(--space-4)', fontSize: 'var(--text-sm)', fontWeight: 500, cursor: 'pointer', listStyle: 'none' }}>
          Проверить сбой сервиса · Демо
        </summary>
        <div style={{ padding: 'var(--space-4)' }}>
          <p style={{ margin: '0 0 var(--space-3)', fontSize: 'var(--text-sm)', color: 'var(--fg-2)' }}>
            Отключи имитацию сервиса, чтобы увидеть ошибку и повторить действие без дубликатов.
          </p>
          <label style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-2)', fontSize: 'var(--text-sm)', cursor: 'pointer' }}>
            <input
              type="checkbox"
              id="serviceOffline"
              checked={serviceOffline}
              onChange={handleServiceOfflineChange}
              style={{ width: '16px', height: '16px', accentColor: 'var(--accent)' }}
            />
            Имитировать недоступность сервиса
          </label>
        </div>
      </details>

      <div className="list" id="requestList" style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
        {requests.length === 0 ? (
          <EmptyState
            icon="box"
            title="Пока нет заявок"
            description="Оформи отгул или запроси доступ через buddy."
          />
        ) : (
          [...requests].reverse().map((r) => (
            <article key={r.key} className="list-item" style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)', padding: 'var(--space-4)', border: '1px solid var(--border)', borderRadius: 'var(--radius-md)', background: 'var(--bg)' }}>
              <div className="details" style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-1)' }}>
                <strong style={{ fontSize: 'var(--text-sm)', color: 'var(--fg)' }}>{r.title}</strong>
                <p style={{ margin: 0, fontSize: 'var(--text-xs)', color: 'var(--muted)' }}>{r.id || r.key} · {r.detail}</p>
                <p style={{ margin: 0, fontSize: 'var(--text-xs)', color: 'var(--muted)' }}>
                  {r.error || 'Следующий шаг: дождаться решения руководителя.'}
                </p>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-2)', flexWrap: 'wrap' }}>
                <span
                  className="status"
                  style={{
                    padding: '2px var(--space-2)',
                    borderRadius: 'var(--radius-pill)',
                    fontSize: 'var(--text-xs)',
                    fontWeight: 500,
                    background:
                      r.status === 'На согласовании'
                        ? 'var(--brand-tint)'
                        : r.status === 'Ошибка'
                        ? '#fef2f2'
                        : 'var(--surface)',
                    color:
                      r.status === 'На согласовании'
                        ? 'var(--accent-active)'
                        : r.status === 'Ошибка'
                        ? 'var(--danger)'
                        : 'var(--muted)',
                  }}
                >
                  {r.status}
                </span>
                {r.status === 'Ошибка' && (
                  <Button variant="text" size="sm" onClick={() => handleRetry(r.key)} disabled={r.busy} style={{ padding: 'var(--space-1) var(--space-3)' }}>
                    {r.busy ? 'Отправка…' : 'Повторить'}
                  </Button>
                )}
              </div>
            </article>
          ))
        )}
      </div>
    </section>
  );
}