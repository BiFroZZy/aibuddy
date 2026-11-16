import { useCallback, useState } from 'react';
import type { ApiClient } from '../api/client';
import type { Confirmation, Request, StepStatus } from '../api/types';
import { Badge, Empty, ErrorNotice, Heading, Loading, PayloadDetails } from './shared';
import { useMutation, useResource } from './hooks';
import { date, label } from './format';

export function PlanScreen({ api }: { api: ApiClient }) {
  const load = useCallback((s: AbortSignal) => api.progress(s), [api]);
  const r = useResource(load);
  const m = useMutation();
  const p = r.value;
  const groups = [...new Set(p?.steps.map(s => s.scenario_title) || [])];
  return (
    <>
      <Heading eyebrow="Ваш маршрут / Адаптация" title="Большой старт. Маленькими шагами.">
        Часть шагов можно завершить самостоятельно. Для остальных потребуется подтверждение
        наставника.
      </Heading>
      <ErrorNotice message={m.error} />
      <ErrorNotice message={r.error} retry={r.reload} />
      {r.loading ? (
        <Loading />
      ) : (
        p && (
          <>
            <div className="panel">
              <div className="item-meta">
                <h2>
                  {p.done} из {p.total} шагов
                </h2>
                <span className="small muted">{p.percent}% завершено</span>
              </div>
              <div
                className="progressbar"
                role="progressbar"
                aria-valuenow={p.done}
                aria-valuemin={0}
                aria-valuemax={p.total || 1}
                aria-label="Выполненные шаги"
              >
                <span style={{ width: p.percent + '%' }} />
              </div>
              <p className="quiet-note">{p.waiting.length} ожидают наставника</p>
            </div>
            <div className="notice">
              Следующий шаг:{' '}
              {p.next?.title ||
                (p.waiting.length ? 'Дождаться проверки наставником' : 'Все шаги завершены')}
            </div>
            {groups.map(g => (
              <section key={g}>
                <div className="section-head">
                  <h2>{g}</h2>
                </div>
                {p.steps
                  .filter(s => s.scenario_title === g)
                  .map(s => (
                    <article className="row" key={s.id}>
                      <span className="avatar">{s.status === 'done' ? '✓' : s.position}</span>
                      <div className="row-details">
                        <h3>{s.title}</h3>
                        <p>{s.description}</p>
                        <p className="quiet-note">
                          {s.confirm_by === 'mentor'
                            ? 'Подтверждает наставник'
                            : 'Самостоятельное выполнение'}
                        </p>
                      </div>
                      {s.status === 'done' ? (
                        <Badge status="done" />
                      ) : (
                        <label className="step-control">
                          <span className="sr-only">Статус: {s.title}</span>
                          <select
                            aria-label={'Статус: ' + s.title}
                            value={s.status}
                            disabled={m.busy}
                            onChange={e =>
                              void m.run(async () => {
                                await api.setStep(s.id, e.target.value as StepStatus);
                              }, r.reload)
                            }
                          >
                            {(
                              [
                                'not_started',
                                'in_progress',
                                s.confirm_by === 'mentor' ? 'waiting' : 'done',
                              ] as StepStatus[]
                            ).map(v => (
                              <option key={v} value={v}>
                                {v === 'waiting'
                                  ? 'На проверку наставнику'
                                  : v === 'done'
                                    ? 'Завершить'
                                    : v === 'in_progress'
                                      ? 'В работе'
                                      : 'Не начато'}
                              </option>
                            ))}
                          </select>
                          {s.status === 'waiting' && (
                            <span className="small muted">Ожидает подтверждения наставником</span>
                          )}
                        </label>
                      )}
                    </article>
                  ))}
              </section>
            ))}
            {!p.steps.length && <Empty>План пока не назначен. Обратитесь к наставнику.</Empty>}
          </>
        )
      )}
    </>
  );
}
export function MentorScreen({ api }: { api: ApiClient }) {
  const load = useCallback((s: AbortSignal) => api.confirmations(s), [api]);
  const r = useResource(load);
  const m = useMutation();
  const [search, setSearch] = useState('');
  const [selected, setSelected] = useState('');
  const list = (r.value?.confirmations || []).filter(c =>
    (c.intern_name + ' ' + c.step_title).toLowerCase().includes(search.toLowerCase())
  );
  const key = (c: Confirmation) => c.intern_id + '/' + c.step_id;
  const current = list.find(c => key(c) === selected) || list[0];
  return (
    <>
      <Heading eyebrow="Наставничество" title="Помочь сделать следующий шаг.">
        Подтвердите готовность или верните шаг стажёру в работу.
      </Heading>
      <div className="metrics-strip">
        <div className="metric">
          <label>Ожидают проверки</label>
          <strong>{r.value?.confirmations.length ?? '—'}</strong>
          <p>Шаги ваших стажёров</p>
        </div>
      </div>
      <input
        className="search"
        aria-label="Поиск шагов"
        placeholder="Стажёр или шаг"
        value={search}
        onChange={e => setSearch(e.target.value)}
      />
      <ErrorNotice message={r.error} retry={r.reload} />
      <ErrorNotice message={m.error} />
      {r.loading ? (
        <Loading />
      ) : current ? (
        <div className="split">
          <section className="queue" aria-label="Очередь шагов">
            {list.map(c => (
              <button
                className={'queue-item ' + (key(c) === key(current) ? 'selected' : '')}
                key={key(c)}
                onClick={() => setSelected(key(c))}
                aria-pressed={key(c) === key(current)}
              >
                <span className="small muted">{c.intern_name}</span>
                <h3>{c.step_title}</h3>
                <span className="small muted">На проверке с {date(c.since)}</span>
              </button>
            ))}
          </section>
          <section className="panel">
            <Badge status="waiting" />
            <h2>{current.step_title}</h2>
            <p>{current.intern_name}</p>
            <div className="evidence">
              <strong>Что проверить</strong>
              <p>Обсудите результат со стажёром перед подтверждением.</p>
            </div>
            <div className="decision-footer">
              <button
                className="btn"
                disabled={m.busy}
                onClick={() =>
                  void m.run(async () => {
                    await api.decideStep(current.intern_id, current.step_id, 'reject');
                  }, r.reload)
                }
              >
                Вернуть в работу
              </button>
              <button
                className="btn primary"
                disabled={m.busy}
                onClick={() =>
                  void m.run(async () => {
                    await api.decideStep(current.intern_id, current.step_id, 'approve');
                  }, r.reload)
                }
              >
                Подтвердить шаг
              </button>
            </div>
          </section>
        </div>
      ) : (
        !r.error && <Empty>Нет шагов, ожидающих проверки по выбранным условиям.</Empty>
      )}
    </>
  );
}
export function ApprovalsScreen({ api }: { api: ApiClient }) {
  const load = useCallback((s: AbortSignal) => api.approvals(s), [api]);
  const r = useResource(load);
  const m = useMutation();
  const [search, setSearch] = useState('');
  const [selected, setSelected] = useState('');
  const list = (r.value?.requests || []).filter(x =>
    (x.user_name + ' ' + x.id + ' ' + JSON.stringify(x.payload))
      .toLowerCase()
      .includes(search.toLowerCase())
  );
  const current = list.find(x => x.id === selected) || list[0];
  return (
    <>
      <Heading eyebrow="Руководитель / Согласования" title="Решения, которые двигают вперёд.">
        Входящие заявки сотрудников на отсутствие и доступы.
      </Heading>
      <div className="metrics-strip">
        <div className="metric">
          <label>На согласовании</label>
          <strong>{r.value?.requests.length ?? '—'}</strong>
          <p>Ожидают вашего решения</p>
        </div>
      </div>
      <input
        className="search"
        aria-label="Поиск входящих заявок"
        placeholder="Сотрудник, ресурс или ID заявки"
        value={search}
        onChange={e => setSearch(e.target.value)}
      />
      <ErrorNotice message={r.error} retry={r.reload} />
      <ErrorNotice message={m.error} />
      {r.loading ? (
        <Loading />
      ) : current ? (
        <div className="split">
          <section className="queue" aria-label="Входящие заявки">
            {list.map(x => (
              <button
                className={'queue-item ' + (x.id === current.id ? 'selected' : '')}
                aria-pressed={x.id === current.id}
                key={x.id}
                onClick={() => setSelected(x.id)}
              >
                <span className="small muted">{x.user_name}</span>
                <h3>
                  {x.kind === 'access'
                    ? 'Доступ к ' + x.payload.project
                    : label(x.payload.kind || 'day_off')}
                </h3>
                <Badge status={x.status} />
              </button>
            ))}
          </section>
          <Decision
            key={current.id}
            request={current}
            busy={m.busy}
            decide={(decision, comment) =>
              void m.run(async () => {
                await api.decideRequest(current.id, decision, comment);
              }, r.reload)
            }
          />
        </div>
      ) : (
        !r.error && <Empty>Входящих заявок по выбранным условиям нет.</Empty>
      )}
    </>
  );
}
function Decision({
  request: r,
  busy,
  decide,
}: {
  request: Request;
  busy: boolean;
  decide: (decision: 'approve' | 'reject', comment: string) => void;
}) {
  const [comment, setComment] = useState('');
  return (
    <section className="panel">
      <div className="item-meta">
        <span className="small muted">{r.id}</span>
        <Badge status={r.status} />
      </div>
      <h2>{r.kind === 'access' ? 'Запрос доступа' : 'Заявка на отсутствие'}</h2>
      <dl className="facts">
        <div>
          <dt>Сотрудник</dt>
          <dd>{r.user_name || r.user_id}</dd>
        </div>
        <div>
          <dt>Создана</dt>
          <dd>{date(r.created_at)}</dd>
        </div>
        <div>
          <dt>Согласующий</dt>
          <dd>{label(r.approver_role)}</dd>
        </div>
      </dl>
      <PayloadDetails payload={r.payload} />
      <label className="field">
        Комментарий сотруднику · необязательно
        <textarea value={comment} onChange={e => setComment(e.target.value)} maxLength={1000} />
      </label>
      <div className="decision-footer">
        {(['reject', 'approve'] as const).map(d => (
          <button
            key={d}
            className={'btn ' + (d === 'approve' ? 'primary' : '')}
            disabled={busy}
            onClick={() => decide(d, comment)}
          >
            {d === 'approve' ? 'Одобрить заявку' : 'Отклонить'}
          </button>
        ))}
      </div>
      <p className="quiet-note">Сотрудник увидит решение и комментарий в своих заявках.</p>
    </section>
  );
}
const eventLabels: Record<string, string> = {
  chat_message: 'Сообщения в чат',
  tool_called: 'Вызовы инструментов',
  tool_denied: 'Отказы в доступе',
  kb_miss: 'Поиск без результатов',
  action_proposed: 'Предложенные действия',
  action_confirmed: 'Подтверждённые действия',
  action_cancelled: 'Отменённые действия',
  action_failed: 'Неуспешные действия',
  step_status_changed: 'Изменения статусов шагов',
  step_decided: 'Решения наставника',
  search_kb: 'Поиск по базе знаний',
  get_my_progress: 'Просмотр прогресса',
  get_who_to_ask: 'Контакты для помощи',
  create_absence_request: 'Подготовка отсутствий',
  get_absence_status: 'Статусы отсутствий',
  request_access: 'Запросы доступа',
};
export function MetricsScreen({ api }: { api: ApiClient }) {
  const [days, setDays] = useState(30);
  const load = useCallback((s: AbortSignal) => api.metrics(days, s), [api, days]);
  const r = useResource(load);
  return (
    <>
      <Heading eyebrow="Аналитика / Обзор" title="Адаптация в фокусе.">
        Активность и использование инструментов без содержимого личных диалогов.
      </Heading>
      <div className="toolbar">
        <span className="small muted">
          Область данных: все события, доступные серверной метрике
        </span>
        <label>
          Период{' '}
          <select aria-label="Период метрик" value={days} onChange={e => setDays(+e.target.value)}>
            {[7, 30, 90].map(n => (
              <option key={n} value={n}>
                {n} дней
              </option>
            ))}
          </select>
        </label>
      </div>
      <ErrorNotice message={r.error} retry={r.reload} />
      {r.loading ? (
        <Loading />
      ) : (
        r.value && (
          <>
            <div className="metrics-strip">
              <div className="metric">
                <label>Активные пользователи</label>
                <strong>{r.value.active_users}</strong>
                <p>С {date(r.value.since)}</p>
              </div>
              <div className="metric">
                <label>События за период</label>
                <strong>{Object.values(r.value.events).reduce((a, b) => a + b, 0)}</strong>
                <p>Зарегистрированы сервером</p>
              </div>
              <div className="metric">
                <label>Вызовы инструментов</label>
                <strong>{Object.values(r.value.tools).reduce((a, b) => a + b, 0)}</strong>
                <p>За выбранный период</p>
              </div>
            </div>
            <div className="chart-grid">
              {(
                [
                  ['tools', 'Использование инструментов'],
                  ['events', 'События'],
                ] as const
              ).map(([key, title]) => (
                <section className="panel" key={key}>
                  <h2>{title}</h2>
                  {Object.entries(r.value![key]).map(([name, n]) => (
                    <div className="chart-row" key={name}>
                      <span>{eventLabels[name] || name}</span>
                      <div className="progressbar">
                        <span
                          style={{
                            width:
                              Math.round((n / Math.max(1, ...Object.values(r.value![key]))) * 100) +
                              '%',
                          }}
                        />
                      </div>
                      <strong>{n}</strong>
                    </div>
                  ))}
                  {!Object.keys(r.value![key]).length && <Empty>За период данных нет.</Empty>}
                </section>
              ))}
            </div>
            <p className="quiet-note">
              Синтетические демонстрационные данные. Метрики не ограничены одной командой.
            </p>
          </>
        )
      )}
    </>
  );
}
export function MaterialsScreen() {
  return (
    <>
      <Heading eyebrow="База знаний" title="Ответы под рукой.">
        Источники и помощь в знакомстве с процессами и инструментами.
      </Heading>
      <div className="notice">
        Инструкции в базе знаний — учебные примеры, а не внутренние регламенты Авито.
      </div>
      <div className="list">
        <article className="doc">
          <h2>Спросить помощника</h2>
          <p>
            Поиск по доступным вашей роли и команде материалам работает через чат. Там же можно
            уточнить контакты для помощи.
          </p>
          <a className="btn" href="#chat">
            Открыть чат →
          </a>
        </article>
        <article className="doc">
          <h2>AvitoTech Playbook</h2>
          <p>Открытый справочник инженерных команд. Внешний источник.</p>
          <a
            className="btn"
            href="https://github.com/avito-tech/playbook"
            target="_blank"
            rel="noopener noreferrer"
          >
            Открыть справочник ↗
          </a>
        </article>
      </div>
    </>
  );
}
