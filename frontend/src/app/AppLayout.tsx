import { Sidebar } from '@/components/layout/Sidebar';
import { Topbar } from '@/components/layout/Topbar';
import { Content } from '@/components/layout/Content';

export function AppLayout() {
  return (
    <div className="app">
      <Sidebar />
      <div className="workspace" style={{ display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <Topbar />
        <Content />
      </div>
    </div>
  );
}