import { useCallback, useEffect, useRef, useState } from 'react';
import type { ApiClient } from '../api/client';
import type { Action, Conversation, Identity, Message, Progress } from '../api/types';
import { remember } from './model';
import { ActionCard } from './Actions';
import { Empty, ErrorNotice, Heading, Loading } from './shared';
import { useMutation } from './hooks';

export function ChatScreen({
  api,
  me,
  conversation,
  onRemember,
  onConversation,
  draft,
  onDraft,
}: {
  api: ApiClient;
  me: Identity;
  conversation?: string;
  onRemember: (list: Conversation[]) => void;
  onConversation: (id: string) => void;
  draft: string;
  onDraft: (v: string) => void;
}) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [actions, setActions] = useState<Action[]>([]);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState('');
  const [retry, setRetry] = useState(0);
  const [progress, setProgress] = useState<Progress>();
  const mutation = useMutation();
  const active = useRef(true);
  const sentId = useRef('');
  const end = useRef<HTMLDivElement>(null);
  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
    };
  }, []);
  const syncActions = useCallback(async () => {
    const value = await api.actions();
    if (active.current) setActions(value.actions || []);
  }, [api]);
  useEffect(() => {
    const controller = new AbortController();
    if (!conversation) {
      setMessages([]);
      setActions([]);
      api
        .actions(controller.signal)
        .then(value => {
          if (!controller.signal.aborted) setActions(value.actions || []);
        })
        .catch(() => {});
      return () => controller.abort();
    }
    if (sentId.current === conversation) {
      sentId.current = '';
      return;
    }
    setLoading(true);
    setLoadError('');
    setMessages([]);
    Promise.all([api.history(conversation, controller.signal), api.actions(controller.signal)])
      .then(([h, a]) => {
        if (!controller.signal.aborted) {
          setMessages(h.messages || []);
          setActions(a.actions || []);
        }
      })
      .catch(() => {
        if (!controller.signal.aborted)
          setLoadError('Не удалось загрузить диалог. Обновите данные или начните новый.');
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [api, conversation, retry]);
  useEffect(() => {
    const controller = new AbortController();
    if (me.role === 'intern')
      api
        .progress(controller.signal)
        .then(p => {
          if (!controller.signal.aborted) setProgress(p);
        })
        .catch(() => {});
    return () => controller.abort();
  }, [api, me.role]);
  useEffect(() => {
    end.current?.scrollIntoView?.({ block: 'nearest', behavior: 'smooth' });
  }, [messages, actions]);
  async function send() {
    const text = draft.trim();
    if (!text || loading || loadError) return;
    await mutation.run(
      async () => {
        onDraft('');
        const user: Message = { id: crypto.randomUUID(), role: 'user', content: text };
        setMessages(m => [...m, user]);
        const result = await api.chat(text, conversation);
        // Record successful new conversations even if their screen was left while awaiting the model.
        onRemember(remember(me.user_id, { id: result.conversation_id, title: text.slice(0, 55) }));
        if (!active.current) return;
        sentId.current = result.conversation_id;
        setMessages(m => [
          ...m,
          {
            id: crypto.randomUUID(),
            role: 'assistant',
            content: result.reply,
            tools: result.tools_used,
            actions: result.pending_actions,
          },
        ]);
        setActions(a => [
          ...a.filter(x => !result.pending_actions.some(p => p.id === x.id)),
          ...result.pending_actions,
        ]);
        onConversation(result.conversation_id);
      },
      async () => {
        if (active.current) await syncActions();
      }
    );
  }
  const decide = (id: string, decision: 'confirm' | 'cancel') =>
    void mutation.run(async () => {
      const a = await api.action(id, decision);
      if (active.current) setActions(list => [...list.filter(x => x.id !== id), a]);
      if (a.status === 'failed')
        mutation.setError('Действие не выполнено. Проверьте список заявок перед новым запросом.');
    }, syncActions);
  const visibleActionIds = new Set(messages.flatMap(m => m.actions?.map(a => a.id) || []));
  return (
    <section className="home-layout">
      <Heading eyebrow={me.team || 'Помощник по адаптации'} title="С чего начнём?">
        Помогу разобраться в процессах, найти следующий шаг и подготовить организационные действия.
      </Heading>
      <ErrorNotice message={loadError} retry={() => setRetry(v => v + 1)} />
      <ErrorNotice
        message={mutation.error}
        retry={() => {
          setRetry(v => v + 1);
          void syncActions().catch(() => {});
        }}
      />
      {loading && <Loading />}
      <div className="messages" role="log" aria-label="Диалог" aria-live="polite">
        {messages.map(m => (
          <div key={m.id}>
            <div className={'message ' + m.role}>
              <div className="message-label">{m.role === 'user' ? 'Вы' : 'AI-buddy'}</div>
              {m.content}
            </div>
            {!!m.tools?.length && <p className="quiet-note">Инструменты: {m.tools.join(', ')}</p>}
            {m.actions?.map(original => (
              <ActionCard
                key={original.id}
                action={actions.find(a => a.id === original.id) || original}
                busy={mutation.busy}
                onDecision={decide}
              />
            ))}
          </div>
        ))}
        <div ref={end} />
      </div>
      {actions.some(a => a.status === 'pending_confirmation' && !visibleActionIds.has(a.id)) && (
        <section>
          <h2>Ожидают вашего подтверждения</h2>
          <p className="quiet-note">Последние подготовленные действия пользователя.</p>
          {actions
            .filter(a => a.status === 'pending_confirmation' && !visibleActionIds.has(a.id))
            .map(a => (
              <ActionCard key={a.id} action={a} busy={mutation.busy} onDecision={decide} />
            ))}
        </section>
      )}
      {mutation.busy && <p role="status">Помощник обрабатывает запрос…</p>}
      <form
        className="composer"
        onSubmit={e => {
          e.preventDefault();
          void send();
        }}
      >
        <label className="sr-only" htmlFor="chat-message">
          Сообщение
        </label>
        <textarea
          id="chat-message"
          placeholder="Что хотите узнать?"
          maxLength={2000}
          value={draft}
          onChange={e => onDraft(e.target.value)}
          disabled={mutation.busy || loading}
        />
        <div className="composer-bottom">
          <span className="knowledge">{me.team} · Учебные данные</span>
          <button
            className="btn primary"
            disabled={!draft.trim() || mutation.busy || loading || !!loadError}
            type="submit"
          >
            Отправить
          </button>
        </div>
      </form>
      <p className="hint">{draft.length}/2000 · Заявки отправляются только после подтверждения.</p>
      {!conversation && !messages.length && (
        <div className="suggestions">
          {(me.role === 'intern'
            ? [
                'Какой мой следующий шаг?',
                'Оформи отгул на ГГГГ-ММ-ДД',
                'Нужен доступ к ресурсу на чтение',
              ]
            : ['К кому обратиться с вопросом?', 'Как устроены процессы команды?']
          ).map(t => (
            <button className="suggestion" key={t} onClick={() => onDraft(t)}>
              {t}
            </button>
          ))}
        </div>
      )}
      {me.role === 'intern' && (
        <div className="next-step">
          <div>
            <span className="small muted">Ваш следующий шаг</span>
            {progress ? (
              <p>
                {progress.next?.title ||
                  (progress.waiting.length
                    ? 'Дождаться подтверждения наставником'
                    : 'Все шаги завершены')}
              </p>
            ) : (
              <Empty>План доступен в разделе адаптации.</Empty>
            )}
          </div>
          <a className="btn" href="#plan">
            Открыть план →
          </a>
        </div>
      )}
      <p className="quiet-note">
        Для отсутствия укажите отгул, отпуск или больничный и даты. Для доступа — ресурс, уровень
        read/write/admin и обоснование. Не указывайте медицинские сведения.
      </p>
    </section>
  );
}
