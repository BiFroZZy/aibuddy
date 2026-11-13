import { forwardRef } from 'react';
import { Icon } from '@/components/icons/Icon';

interface SuggestionChipProps {
  prompt: string;
  icon: string;
  title: string;
  desc: string;
  onClick: (prompt: string) => void;
  className?: string;
  style?: React.CSSProperties;
}

export const SuggestionChip = forwardRef<HTMLButtonElement, SuggestionChipProps>(
  ({ prompt, icon, title, desc, onClick, className = '', style, ...props }, ref) => {
    return (
      <button
        ref={ref}
        className={`suggestion ${className}`}
        onClick={() => onClick(prompt)}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 'var(--space-3)',
          padding: 'var(--space-3) var(--space-3)',
          borderRadius: 'var(--radius-md)',
          border: '1px solid var(--border)',
          background: 'var(--bg)',
          cursor: 'pointer',
          textAlign: 'left',
          transition: 'background var(--motion-fast) var(--ease-standard), border-color var(--motion-fast) var(--ease-standard), box-shadow var(--motion-fast) var(--ease-standard)',
          width: '100%',
          ...style,
        }}
        {...props}
      >
        <Icon name={icon} size={20} style={{ color: 'var(--muted)', flexShrink: 0 }} />
        <span style={{ display: 'flex', flexDirection: 'column', gap: 2, minWidth: 0 }}>
          <strong style={{ fontSize: 'var(--text-sm)', color: 'var(--fg)', fontWeight: 500, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
            {title}
          </strong>
          <small style={{ fontSize: 'var(--text-xs)', color: 'var(--muted)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
            {desc}
          </small>
        </span>
      </button>
    );
  }
);

SuggestionChip.displayName = 'SuggestionChip';