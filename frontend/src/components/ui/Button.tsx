import { ButtonHTMLAttributes, forwardRef } from 'react';

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'text' | 'outline';
  size?: 'sm' | 'md' | 'lg';
  loading?: boolean;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ variant = 'primary', size = 'md', loading, disabled, children, className = '', style, ...props }, ref) => {
    const baseStyles: React.CSSProperties = {
      display: 'inline-flex',
      alignItems: 'center',
      justifyContent: 'center',
      gap: 'var(--space-2)',
      borderRadius: 'var(--radius-md)',
      fontWeight: 500,
      fontSize: 'var(--text-sm)',
      lineHeight: 'var(--leading-body)',
      transition: 'background var(--motion-fast) var(--ease-standard), color var(--motion-fast) var(--ease-standard), box-shadow var(--motion-fast) var(--ease-standard)',
      cursor: disabled || loading ? 'not-allowed' : 'pointer',
      opacity: disabled || loading ? 0.6 : 1,
      border: 'none',
      background: 'transparent',
      color: 'inherit',
    };

    const variantStyles: Record<string, React.CSSProperties> = {
      primary: {
        background: 'var(--accent)',
        color: 'var(--accent-on)',
      },
      text: {
        background: 'transparent',
        color: 'var(--accent-active)',
      },
      outline: {
        border: '1px solid var(--border)',
        background: 'var(--bg)',
        color: 'var(--fg)',
      },
    };

    const sizeStyles: Record<string, React.CSSProperties> = {
      sm: { padding: 'var(--space-1) var(--space-3)', minHeight: '36px', fontSize: 'var(--text-xs)' },
      md: { padding: 'var(--space-2) var(--space-4)', minHeight: '44px' },
      lg: { padding: 'var(--space-3) var(--space-6)', minHeight: '52px', fontSize: 'var(--text-base)' },
    };

    const combinedStyle = {
      ...baseStyles,
      ...variantStyles[variant],
      ...sizeStyles[size],
      ...style,
    };

    // We'll handle hover/active via CSS classes instead of inline styles
    const classNames = [
      'btn',
      `btn--${variant}`,
      `btn--${size}`,
      className,
    ].filter(Boolean).join(' ');

    return (
      <button
        ref={ref}
        className={classNames}
        style={combinedStyle}
        disabled={disabled || loading}
        {...props}
      >
        {loading && <span className="btn__spinner" aria-hidden="true">⏳</span>}
        {children}
      </button>
    );
  }
);

Button.displayName = 'Button';