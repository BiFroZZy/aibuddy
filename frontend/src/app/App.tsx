import { Icon } from '../components/icons/Icon';
import { useCallback, useEffect, useState } from 'react';
import { ApiClient, errorMessage } from '../api/client';
import type { Conversation, Identity } from '../api/types';
import {
  allowed,
  conversations,
  labels,
  profiles,
  roleLabels,
  route,
  saveToken,
  sessionToken,
} from '../features/model';
import type { Page } from '../features/model';
import { Empty, ErrorNotice, Loading } from '../features/shared';
import { ChatScreen } from '../features/Chat';
import { RequestsScreen } from '../features/Actions';
import {
  ApprovalsScreen,
  MaterialsScreen,
  MentorScreen,
  MetricsScreen,
  PlanScreen,
} from '../features/Screens';
import '../styles/server.css';
import '../styles/app.css';
export function App() {
  const [token, setToken] = useState(sessionToken);
  const [session, setSession] = useState<{ api: ApiClient; me: Identity; token: string }>();
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [retry, setRetry] = useState(0);
  const logout = useCallback(() => {
    saveToken('');
    setToken('');
    setSession(undefined);
    location.hash = 'login';
  }, []);
  useEffect(() => {
    setSession(undefined);
    setError('');
    if (!token) {
      setLoading(false);
      return;
    }
    let active = true;
    const api = new ApiClient(token, () => {
      if (active) logout();
    });
    setLoading(true);
    api
      .me()
      .then(me => {
        if (active) setSession({ api, me, token });
      })
      .catch(e => {
        if (active) setError(errorMessage(e));
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
      api.dispose();
    };
  }, [token, retry, logout]);
  if (!session || session.token !== token)
    return (
      <main className="login content">
        <Brand />
        <p className="eyebrow">AI-buddy / Демо-доступ</p>
        <h1>
          Одна команда.
          <br />
          Разные возможности.
        </h1>
        <p className="intro">Выберите профиль, чтобы пройти свой сценарий адаптации.</p>
        <ErrorNotice message={error} retry={() => setRetry(x => x + 1)} />
        {loading ? (
          <Loading />
        ) : (
          <div className="role-grid">
            {profiles.map(p => (
              <button
                className="role-card"
                key={p.token}
                onClick={() => {
                  saveToken(p.token);
                  setToken(p.token);
                  setRetry(x => x + 1);
                  location.hash = allowed[p.role][0];
                }}
              >
                <span className="avatar">
                  {p.name
                    .split(' ')
                    .map(n => n[0])
                    .join('')}
                </span>
                <strong>{p.name}</strong>
                <small>{roleLabels[p.role]}</small>
              </button>
            ))}
          </div>
        )}
        <p className="quiet-note">
          Учебные данные. Выбор профиля использует демонстрационный токен, а не корпоративную
          учётную запись.
        </p>
      </main>
    );
  return (
    <Workspace
      key={session.me.user_id}
      api={session.api}
      me={session.me}
      token={token}
      logout={logout}
    />
  );
}
function Brand() {
  return (
    <div className="brand">
      <span className="buddy-mark" aria-hidden="true">
        <i />
        <i />
        <i />
        <i />
      </span>
      <span>
        AI-buddy<small>для стажёров Авито</small>
      </span>
    </div>
  );
}
function Workspace({
  api,
  me,
  token,
  logout,
}: {
  api: ApiClient;
  me: Identity;
  token: string;
  logout: () => void;
}) {
  const [current, setCurrent] = useState(route);
  const [menu, setMenu] = useState(false);
  const [history, setHistory] = useState<Conversation[]>(() => conversations(me.user_id));
  const [draft, setDraft] = useState('');
  const [chatVersion, setChatVersion] = useState(0);
  const page = current.page as Page;
  const permitted = allowed[me.role]?.includes(page);
  const profile = profiles.find(p => p.token === token);
  const displayName = profile?.name || roleLabels[me.role];
  useEffect(() => {
    const update = () => {
      setCurrent(route());
      setChatVersion(v => v + 1);
      setMenu(false);
    };
    const escape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setMenu(false);
        document.getElementById('menu-toggle')?.focus();
      }
    };
    addEventListener('hashchange', update);
    addEventListener('keydown', escape);
    return () => {
      removeEventListener('hashchange', update);
      removeEventListener('keydown', escape);
    };
  }, []);
  useEffect(() => {
    if (!menu) return;
    const sidebar = document.getElementById('sidebar');
    const workspace = document.querySelector<HTMLElement>('.workspace');
    workspace?.setAttribute('inert', '');
    const controls = () =>
      Array.from(sidebar?.querySelectorAll<HTMLElement>('a[href],button:not(:disabled)') || []);
    controls()[0]?.focus();
    const trap = (e: KeyboardEvent) => {
      if (e.key !== 'Tab') return;
      const list = controls(),
        first = list[0],
        last = list[list.length - 1];
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault();
        last?.focus();
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault();
        first?.focus();
      }
    };
    document.addEventListener('keydown', trap);
    return () => {
      workspace?.removeAttribute('inert');
      document.removeEventListener('keydown', trap);
      document.getElementById('menu-toggle')?.focus();
    };
  }, [menu]);
  const newChat = () => {
    setChatVersion(v => v + 1);
    setDraft('');
    location.hash = 'chat';
    setCurrent({ page: 'chat', conversation: undefined });
    setMenu(false);
  };
  const onConversation = useCallback((id: string) => {
    historyReplace(id);
    setCurrent({ page: 'chat', conversation: id });
  }, []);
  function startRequest() {
    setChatVersion(v => v + 1);
    setDraft('Хочу оформить заявку: ');
    location.hash = 'chat';
    setCurrent({ page: 'chat', conversation: undefined });
  }
  return (
    <div className="app">
      <a
        className="skip-link"
        href="#main-content"
        onClick={e => {
          e.preventDefault();
          document.getElementById('main-content')?.focus();
        }}
      >
        К содержимому
      </a>
      {menu && (
        <button
          className="menu-backdrop"
          aria-label="Закрыть меню"
          onClick={() => setMenu(false)}
        />
      )}
      <aside
        role={menu ? 'dialog' : undefined}
        aria-modal={menu || undefined}
        aria-label="Рабочее пространство"
        className={'sidebar ' + (menu ? 'open' : '')}
        id="sidebar"
      >
        <Brand />
        <button className="new btn" onClick={newChat}>
          Новый диалог <Icon name="plus" />
        </button>
        <nav className="nav" aria-label="Разделы">
          <div className="nav-label">Рабочее пространство</div>
          {allowed[me.role].map(p => (
            <a
              className={p === page ? 'active' : ''}
              aria-current={p === page ? 'page' : undefined}
              key={p}
              href={'#' + p}
              onClick={() => setMenu(false)}
            >
              <Icon
                name={
                  {
                    chat: 'chat',
                    plan: 'plan',
                    requests: 'box',
                    mentor: 'people',
                    approvals: 'check',
                    metrics: 'plan',
                    materials: 'book',
                  }[p]
                }
              />
              {labels[p]}
            </a>
          ))}
        </nav>
        <section className="chat-history" aria-label="История диалогов">
          <div className="nav-label">Чаты</div>
          {history.length ? (
            <div className="history-list">
              {history.map(c => (
                <a
                  key={c.id}
                  href={'#chat/' + c.id}
                  aria-current={
                    page === 'chat' && current.conversation === c.id ? 'page' : undefined
                  }
                  onClick={() => setMenu(false)}
                  title={c.title}
                >
                  {c.title}
                </a>
              ))}
            </div>
          ) : (
            <p className="quiet-note">Здесь появятся ваши диалоги.</p>
          )}
          <p className="quiet-note">Диалоги, открытые в этом браузере.</p>
        </section>
        <div className="sidebar-bottom">
          <div className="profile">
            <span className="avatar">
              {displayName
                .split(' ')
                .map(n => n[0])
                .join('')
                .slice(0, 2)}
            </span>
            <div>
              <strong>{displayName}</strong>
              <small>
                {me.team} · {roleLabels[me.role]}
              </small>
            </div>
          </div>
          <button className="role-link" onClick={logout}>
            Сменить пользователя →
          </button>
        </div>
      </aside>
      <div className="workspace">
        <header className="topbar">
          <div className="topbar-left">
            <button
              className="iconbutton mobile-menu"
              id="menu-toggle"
              aria-label="Открыть меню"
              aria-expanded={menu}
              aria-controls="sidebar"
              onClick={() => setMenu(v => !v)}
            >
              <Icon name="menu" />
            </button>
            <span className="breadcrumb">
              AI-buddy <span className="muted"> / </span>
              <b>{labels[page] || 'Раздел недоступен'}</b>
            </span>
          </div>
          <span className="demo">Демо · учебные данные</span>
        </header>
        <main className="content" id="main-content" tabIndex={-1}>
          {!permitted ? (
            <Empty>Раздел недоступен для вашей роли. Выберите раздел в меню.</Empty>
          ) : page === 'chat' ? (
            <ChatScreen
              key={chatVersion}
              api={api}
              me={me}
              conversation={current.conversation}
              draft={draft}
              onDraft={setDraft}
              onRemember={setHistory}
              onConversation={onConversation}
            />
          ) : page === 'plan' ? (
            <PlanScreen api={api} />
          ) : page === 'requests' ? (
            <RequestsScreen api={api} onNew={startRequest} />
          ) : page === 'mentor' ? (
            <MentorScreen api={api} />
          ) : page === 'approvals' ? (
            <ApprovalsScreen api={api} />
          ) : page === 'metrics' ? (
            <MetricsScreen api={api} />
          ) : (
            <MaterialsScreen />
          )}
        </main>
        <footer className="bottom-note">
          <span>AI-buddy · {me.team}</span>
          <span>Синтетические демонстрационные данные</span>
        </footer>
      </div>
    </div>
  );
}
function historyReplace(id: string) {
  window.history.replaceState(null, '', '#chat/' + id);
}
