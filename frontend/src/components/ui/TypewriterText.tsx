import { useEffect, useState } from 'react';

export function TypewriterText({ text, animate = false }: { text: string; animate?: boolean }) {
  const characters = Array.from(text);
  const [visible, setVisible] = useState(animate ? 0 : characters.length);

  useEffect(() => {
    const preference = window.matchMedia('(prefers-reduced-motion: reduce)');
    if (!animate || preference.matches) {
      setVisible(Array.from(text).length);
      return;
    }
    const length = Array.from(text).length;
    const step = Math.max(1, Math.ceil(length / 160));
    setVisible(0);
    const timer = window.setInterval(() => {
      setVisible(count => {
        const next = Math.min(length, count + step);
        if (next === length) window.clearInterval(timer);
        return next;
      });
    }, 20);
    const finish = () => {
      if (preference.matches) {
        window.clearInterval(timer);
        setVisible(length);
      }
    };
    preference.addEventListener('change', finish);
    return () => {
      window.clearInterval(timer);
      preference.removeEventListener('change', finish);
    };
  }, [text, animate]);

  const typing = animate && visible < characters.length;
  return (
    <span className="typewriter-text">
      <span className="sr-only">{text}</span>
      <span className="typewriter-reserve" aria-hidden="true">{text}</span>
      <span className="typewriter-visible" aria-hidden="true">
        {characters.slice(0, visible).join('')}
        {typing && <span className="typewriter-caret" />}
      </span>
    </span>
  );
}
