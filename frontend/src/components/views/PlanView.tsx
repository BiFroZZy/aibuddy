import { useAppState } from '@/state/useAppState';
import { Select } from '@/components/ui/Select';

const TASKS = [
  'Познакомиться с наставником',
  'Представиться команде',
  'Получить доступ к репозиторию',
  'Настроить окружение',
  'Изучить процессы команды',
  'Обсудить цели стажировки',
  'Выбрать первую задачу',
  'Договориться о первом ревью',
];

const PHASES = [
  'Знакомство · Первая неделя',
  'Знакомство · Первая неделя',
  'Инструменты · Первая неделя',
  'Инструменты · Первая неделя',
  'Развитие · Первая неделя',
  'Развитие · Первая неделя',
  'Развитие · Первая неделя',
  'Развитие · Первая неделя',
];

const STATUS_OPTIONS = [
  { value: 'todo', label: 'Не начато' },
  { value: 'running', label: 'В работе' },
  { value: 'done', label: 'Готово' },
];

export function PlanView() {
  const { planStatuses, updateTaskStatus } = useAppState();

  const doneCount = planStatuses.filter((s) => s === 'done').length;
  const nextIdx = planStatuses.findIndex((s) => s !== 'done');
  const nextTaskText = nextIdx < 0 ? 'Все шаги завершены. Обсуди с наставником дальнейшие цели.' : `Следующий шаг: ${TASKS[nextIdx]}`;

  return (
    <section className="view secondary adaptation-plan" id="plan" style={{ width: '100%', maxWidth: '720px', margin: '0 auto', display: 'flex', flexDirection: 'column', gap: 'var(--space-6)' }}>
      <div className="plan-heading">
        <p className="eyebrow" style={{ display: 'block', marginBottom: 'var(--space-4)', fontSize: 'var(--text-sm)', fontWeight: 400, color: 'var(--muted)' }}>
          Твой маршрут
        </p>
        <h1 style={{ fontSize: 'clamp(28px, 3vw, 40px)', letterSpacing: '-1.2px', lineHeight: '1.2', fontWeight: 600, marginBottom: 'var(--space-4)' }}>
          Большой старт.<br />Маленькими шагами.
        </h1>
        <p className="intro" style={{ fontSize: 'var(--text-base)', lineHeight: 'var(--leading-body)', color: 'var(--fg-2)', maxWidth: 'none', margin: 0 }}>
          Демонстрационный чек-лист первой недели. Отмечай выполненное — прогресс сохранится в этом браузере.
        </p>
      </div>

      <div className="progressline" style={{ marginBottom: 'var(--space-6)' }}>
        <div className="track" style={{ height: '6px', background: 'var(--border-soft)', borderRadius: 'var(--radius-pill)', overflow: 'hidden' }}>
          <span
            id="progress"
            style={{
              display: 'block',
              height: '100%',
              background: 'var(--accent)',
              borderRadius: 'var(--radius-pill)',
              transition: 'width var(--motion-base) var(--ease-standard)',
              width: `${(doneCount / 8) * 100}%`,
            }}
          />
        </div>
        <small id="progressText" style={{ display: 'block', marginTop: 'var(--space-2)', fontSize: 'var(--text-xs)', color: 'var(--muted)' }}>
          {Math.round((doneCount / 8) * 100)}% · {doneCount} из 8 шагов
        </small>
      </div>

      <div className="notice" id="nextTask" style={{ padding: 'var(--space-3) var(--space-4)', background: 'var(--brand-tint)', border: '1px solid var(--brand-blue)', borderRadius: 'var(--radius-md)', fontSize: 'var(--text-sm)', color: 'var(--accent-active)' }}>
        {nextTaskText}
      </div>

      <div className="list" id="tasks" style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
        {TASKS.map((title, i) => (
          <article key={i} className="list-item plan-task" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 'var(--space-4)', padding: 'var(--space-3) var(--space-4)', border: '1px solid var(--border)', borderRadius: 'var(--radius-md)', background: 'var(--bg)' }}>
            <div className="details" style={{ flex: 1, minWidth: 0 }}>
              <strong style={{ display: 'block', fontSize: 'var(--text-sm)', color: 'var(--fg)', marginBottom: 2 }}>{title}</strong>
              <p style={{ margin: 0, fontSize: 'var(--text-xs)', color: 'var(--muted)' }}>{PHASES[i]}</p>
            </div>
            <Select
              value={planStatuses[i]}
              onValueChange={(value) => updateTaskStatus(i, value as 'todo' | 'running' | 'done')}
              options={STATUS_OPTIONS}
              aria-label={`Статус: ${title}`}
              className="plan-task-status"
            />
          </article>
        ))}
      </div>
    </section>
  );
}