import { ButtonHTMLAttributes, forwardRef } from 'react';
import { Icon } from '@/components/icons/Icon';

export interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  icon: string;
  size?: 'sm' | 'md' | 'lg';
  'aria-label': string;
}

export const IconButton = forwardRef<HTMLButtonElement, IconButtonProps>(
  ({ icon, size = 'md', 'aria-label': ariaLabel, className = '', style, children, ...props }, ref) => {
    const sizeMap = {
      sm: 36,
      md: 44,
      lg: 52,
    };

    const iconSizeMap = {
      sm: 16,
      md: 18,
      lg: 20,
    };

    const buttonSize = sizeMap[size];
    const iconSize = iconSizeMap[size];

    return (
      <button
        ref={ref}
        className={`icon-btn icon-btn--${size} ${className}`}
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          justifyContent: 'center',
          width: buttonSize,
          height: buttonSize,
          borderRadius: 'var(--radius-md)',
          background: 'transparent',
          border: 'none',
          color: 'inherit',
          cursor: 'pointer',
          transition: 'background var(--motion-fast) var(--ease-standard)',
          ...style,
        }}
        aria-label={ariaLabel}
        {...props}
      >
        <Icon name={icon} size={iconSize} />
        {children}
      </button>
    );
  }
);

IconButton.displayName = 'IconButton';