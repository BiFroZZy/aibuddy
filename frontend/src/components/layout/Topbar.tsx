import { useAppState } from '@/state/useAppState';
import { IconButton } from '@/components/ui/IconButton';

const VIEW_LABELS: Record<string, string> = {
  home: 'Мой buddy',
  plan: 'План адаптации',
  materials: 'Рабочие материалы',
  requests: 'Мои заявки',
};

export function Topbar() {
  const { currentView, toggleSidebar, sidebarOpen } = useAppState();

  return (
    <header
      className="topbar"
      style={{
        height: '76px',
        borderBottom: '1px solid var(--border-soft)',
        padding: '0 var(--space-10)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        position: 'sticky',
        top: 0,
        zIndex: 20,
        background: 'var(--bg)',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-3)' }}>
        <IconButton
          icon="menu"
          aria-label="Открыть меню"
          id="menu"
          onClick={toggleSidebar}
          className="mobile-menu"
          style={{ display: sidebarOpen ? 'none' : 'inline-flex' }}
        />
        <span className="breadcrumb" style={{ fontSize: 'var(--text-sm)', fontWeight: 400 }}>
          <b id="crumb">{VIEW_LABELS[currentView]}</b>
        </span>
      </div>

      <div className="top-right" style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-3)' }}>
        <span className="demo" style={{ border: 'none', background: 'var(--surface)', padding: '4px var(--space-3)', borderRadius: 'var(--radius-pill)', fontSize: 'var(--text-xs)', color: 'var(--muted)' }}>
          Демо
        </span>
        <IconButton
          icon="shield"
          aria-label="О прототипе"
          onClick={() => window.dispatchEvent(new CustomEvent('ask-about'))}
        />
      </div>
    </header>
  );
}