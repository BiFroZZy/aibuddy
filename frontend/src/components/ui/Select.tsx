import { SelectHTMLAttributes, forwardRef } from 'react';

export interface SelectOption {
  value: string;
  label: string;
}

export interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  label?: string;
  error?: string;
  options: SelectOption[];
  placeholder?: string;
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(
  ({ label, error, options, placeholder, className = '', id, ...props }, ref) => {
    const selectId = id || label?.toLowerCase().replace(/\s+/g, '-');
    const errorId = error ? `${selectId}-error` : undefined;

    return (
      <label className={`select-wrapper ${className}`} style={{ display: 'block', width: '100%' }}>
        {label && (
          <span className="select__label" style={{ display: 'block', fontSize: 'var(--text-sm)', color: 'var(--fg-2)', marginBottom: 'var(--space-1)' }}>
            {label}
          </span>
        )}
        <select
          ref={ref}
          id={selectId}
          className={`select ${error ? 'select--error' : ''}`}
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
            appearance: 'none',
            backgroundImage: `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='%236e6e73' stroke-width='1.6' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M6 9l6 6 6-6'/%3E%3C/svg%3E")`,
            backgroundRepeat: 'no-repeat',
            backgroundPosition: 'right var(--space-3) center',
            paddingRight: 'var(--space-10)',
          }}
          aria-invalid={error ? 'true' : 'false'}
          aria-describedby={errorId}
          {...props}
        >
          {placeholder && (
            <option value="" disabled selected hidden>
              {placeholder}
            </option>
          )}
          {options.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
        {error && (
          <p id={errorId} className="select__error" role="alert" style={{ marginTop: 'var(--space-1)', fontSize: 'var(--text-xs)', color: 'var(--danger)' }}>
            {error}
          </p>
        )}
      </label>
    );
  }
);

Select.displayName = 'Select';