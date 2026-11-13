import { useAppState } from '@/state/useAppState';
import { Button } from '@/components/ui/Button';
import { Icon } from '@/components/icons/Icon';
import { AnimatedDetails } from '@/components/ui/AnimatedDetails';
import { sourceDocs } from '@/utils/intentRouter';

export function MaterialsView() {
  const { setView } = useAppState();

  const handleMentorClick = () => {
    setView('home');
    window.dispatchEvent(new CustomEvent('ask-mentor'));
  };

  return (
    <section className="view secondary" id="materials" style={{ width: '100%', maxWidth: '720px', margin: '0 auto', display: 'flex', flexDirection: 'column', gap: 'var(--space-6)' }}>
      <div style={{ textAlign: 'center' }}>
        <p className="eyebrow" style={{ display: 'block', marginBottom: 'var(--space-4)', fontSize: 'var(--text-sm)', fontWeight: 400, color: 'var(--muted)' }}>
          База знаний
        </p>
        <h1 style={{ fontSize: 'clamp(28px, 3vw, 40px)', letterSpacing: '-1.2px', lineHeight: '1.2', fontWeight: 600, marginBottom: 'var(--space-4)' }}>
          Ответы под рукой.
        </h1>
        <p className="intro" style={{ fontSize: 'var(--text-base)', lineHeight: 'var(--leading-body)', color: 'var(--fg-2)', maxWidth: '36ch', margin: '0 auto' }}>
          Материалы для знакомства с процессами и инструментами.
        </p>
      </div>

      <div className="notice" style={{ padding: 'var(--space-3) var(--space-4)', background: 'var(--surface)', border: '1px solid var(--border)', borderRadius: 'var(--radius-md)', fontSize: 'var(--text-sm)', color: 'var(--fg-2)' }}>
        Все инструкции ниже — учебные примеры для прототипа, а не внутренние регламенты Авито.
      </div>

      <AnimatedDetails className="team-context doc" style={{ border: '1px solid var(--border)', borderRadius: 'var(--radius-md)', background: 'var(--surface-warm)', overflow: 'hidden' }}>
        <summary style={{ padding: 'var(--space-3) var(--space-4)', fontSize: 'var(--text-sm)', fontWeight: 500, cursor: 'pointer', listStyle: 'none', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          Контекст моей команды · Backend Team
          <svg className="material-chevron" viewBox="0 0 24 24" aria-hidden="true"><path d="m6 9 6 6 6-6" /></svg>
        </summary>
        <div style={{ padding: '0 var(--space-4) var(--space-4)', fontSize: 'var(--text-sm)', lineHeight: 'var(--leading-body)', color: 'var(--fg-2)' }}>
          <p style={{ margin: 'var(--space-3) 0' }}>
            Роль: стажёр. Стек: Go, PostgreSQL, Redis. Наставник: Мария. Канал: #backend-interns. Первая задача: изучить обработчик /health и обсудить критерии готовности с наставником.
          </p>
          <p style={{ margin: 'var(--space-3) 0', color: 'var(--muted)', fontSize: 'var(--text-xs)' }}>
            Синтетический профиль. Доступны только учебные материалы этой команды.
          </p>
          <Button variant="text" size="sm" onClick={handleMentorClick} style={{ marginTop: 'var(--space-2)', padding: 0 }}>
            Контакт наставника →
          </Button>
        </div>
      </AnimatedDetails>

      <div className="list" style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)', marginTop: 'var(--space-4)' }}>
        {[
          {
            key: 'access' as const,
            title: 'Доступ к репозиторию · Учебная инструкция',
            buttonLabel: 'Подготовить заявку →',
            buttonAction: () => {
              setView('home');
              window.dispatchEvent(new CustomEvent('ask-prompt', { detail: 'Как получить доступ к репозиторию?' }));
            },
          },
          {
            key: 'absence' as const,
            title: 'Отсутствия · Учебная инструкция',
            buttonLabel: 'Оформить заявку →',
            buttonAction: () => {
              setView('home');
              window.dispatchEvent(new CustomEvent('ask-prompt', { detail: 'Как оформить отгул?' }));
            },
          },
          {
            key: 'readme' as const,
            title: 'Первые задачи · Учебный маршрут',
            buttonLabel: null,
            buttonAction: null,
          },
        ].map((doc) => (
          <AnimatedDetails id={'doc-' + doc.key} key={doc.key} className="doc" style={{ border: '1px solid var(--border)', borderRadius: 'var(--radius-md)', background: 'var(--surface-warm)', overflow: 'hidden' }}>
            <summary style={{ padding: 'var(--space-3) var(--space-4)', fontSize: 'var(--text-sm)', fontWeight: 500, cursor: 'pointer', listStyle: 'none', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              {doc.title}
              <svg className="material-chevron" viewBox="0 0 24 24" aria-hidden="true"><path d="m6 9 6 6 6-6" /></svg>
            </summary>
            <div style={{ padding: '0 var(--space-4) var(--space-4)', fontSize: 'var(--text-sm)', lineHeight: 'var(--leading-body)', color: 'var(--fg-2)' }}>
              <p style={{ margin: 'var(--space-3) 0' }}>{sourceDocs[doc.key].body}</p>
              {doc.buttonLabel && doc.buttonAction && (
                <Button variant="text" size="sm" onClick={doc.buttonAction} style={{ marginTop: 'var(--space-2)', padding: 0 }}>
                  {doc.buttonLabel}
                </Button>
              )}
            </div>
          </AnimatedDetails>
        ))}

        <article className="doc" style={{ border: '1px solid var(--border)', borderRadius: 'var(--radius-md)', background: 'var(--surface-warm)', padding: 'var(--space-4)' }}>
          <strong style={{ display: 'block', marginBottom: 'var(--space-2)', fontSize: 'var(--text-sm)', color: 'var(--fg)' }}>AvitoTech Playbook</strong>
          <p style={{ margin: '0 0 var(--space-3)', fontSize: 'var(--text-sm)', color: 'var(--fg-2)', lineHeight: 'var(--leading-body)' }}>
            Открытый справочник инженерных команд. Внешний источник; его содержимое не загружено в этот прототип.
          </p>
          <a
            href="https://github.com/avito-tech/playbook"
            target="_blank"
            rel="noopener"
            style={{ display: 'inline-flex', alignItems: 'center', gap: 'var(--space-2)', fontSize: 'var(--text-sm)', color: 'var(--accent-active)', textDecoration: 'none' }}
          >
            Открыть справочник
            <Icon name="arrow" size={14} />
          </a>
        </article>
      </div>
    </section>
  );
}