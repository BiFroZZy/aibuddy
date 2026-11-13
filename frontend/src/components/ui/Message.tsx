import { useState } from 'react';
import { useAppState } from '@/state/useAppState';
import { Message as MessageType } from '@/state/types';
import { SourceCard } from './SourceCard';
import { RequestForm } from './RequestForm';
import { Button } from './Button';

interface MessageProps {
  message: MessageType;
  onActionClick: (action: () => void) => void;
  onFormSubmit: (formData: Record<string, string>) => void;
}

export function Message({ message, onActionClick, onFormSubmit }: MessageProps) {
  const { setView } = useAppState();
  const [cancelled, setCancelled] = useState(false);
  const isUser = message.role === 'user';

  return (
    <div className={`message ${isUser ? 'message--user' : ''}`} style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-2)', maxWidth: '85%', alignSelf: isUser ? 'flex-end' : 'flex-start', width: '100%' }}>
      {!isUser && (
        <div className="message-label" style={{ fontSize: 'var(--text-xs)', fontWeight: 500, color: 'var(--muted)', marginBottom: 'var(--space-1)', paddingLeft: 'var(--space-1)' }}>
          AI-buddy
        </div>
      )}
      <div
        className="message-content"
        style={{
          padding: 'var(--space-3) var(--space-4)',
          borderRadius: 'var(--radius-lg)',
          background: isUser ? 'var(--accent)' : 'var(--surface)',
          color: isUser ? 'var(--accent-on)' : 'var(--fg)',
          fontSize: 'var(--text-sm)',
          lineHeight: 'var(--leading-body)',
          wordWrap: 'break-word',
          boxShadow: 'var(--elev-flat)',
        }}
      >
        {message.content}
      </div>

      {message.sourceCard && (
        <SourceCard doc={message.sourceCard} onOpen={() => { setView('materials'); requestAnimationFrame(() => { const target = document.getElementById('doc-' + message.sourceCard!.key) as HTMLDetailsElement | null; if (target) { target.open = true; target.scrollIntoView({block:'center'}); target.querySelector('summary')?.focus(); } }); }} />
      )}

      {message.form && !cancelled && (
        <RequestForm
          form={message.form}
          onSubmit={(data) => onFormSubmit(data)}
          onCancel={() => setCancelled(true)}
        />
      )}

      {cancelled && <p role="status">Отменено. Заявка не отправлена.</p>}
      {message.actions && message.actions.length > 0 && (
        <div style={{ display: 'flex', gap: 'var(--space-2)', marginTop: 'var(--space-2)' }}>
          {cancelled && <p role="status">Отменено. Заявка не отправлена.</p>}
      {message.actions.map((action, i) => (
            <Button key={i} variant="text" size="sm" onClick={() => onActionClick(action.action)}>
              {action.label}
            </Button>
          ))}
        </div>
      )}
    </div>
  );
}