import { useEffect, useRef, type DetailsHTMLAttributes, type MouseEvent } from 'react';

/** Keep native details semantics while animating the measured content height. */
export function AnimatedDetails({ children, ...props }: DetailsHTMLAttributes<HTMLDetailsElement>) {
  const animation = useRef<Animation | null>(null);

  useEffect(() => () => animation.current?.cancel(), []);

  const handleClick = (event: MouseEvent<HTMLDetailsElement>) => {
    const details = event.currentTarget;
    const summary = details.querySelector('summary');
    if (!summary?.contains(event.target as Node)) return;
    event.preventDefault();

    const expanding = !details.open || details.hasAttribute('data-closing');
    const start = details.getBoundingClientRect().height;
    animation.current?.cancel();
    animation.current = null;
    details.removeAttribute('data-closing');

    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      details.open = expanding;
      return;
    }

    // Measure both states synchronously; keep content rendered until collapse ends.
    details.open = expanding;
    const end = details.getBoundingClientRect().height;
    details.open = true;
    if (!expanding) details.setAttribute('data-closing', '');

    const current = details.animate(
      [{ height: `${start}px` }, { height: `${end}px` }],
      { duration: 240, easing: 'cubic-bezier(.28,0,.22,1)' },
    );
    animation.current = current;
    current.onfinish = () => {
      if (animation.current !== current) return;
      details.open = expanding;
      details.removeAttribute('data-closing');
      animation.current = null;
      current.cancel();
    };
  };

  return <details {...props} onClick={handleClick}>{children}</details>;
}
