import { useAppState } from '@/state/useAppState';
import { useRef, useEffect, FormEvent, KeyboardEvent } from 'react';
import { Button } from '@/components/ui/Button';
import { Textarea } from '@/components/ui/Textarea';
import { Icon } from '@/components/icons/Icon';
import { Message } from '@/components/ui/Message';
import { SuggestionChip } from '@/components/ui/SuggestionChip';
import { routeIntent } from '@/utils/intentRouter';
import { nextIndex } from '@/state/reducer';

const SUGGESTIONS = [
  { prompt: 'Как оформить отгул?', icon: 'calendar', title: 'Оформить отгул', desc: 'От запроса до результата' },
  { prompt: 'Как запустить сервис локально?', icon: 'book', title: 'Запустить проект', desc: 'Инструкция из базы команды' },
  { prompt: 'Как получить доступ к репозиторию?', icon: 'key', title: 'Запросить доступ', desc: 'Ресурс, цель, подтверждение' },
];

export function HomeView() {
  const {
    messages,
    activeChatId,
    welcomeVisible,
    planStatuses,
    addMessage,
    clearMessages,
    setWelcomeVisible,
    setView,
  } = useAppState();

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const promptRef = useRef<HTMLTextAreaElement>(null);
  const composerRef = useRef<HTMLFormElement>(null);

  const nextTaskIdx = nextIndex(planStatuses);
  const nextTaskText = nextTaskIdx < 0 ? 'Все шаги завершены' : `Следующий шаг: ${[
    'Познакомиться с наставником',
    'Представиться команде',
    'Получить доступ к репозиторию',
    'Настроить окружение',
    'Изучить процессы команды',
    'Обсудить цели стажировки',
    'Выбрать первую задачу',
    'Договориться о первом ревью',
  ][nextTaskIdx]}`;

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'instant' : 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    const textarea = promptRef.current;
    if (!textarea?.value.trim()) return;
    ask(textarea.value.trim());
    textarea.value = '';
    textarea.style.height = 'auto';
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      composerRef.current?.requestSubmit();
    }
  };

  const ask = (text: string) => {
    if (!text.trim()) return;
    setView('home');
    setWelcomeVisible(false);
    addMessage({ id: crypto.randomUUID(), role: 'user', content: text });

    const result = routeIntent(text, nextTaskText, setView);

    const assistantMessage: Parameters<typeof addMessage>[0] = {
      id: crypto.randomUUID(),
      role: 'assistant',
      content: result.message,
    };

    if (result.sourceCard) {
      assistantMessage.sourceCard = result.sourceCard;
    }

    if (result.showForm) {
      assistantMessage.form = {
        type: result.showForm,
        title: result.showForm === 'absence' ? 'Оформить отгул' : 'Запросить доступ к репозиторию',
        fields: result.showForm === 'absence'
          ? [
              { name: 'from', label: 'Первый день', type: 'date', value: '', required: true },
              { name: 'to', label: 'Последний день', type: 'date', value: '', required: true },
              { name: 'comment', label: 'Комментарий (необязательно)', type: 'textarea', value: '', required: false, maxLength: 200 },
            ]
          : [
              { name: 'resource', label: 'Учебный репозиторий', type: 'text', value: 'backend-onboarding', required: true, readonly: true },
              { name: 'reason', label: 'Обоснование', type: 'textarea', value: '', required: true, maxLength: 200 },
            ],
      };
    }

    if (result.actions) {
      assistantMessage.actions = result.actions;
    }

    addMessage(assistantMessage);
  };

  const handleSuggestionClick = (prompt: string) => {
    ask(prompt);
  };

  const handleNewChat = () => {
    clearMessages();
    setWelcomeVisible(true);
    if (promptRef.current) {
      promptRef.current.focus();
    }
  };

  // Listen for new-chat event from sidebar
  useEffect(() => {
    const handler = () => handleNewChat();
    window.addEventListener('new-chat', handler);
    return () => window.removeEventListener('new-chat', handler);
  });

  // Listen for ask-mentor event
  useEffect(() => {
    const handler = () => ask('Кто мой наставник?');
    window.addEventListener('ask-mentor', handler);
    return () => window.removeEventListener('ask-mentor', handler);
  });

  // Listen for ask-about event
  useEffect(() => {
    const handler = () => ask('Что умеет этот прототип?');
    window.addEventListener('ask-about', handler);
    return () => window.removeEventListener('ask-about', handler);
  });

  return (
    <section className="view" id="home" style={{ width: '100%', maxWidth: '720px', margin: '0 auto', display: 'flex', flexDirection: 'column', gap: 0, padding: 'var(--space-14) 0 var(--space-10)' }}>
      <div className="hero" id="welcome" hidden={!welcomeVisible} style={{ margin: '0 0 var(--space-9)', textAlign: 'center', maxWidth: 'none' }}>
        <p className="eyebrow" style={{ display: 'block', marginBottom: 'var(--space-4)', fontSize: 'var(--text-sm)', fontWeight: 400, color: 'var(--muted)' }}>
          Первая неделя в команде
        </p>
        <h1 style={{ fontSize: 'clamp(28px, 3vw, 40px)', letterSpacing: '-1.2px', lineHeight: '1.2', fontWeight: 600, marginBottom: 'var(--space-4)' }}>
          Алексей, с чего начнём?
        </h1>
        <p className="intro" style={{ fontSize: 'var(--text-base)', lineHeight: 'var(--leading-body)', color: 'var(--fg-2)', maxWidth: '36ch', margin: '0 auto' }}>
          Помогу с доступами, процессами и первыми задачами.
        </p>
      </div>

      <div key={activeChatId ?? 'new'} className="messages" id="messages" role="log" aria-live="polite" aria-label="Диалог с AI-buddy" style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 'var(--space-4)', minHeight: 0 }}>
        {messages.map((msg) => (
          <Message
            key={msg.id}
            message={msg}
            onActionClick={(action) => action()}
            onFormSubmit={(formData) => handleFormSubmit(formData, msg.id)}
          />
        ))}
        <div ref={messagesEndRef} />
      </div>

      <form className="composer" ref={composerRef} id="composer" onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)', marginTop: 'var(--space-6)' }}>
        <label htmlFor="prompt" className="sr-only">Вопрос помощнику</label>
        <Textarea
          ref={promptRef}
          id="prompt"
          placeholder="Что хочешь узнать?"
          rows={2}
          style={{
            padding: 'var(--space-3) var(--space-4)',
            borderRadius: 'var(--radius-lg)',
            border: '1px solid var(--border)',
            background: 'var(--bg)',
            resize: 'none',
            minHeight: '52px',
            maxHeight: '120px',
            fontSize: 'var(--text-sm)',
            lineHeight: 'var(--leading-body)',
          }}
          onKeyDown={handleKeyDown}
        />
        <div className="composer-bottom" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 'var(--space-3)' }}>
          <span className="knowledge" style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-2)', fontSize: 'var(--text-xs)', color: 'var(--muted)' }}>
            <Icon name="book" size={14} />
            Backend Team · Учебные данные
          </span>
          <Button type="submit" variant="primary" size="md" className="send" aria-label="Отправить сообщение" style={{ minWidth: '44px', minHeight: '44px', padding: 0, borderRadius: 'var(--radius-pill)' }}>
            <Icon name="arrow" size={18} />
          </Button>
        </div>
      </form>

      <div className="suggestions" style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 'var(--space-2)', marginTop: 'var(--space-4)' }}>
        {SUGGESTIONS.map((s) => (
          <SuggestionChip
            key={s.prompt}
            prompt={s.prompt}
            icon={s.icon}
            title={s.title}
            desc={s.desc}
            onClick={handleSuggestionClick}
          />
        ))}
      </div>

      <div className="next-step" style={{ display: 'flex', alignItems: 'flex-start', gap: 'var(--space-3)', marginTop: 'var(--space-10)', paddingTop: 'var(--space-6)', borderTop: '1px solid var(--border-soft)' }}>
        <span className="next-dot" aria-hidden="true" style={{ width: '8px', height: '8px', borderRadius: '50%', background: 'var(--accent)', flexShrink: 0, marginTop: '6px' }} />
        <div style={{ flex: 1, minWidth: 0 }}>
          <span className="next-label" style={{ display: 'block', fontSize: 'var(--text-sm)', fontWeight: 500, color: 'var(--fg-2)', marginBottom: 'var(--space-1)' }}>
            Твой следующий шаг
          </span>
          <p id="homeNext" style={{ margin: 0, fontSize: 'var(--text-sm)', color: 'var(--fg)', lineHeight: 'var(--leading-body)' }}>
            {nextTaskText}
          </p>
        </div>
        <Button variant="text" onClick={() => setView('plan')} style={{ marginLeft: 'var(--space-6)', whiteSpace: 'nowrap' }}>
          Открыть план →
        </Button>
      </div>
    </section>
  );
}

function handleFormSubmit(formData: Record<string, string>, messageId: string) {
  // This will be handled by the parent via event or context
  window.dispatchEvent(new CustomEvent('form-submit', { detail: { formData, messageId } }));
}