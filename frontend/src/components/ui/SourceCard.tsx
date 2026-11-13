import { SourceCardData } from '@/state/types';
import { Button } from './Button';

interface SourceCardProps {
  doc: SourceCardData;
  onOpen: () => void;
}

export function SourceCard({ doc, onOpen }: SourceCardProps) {
  return (
    <aside className="source-card" style={{ marginTop: 'var(--space-3)', padding: 'var(--space-4)', border: '1px solid var(--border)', borderRadius: 'var(--radius-md)', background: 'var(--surface-warm)', fontSize: 'var(--text-sm)', lineHeight: '1.6' }}>
      <strong style={{ display: 'block', marginBottom: 'var(--space-2)', color: 'var(--fg)' }}>{doc.title}</strong>
      <blockquote style={{ margin: 'var(--space-2) 0', padding: 0, color: 'var(--fg-2)', fontStyle: 'italic', borderLeft: '2px solid var(--brand-blue)', paddingLeft: 'var(--space-3)' }}>
        «{doc.quote}»
      </blockquote>
      <p style={{ margin: 'var(--space-2) 0 0', color: 'var(--muted)', fontSize: 'var(--text-xs)' }}>Учебный источник · Backend Team</p>
      <Button variant="text" size="sm" onClick={onOpen} style={{ marginTop: 'var(--space-3)', padding: 0 }}>
        Открыть материал →
      </Button>
    </aside>
  );
}