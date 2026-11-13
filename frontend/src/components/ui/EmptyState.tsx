import { Icon } from '@/components/icons/Icon';

interface EmptyStateProps {
  icon: string;
  title: string;
  description: string;
}

export function EmptyState({ icon, title, description }: EmptyStateProps) {
  return (
    <div className="empty" style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', padding: 'var(--space-12) var(--space-6)', textAlign: 'center', color: 'var(--muted)' }}>
      <Icon name={icon} size={48} style={{ marginBottom: 'var(--space-4)', opacity: 0.5 }} />
      <p style={{ margin: '0 0 var(--space-2)', fontSize: 'var(--text-sm)', fontWeight: 500, color: 'var(--fg-2)' }}>{title}</p>
      <p style={{ margin: 0, fontSize: 'var(--text-sm)', lineHeight: 'var(--leading-body)' }}>{description}</p>
    </div>
  );
}