import { TextareaHTMLAttributes, forwardRef } from 'react';

export interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  label?: string;
  error?: string;
  helperText?: string;
}

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ label, error, helperText, className = '', id, ...props }, ref) => {
    const inputId = id || label?.toLowerCase().replace(/\s+/g, '-');
    const errorId = error ? `${inputId}-error` : undefined;
    const helperId = helperText ? `${inputId}-helper` : undefined;

    return (
      <label className={`textarea-wrapper ${className}`} style={{ display: 'block', width: '100%' }}>
        {label && (
          <span className="textarea__label" style={{ display: 'block', fontSize: 'var(--text-sm)', color: 'var(--fg-2)', marginBottom: 'var(--space-1)' }}>
            {label}
          </span>
        )}
        <textarea
          ref={ref}
          id={inputId}
          className={`textarea ${error ? 'textarea--error' : ''}`}
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
            resize: 'vertical',
            minHeight: '80px',
            fontFamily: 'inherit',
          }}
          aria-invalid={error ? 'true' : 'false'}
          aria-describedby={error ? errorId : helperId}
          {...props}
        />
        {error && (
          <p id={errorId} className="textarea__error" role="alert" style={{ marginTop: 'var(--space-1)', fontSize: 'var(--text-xs)', color: 'var(--danger)' }}>
            {error}
          </p>
        )}
        {helperText && !error && (
          <p id={helperId} className="textarea__helper" style={{ marginTop: 'var(--space-1)', fontSize: 'var(--text-xs)', color: 'var(--muted)' }}>
            {helperText}
          </p>
        )}
      </label>
    );
  }
);

Textarea.displayName = 'Textarea';