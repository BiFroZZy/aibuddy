import { useAppState } from '@/state/useAppState';
import { HomeView } from '@/components/views/HomeView';
import { PlanView } from '@/components/views/PlanView';
import { MaterialsView } from '@/components/views/MaterialsView';
import { RequestsView } from '@/components/views/RequestsView';

export function Content() {
  const { currentView } = useAppState();

  const renderView = () => {
    switch (currentView) {
      case 'home':
        return <HomeView />;
      case 'plan':
        return <PlanView />;
      case 'materials':
        return <MaterialsView />;
      case 'requests':
        return <RequestsView />;
      default:
        return <HomeView />;
    }
  };

  return (
    <main
      className="content"
      style={{
        maxWidth: '1100px',
        margin: '0 auto',
        padding: '0 var(--space-10) var(--space-6)',
        minHeight: 'calc(100dvh - 76px)',
        display: 'flex',
        flexDirection: 'column',
        width: '100%',
      }}
    >
      <div hidden={currentView !== 'home'}><HomeView /></div>
      {currentView !== 'home' && renderView()}
      <footer className="footer" style={{ marginTop: 'auto', paddingTop: 'var(--space-6)' }}>
        <button
          className="textbutton"
          onClick={() => window.dispatchEvent(new CustomEvent('ask-about'))}
          style={{
            background: 'transparent',
            border: 'none',
            color: 'var(--accent-active)',
            fontSize: 'var(--text-sm)',
            cursor: 'pointer',
            padding: 0,
          }}
        >
          О демо-прототипе
        </button>
      </footer>
    </main>
  );
}