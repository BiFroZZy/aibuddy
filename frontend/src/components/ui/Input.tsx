import { InputHTMLAttributes, forwardRef } from 'react';

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  helperText?: string;
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ label, error, helperText, className = '', id, style, ...props }, ref) => {
    const inputId = id || label?.toLowerCase().replace(/\s+/g, '-');
    const errorId = error ? `${inputId}-error` : undefined;
    const helperId = helperText ? `${inputId}-helper` : undefined;

    return (
      <label className={`input-wrapper ${className}`} style={{ display: 'block', width: '100%' }}>
        {label && (
          <span className="input__label" style={{ display: 'block', fontSize: 'var(--text-sm)', color: 'var(--fg-2)', marginBottom: 'var(--space-1)' }}>
            {label}
          </span>
        )}
        <input
          ref={ref}
          id={inputId}
          className={`input ${error ? 'input--error' : ''}`}
          style={{
            width: '100%',
            padding: 'var(--space-2) var(--space-3)',
            fontSize: 'var(--text-sm)',
            lineHeight: 'var(--leading-body)',
            borderRadius: 'var(--radius-md)',
            border: error ? '1px solid var(--danger)' : '1px solid var(--border)',
            background: 'var(--bg)',
            color: 'var(--fg)',
            transition: 'border-color var(--motion-fast) var(--ease-standard), box-shadow var(--motion-fast) var(--ease-standard)',
            outline: 'none',
            boxSizing: 'border-box',
            ...style,
          }}
          aria-invalid={error ? 'true' : 'false'}
          aria-describedby={error ? errorId : helperId}
          {...props}
        />
        {error && (
          <p id={errorId} className="input__error" role="alert" style={{ marginTop: 'var(--space-1)', fontSize: 'var(--text-xs)', color: 'var(--danger)' }}>
            {error}
          </p>
        )}
        {helperText && !error && (
          <p id={helperId} className="input__helper" style={{ marginTop: 'var(--space-1)', fontSize: 'var(--text-xs)', color: 'var(--muted)' }}>
            {helperText}
          </p>
        )}
      </label>
    );
  }
);

Input.displayName = 'Input';