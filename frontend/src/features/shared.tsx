import { date, label } from './format';
import type { ReactNode } from 'react';
import type { Payload } from '../api/types';
import { statusLabels } from './model';
export function ErrorNotice({ message, retry }: { message: string; retry?: () => void }) {
  return message ? (
    <div className="notice error-notice" role="alert">
      <span>{message}</span>
      {retry && (
        <button className="btn" onClick={retry}>
          Обновить данные
        </button>
      )}
    </div>
  ) : null;
}
export function Loading() {
  return (
    <p className="notice" role="status">
      Загрузка…
    </p>
  );
}
export function Empty({ children }: { children: ReactNode }) {
  return <div className="empty">{children}</div>;
}
export function Heading({
  eyebrow,
  title,
  children,
}: {
  eyebrow: string;
  title: string;
  children: ReactNode;
}) {
  return (
    <div className="page-head">
      <p className="eyebrow">{eyebrow}</p>
      <h1>{title}</h1>
      <p className="intro">{children}</p>
    </div>
  );
}
export function Badge({ status }: { status: string }) {
  const cls = ['waiting', 'pending_confirmation', 'pending_approval', 'running'].includes(status)
    ? 'wait'
    : ['done', 'approved'].includes(status)
      ? 'done'
      : ['failed', 'rejected', 'denied'].includes(status)
        ? 'bad'
        : '';
  return <span className={'badge ' + cls}>{statusLabels[status] || status}</span>;
}
export function PayloadDetails({ payload }: { payload: Payload }) {
  return (
    <dl className="facts">
      {payload.project && (
        <div>
          <dt>Ресурс</dt>
          <dd>{payload.project}</dd>
        </div>
      )}
      {payload.level && (
        <div>
          <dt>Уровень доступа</dt>
          <dd>
            {label(payload.level)} · {payload.level}
          </dd>
        </div>
      )}
      {payload.kind && (
        <div>
          <dt>Тип отсутствия</dt>
          <dd>{label(payload.kind)}</dd>
        </div>
      )}
      {payload.date_from && (
        <div>
          <dt>Даты</dt>
          <dd>
            {date(payload.date_from)} — {date(payload.date_to || payload.date_from)}
          </dd>
        </div>
      )}
      {(payload.reason || payload.justification) && (
        <div>
          <dt>Комментарий / обоснование</dt>
          <dd>{payload.reason || payload.justification}</dd>
        </div>
      )}
    </dl>
  );
}
export function Tabs({
  items,
  value,
  onChange,
  label: ariaLabel,
}: {
  items: [string, string][];
  value: string;
  onChange: (v: string) => void;
  label: string;
}) {
  return (
    <div className="tabs" role="tablist" aria-label={ariaLabel}>
      {items.map(([id, title]) => (
        <button
          key={id}
          type="button"
          role="tab"
          tabIndex={id === value ? 0 : -1}
          onKeyDown={e => {
            const index = items.findIndex(x => x[0] === value);
            let next = index;
            if (e.key === 'ArrowRight') next = (index + 1) % items.length;
            else if (e.key === 'ArrowLeft') next = (index + items.length - 1) % items.length;
            else if (e.key === 'Home') next = 0;
            else if (e.key === 'End') next = items.length - 1;
            else return;
            e.preventDefault();
            onChange(items[next][0]);
            document.getElementById(ariaLabel + '-' + items[next][0])?.focus();
          }}
          aria-selected={id === value}
          aria-controls="requests-panel"
          id={ariaLabel + '-' + id}
          onClick={() => onChange(id)}
        >
          {title}
        </button>
      ))}
    </div>
  );
}
