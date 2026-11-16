import { useCallback, useState } from 'react';
import type { ApiClient } from '../api/client';
import type { Action, Request } from '../api/types';
import { Badge, Empty, ErrorNotice, Heading, Loading, PayloadDetails, Tabs } from './shared';
import { useMutation, useResource } from './hooks';
import { date, label } from './format';

export function ActionCard({
  action,
  busy,
  onDecision,
}: {
  action: Action;
  busy: boolean;
  onDecision: (id: string, decision: 'confirm' | 'cancel') => void;
}) {
  return (
    <article className="request-card" data-testid={'action-' + action.id}>
      <div className="item-meta">
        <span className="small muted">{date(action.created_at)}</span>
        <Badge status={action.status} />
      </div>
      <h3>
        {action.summary ||
          (action.tool === 'request_access'
            ? 'Запрос доступа'
            : action.tool === 'create_absence_request'
              ? 'Заявка на отсутствие'
              : action.tool)}
      </h3>
      <PayloadDetails payload={action.args} />
      {action.status === 'pending_confirmation' && (
        <>
          <p className="quiet-note">Отправка создаст заявку, а не выдаст одобрение.</p>
          <div className="actions">
            <button
              className="btn primary"
              disabled={busy}
              onClick={() => onDecision(action.id, 'confirm')}
            >
              Подтвердить отправку
            </button>
            <button className="btn" disabled={busy} onClick={() => onDecision(action.id, 'cancel')}>
              Отменить
            </button>
          </div>
        </>
      )}
      {action.status === 'done' && (
        <p className="quiet-note">
          Заявка создана. Решение смотрите в «Моих заявках».{' '}
          {action.external_id && <>ID: {action.external_id}</>}
        </p>
      )}
      {action.status === 'failed' && (
        <p role="status">
          Действие не завершено. Проверьте список заявок перед новым запросом через чат.
        </p>
      )}
      {action.status === 'cancelled' && <p className="quiet-note">Отправка не выполнялась.</p>}
      {action.status === 'denied' && (
        <p className="quiet-note">Операция недоступна для вашей роли.</p>
      )}
    </article>
  );
}
export function RequestCard({ request: r }: { request: Request }) {
  return (
    <article className="request-card">
      <div className="item-meta">
        <span className="small muted">
          {r.id} · {date(r.created_at)}
        </span>
        <Badge status={r.status} />
      </div>
      <h3>
        {r.kind === 'access' ? 'Доступ к ' + r.payload.project : label(r.payload.kind || 'day_off')}
      </h3>
      <PayloadDetails payload={r.payload} />
      <p className="quiet-note">Согласующий: {label(r.approver_role)}</p>
      {r.status === 'pending_approval' ? (
        <p className="quiet-note">Следующий шаг — решение согласующего.</p>
      ) : (
        <div className="comment">
          <strong className="small">Комментарий согласующего</strong>
          <p>{r.decision_comment || 'Без комментария'}</p>
          <p className="quiet-note">Решение от {date(r.decided_at)}</p>
        </div>
      )}
    </article>
  );
}
export function RequestsScreen({ api, onNew }: { api: ApiClient; onNew: () => void }) {
  const [tab, setTab] = useState('requests');
  const [filter, setFilter] = useState('all');
  const [search, setSearch] = useState('');
  const load = useCallback(
    async (signal: AbortSignal) => {
      const [a, r] = await Promise.all([api.actions(signal), api.requests(signal)]);
      return { actions: a.actions || [], requests: r.requests || [] };
    },
    [api]
  );
  const resource = useResource(load);
  const mutation = useMutation();
  const decide = (id: string, decision: 'confirm' | 'cancel') =>
    void mutation.run(async () => {
      const result = await api.action(id, decision);
      if (result.status === 'failed')
        mutation.setError('Действие не выполнено. Проверьте обновлённый список заявок.');
    }, resource.reload);
  const actions = resource.value?.actions || [];
  const requests = resource.value?.requests || [];
  const matches = (v: Action | Request) =>
    (filter === 'all' || v.status === filter) &&
    JSON.stringify('args' in v ? v.args : v.payload)
      .toLocaleLowerCase()
      .includes(search.toLocaleLowerCase());
  return (
    <>
      <Heading eyebrow="Личное пространство / Заявки и действия" title="От запроса до результата.">
        Следите за решениями и подтверждайте подготовленные помощником действия перед отправкой.
      </Heading>
      <div className="toolbar">
        <Tabs
          label="Раздел заявок"
          value={tab}
          items={[
            ['requests', 'Заявки'],
            ['actions', 'Действия'],
          ]}
          onChange={v => {
            setTab(v);
            setFilter('all');
          }}
        />
        <button className="btn primary" onClick={onNew}>
          Новая заявка +
        </button>
      </div>
      <div className="toolbar">
        <Tabs
          label="Статус"
          value={filter}
          onChange={setFilter}
          items={
            tab === 'requests'
              ? [
                  ['all', 'Все заявки'],
                  ['pending_approval', 'На согласовании'],
                  ['approved', 'Одобрены'],
                  ['rejected', 'Отклонены'],
                ]
              : [
                  ['all', 'Все действия'],
                  ['pending_confirmation', 'Ожидают подтверждения'],
                  ['running', 'Выполняются'],
                  ['done', 'Выполнены'],
                  ['failed', 'Ошибки'],
                  ['cancelled', 'Отменённые'],
                  ['denied', 'Отказы'],
                ]
          }
        />
        <input
          className="search"
          aria-label="Поиск заявок"
          placeholder="Ресурс или комментарий"
          value={search}
          onChange={e => setSearch(e.target.value)}
        />
      </div>
      <p className="quiet-note">
        Показаны последние 50 {tab === 'requests' ? 'заявок' : 'действий'}. Фильтры применяются к
        загруженному списку.
      </p>
      <ErrorNotice message={mutation.error} />
      <ErrorNotice message={resource.error} retry={resource.reload} />
      {resource.loading ? (
        <Loading />
      ) : (
        <section
          role="tabpanel"
          id="requests-panel"
          aria-label={tab === 'requests' ? 'Заявки' : 'Действия'}
          className="list"
        >
          {tab === 'requests'
            ? requests.filter(matches).map(r => <RequestCard key={r.id} request={r} />)
            : actions
                .filter(matches)
                .map(a => (
                  <ActionCard key={a.id} action={a} busy={mutation.busy} onDecision={decide} />
                ))}
          {!resource.error && !(tab === 'requests' ? requests : actions).filter(matches).length && (
            <Empty>По выбранным условиям ничего нет.</Empty>
          )}
        </section>
      )}
    </>
  );
}
