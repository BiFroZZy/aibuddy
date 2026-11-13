import { useAppState } from '@/state/useAppState';
import { Button } from '@/components/ui/Button';
import { Icon } from '@/components/icons/Icon';

const NAV_ITEMS = [
  { id: 'home', icon: 'chat', label: 'Мой buddy', countId: null },
  { id: 'plan', icon: 'plan', label: 'План адаптации', countId: 'planCount' },
  { id: 'materials', icon: 'book', label: 'Рабочие материалы', countId: null },
  { id: 'requests', icon: 'box', label: 'Мои заявки', countId: 'requestCount' },
] as const;

export function Sidebar() {
  const { chats, activeChatId, selectChat, currentView, setView, sidebarOpen, closeSidebar, planStatuses, requests } = useAppState();

  const doneCount = planStatuses.filter((s) => s === 'done').length;
  const requestCount = requests.length;

  const handleNavClick = (viewId: typeof NAV_ITEMS[number]['id']) => {
    setView(viewId);
    closeSidebar();
  };

  return (
    <aside
      className={`sidebar ${sidebarOpen ? 'sidebar--open' : ''}`}
      id="sidebar"
      style={{
        background: 'var(--surface)',
        borderRight: '1px solid var(--border-soft)',
        padding: 'var(--space-8) var(--space-4) var(--space-5)',
        display: 'flex',
        flexDirection: 'column',
        gap: 'var(--space-9)',
        height: '100dvh',
        position: 'sticky',
        top: 0,
        overflowY: 'auto',
        zIndex: sidebarOpen ? 30 : 10,
      }}
      aria-label="Навигация"
    >
      <div className="brand" style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-3)', fontSize: '23px', fontWeight: 650, letterSpacing: '-0.6px', padding: '0 var(--space-2)' }}>
        <span className="buddy-mark" aria-hidden="true" style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 12px)', gap: '3px', flexShrink: 0 }}>
          <i style={{ width: '12px', height: '12px', borderRadius: '50%', background: 'var(--brand-blue)' }}></i>
          <i style={{ width: '12px', height: '12px', borderRadius: '50%', background: 'var(--brand-green)' }}></i>
          <i style={{ width: '12px', height: '12px', borderRadius: '50%', background: 'var(--brand-pink)' }}></i>
          <i style={{ width: '12px', height: '12px', borderRadius: '50%', background: 'var(--brand-purple)' }}></i>
        </span>
        <span>
          AI-buddy
          <small style={{ fontSize: '11px', letterSpacing: '0', textTransform: 'none', fontWeight: 400, color: 'var(--muted)', display: 'block', marginTop: '3px' }}>
            для стажёров Авито
          </small>
        </span>
      </div>

      <Button
        variant="outline"
        className="new-chat"
        onClick={() => {
          setView('home');
          closeSidebar();
          window.dispatchEvent(new CustomEvent('new-chat'));
        }}
        style={{
          border: 'none',
          boxShadow: 'none',
          padding: 'var(--space-3) var(--space-3)',
          minHeight: '44px',
          justifyContent: 'space-between',
          width: '100%',
          fontSize: 'var(--text-sm)',
          background: 'var(--bg)',
          borderRadius: 'var(--radius-md)',
        }}
      >
        Новый диалог
        <Icon name="plus" size={18} />
      </Button>

      <nav className="nav" aria-label="Разделы" style={{ display: 'grid', gap: 'var(--space-2)' }}>
        {NAV_ITEMS.map((item) => (
          <button
            key={item.id}
            className={`nav__item ${currentView === item.id ? 'nav__item--active' : ''}`}
            onClick={() => handleNavClick(item.id)}
            style={{
              display: 'flex',
              alignItems: 'center',
              textAlign: 'left',
              gap: 'var(--space-3)',
              padding: 'var(--space-3) var(--space-3)',
              borderRadius: 'var(--radius-sm)',
              fontSize: 'var(--text-sm)',
              color: currentView === item.id ? 'var(--accent-active)' : 'var(--fg-2)',
              background: currentView === item.id ? 'var(--brand-tint)' : 'transparent',
              border: 'none',
              cursor: 'pointer',
              transition: 'background var(--motion-fast) var(--ease-standard), color var(--motion-fast) var(--ease-standard)',
              minHeight: '44px',
              width: '100%',
            }}
            aria-current={currentView === item.id ? 'page' : undefined}
          >
            <Icon name={item.icon} size={18} />
            <span className="nav__label" style={{ flex: 1 }}>{item.label}</span>
            {item.countId === 'planCount' && (
              <span className="count" style={{ fontSize: '11px', color: 'var(--muted)', background: 'transparent', padding: 0 }}>
                {doneCount}/8
              </span>
            )}
            {item.countId === 'requestCount' && (
              <span className="count" style={{ fontSize: '11px', color: 'var(--muted)', background: 'transparent', padding: 0 }}>
                {requestCount}
              </span>
            )}
          </button>
        ))}
      </nav>

      <nav className="chat-history" aria-label="История чатов">
        <p className="chat-history-title">Чаты</p>
        {chats.length === 0 && <p className="chat-history-empty">Здесь появятся твои диалоги</p>}
        {chats.map(chat => (
          <button key={chat.id} type="button" className="chat-history-item"
            aria-current={currentView === 'home' && activeChatId === chat.id ? 'page' : undefined}
            onClick={() => selectChat(chat.id)} title={chat.title}>
            <Icon name="chat" size={16} /><span>{chat.title}</span>
          </button>
        ))}
      </nav>

      <div className="sidebar-bottom" style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-4)', marginTop: 'auto' }}>
        <button
          className="mentor-link"
          onClick={() => {
            setView('home');
            closeSidebar();
            window.dispatchEvent(new CustomEvent('ask-mentor'));
          }}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 'var(--space-3)',
            fontSize: 'var(--text-sm)',
            padding: 'var(--space-3) var(--space-3)',
            color: 'var(--fg-2)',
            background: 'transparent',
            border: 'none',
            cursor: 'pointer',
            textAlign: 'left',
            width: '100%',
            borderRadius: 'var(--radius-sm)',
            transition: 'background var(--motion-fast) var(--ease-standard)',
          }}
          onMouseEnter={(e) => (e.currentTarget.style.background = 'var(--surface-warm)')}
          onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
        >
          <Icon name="people" size={18} />
          Мой наставник
        </button>

        <div className="profile" style={{ borderTop: '1px solid var(--border-soft)', padding: 'var(--space-5) var(--space-2) 0' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-3)' }}>
            <span
              className="avatar"
              style={{
                width: '36px',
                height: '36px',
                borderRadius: 'var(--radius-sm)',
                background: 'var(--bg)',
                fontWeight: 400,
                fontSize: 'var(--text-sm)',
                color: 'var(--fg)',
                display: 'grid',
                placeItems: 'center',
              }}
            >
              АС
            </span>
            <div style={{ minWidth: 0 }}>
              <strong style={{ display: 'block', fontSize: 'var(--text-sm)', fontWeight: 600, color: 'var(--fg)' }}>Алексей Смирнов</strong>
              <small style={{ fontSize: 'var(--text-xs)', color: 'var(--muted)' }}>Backend · Стажёр · Демо</small>
            </div>
          </div>
        </div>
      </div>
    </aside>
  );
}