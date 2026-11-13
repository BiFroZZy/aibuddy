import { Sidebar } from '@/components/layout/Sidebar';
import { Topbar } from '@/components/layout/Topbar';
import { Content } from '@/components/layout/Content';

export function AppLayout() {
  return (
    <div className="app" style={{ display: 'grid', gridTemplateColumns: '224px minmax(0, 1fr)', minHeight: '100vh' }}>
      <Sidebar />
      <div className="workspace" style={{ display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <Topbar />
        <Content />
      </div>
    </div>
  );
}