import { useEffect, useId, useRef, useState, type KeyboardEvent } from 'react';

export interface SelectOption {
  value: string;
  label: string;
}

export interface SelectProps {
  value: string;
  onValueChange: (value: string) => void;
  options: SelectOption[];
  label?: string;
  error?: string;
  placeholder?: string;
  className?: string;
  disabled?: boolean;
  'aria-label'?: string;
}

export function Select({ value, onValueChange, options, label, error, placeholder = 'Выбрать', className = '', disabled, 'aria-label': ariaLabel }: SelectProps) {
  const id = useId();
  const root = useRef<HTMLDivElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const list = useRef<HTMLUListElement>(null);
  const [open, setOpen] = useState(false);
  const [above, setAbove] = useState(false);
  const [active, setActive] = useState(0);
  const selected = options.findIndex(option => option.value === value);

  useEffect(() => {
    if (!open) return;
    const outside = (event: PointerEvent) => {
      if (!root.current?.contains(event.target as Node)) setOpen(false);
    };
    const reposition = () => setOpen(false);
    document.addEventListener('pointerdown', outside);
    window.addEventListener('resize', reposition);
    return () => {
      document.removeEventListener('pointerdown', outside);
      window.removeEventListener('resize', reposition);
    };
  }, [open]);

  useEffect(() => {
    if (open) list.current?.children[active]?.scrollIntoView({ block: 'nearest' });
  }, [open, active]);

  const show = (index = Math.max(0, selected)) => {
    if (disabled || options.length === 0) return;
    const rect = trigger.current?.getBoundingClientRect();
    setAbove(Boolean(rect && window.innerHeight - rect.bottom < 200 && rect.top > 200));
    setActive(index);
    setOpen(true);
  };

  const choose = (index: number) => {
    const option = options[index];
    if (!option) return;
    onValueChange(option.value);
    setOpen(false);
    trigger.current?.focus();
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLButtonElement>) => {
    if (event.key === 'Tab') { setOpen(false); return; }
    if (event.key === 'Escape') {
      if (open) { event.preventDefault(); event.stopPropagation(); setOpen(false); }
      return;
    }
    if (['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) {
      event.preventDefault();
      if (!open) { show(event.key === 'End' ? options.length - 1 : event.key === 'Home' ? 0 : Math.max(0, selected)); return; }
      setActive(index => event.key === 'Home' ? 0 : event.key === 'End' ? options.length - 1 : (index + (event.key === 'ArrowDown' ? 1 : -1) + options.length) % options.length);
    } else if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      if (open) choose(active); else show();
    } else if (open && event.key.length === 1) {
      const match = options.findIndex(option => option.label.toLocaleLowerCase().startsWith(event.key.toLocaleLowerCase()));
      if (match >= 0) { event.preventDefault(); setActive(match); }
    }
  };

  return (
    <div ref={root} className={`select-wrapper custom-select ${className}`}>
      {label && <label htmlFor={id} className="select__label">{label}</label>}
      <button ref={trigger} id={id} type="button" role="combobox" className="custom-select-trigger"
        disabled={disabled || options.length === 0} aria-label={ariaLabel || label}
        aria-expanded={open} aria-haspopup="listbox" aria-controls={open ? `${id}-list` : undefined}
        aria-activedescendant={open ? `${id}-option-${active}` : undefined}
        aria-invalid={Boolean(error)} aria-describedby={error ? `${id}-error` : undefined}
        onClick={() => open ? setOpen(false) : show()} onKeyDown={handleKeyDown}
        onBlur={event => { if (!root.current?.contains(event.relatedTarget)) setOpen(false); }}>
        <span>{options[selected]?.label ?? placeholder}</span>
        <svg viewBox="0 0 24 24" aria-hidden="true" className="custom-select-chevron"><path d="m6 9 6 6 6-6" /></svg>
      </button>
      {open && <ul ref={list} id={`${id}-list`} role="listbox" aria-label={ariaLabel || label || 'Варианты'} className={`custom-select-menu ${above ? 'custom-select-menu--above' : ''}`}>
        {options.map((option, index) => <li key={option.value} id={`${id}-option-${index}`} role="option"
          aria-selected={option.value === value} className={`custom-select-option ${active === index ? 'is-active' : ''}`}
          onPointerMove={() => setActive(index)} onMouseDown={event => event.preventDefault()} onClick={() => choose(index)}>
          <span>{option.label}</span>
          {option.value === value && <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m5 12 4 4L19 6" /></svg>}
        </li>)}
      </ul>}
      {error && <p id={`${id}-error`} role="alert" className="select__error">{error}</p>}
    </div>
  );
}
