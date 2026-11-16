import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, expect, test, vi } from 'vitest';
import { App } from './App';
const identity = { user_id: 'intern', role: 'intern', team: 'Demo' };
const progress = { steps: [], next: null, waiting: [], done: 0, total: 0, percent: 0 };
const action = {
  id: 'a',
  tool: 'request_access',
  status: 'pending_confirmation',
  args: { project: 'private-resource', level: 'read' },
  created_at: '2026-11-16',
};
let role = identity.role;
let mode = 'ok';
let readCalls: string[] = [];
let postCalls = 0;
function json(value: unknown, status = 200) {
  return new Response(JSON.stringify(value), { status });
}
beforeEach(() => {
  role = 'intern';
  mode = 'ok';
  readCalls = [];
  postCalls = 0;
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: string, init: RequestInit) => {
      const p = String(input);
      if (init.method === 'POST') {
        postCalls++;
        if (p.endsWith('/confirm')) {
          if (mode === 'lost') {
            mode = 'done';
            throw Error('connection lost');
          }
          mode = 'failed';
          return json({ ...action, status: 'failed', error: 'raw SQL password secret' });
        }
        return json({
          conversation_id: 'c',
          reply: 'Ответ',
          tools_used: [],
          pending_actions: [action],
        });
      }
      readCalls.push(p);
      if (p === '/api/me')
        return json({
          ...identity,
          user_id: String((init.headers as Record<string, string>).Authorization).includes(
            'intern-2'
          )
            ? 'intern2'
            : 'intern',
          role,
        });
      if (p === '/api/progress')
        return mode === 'forbidden' ? json({ error: 'forbidden' }, 403) : json(progress);
      if (
        p === '/api/actions' &&
        String((init.headers as Record<string, string>).Authorization).includes('intern-2')
      )
        return json({ actions: [] });
      if (p === '/api/actions')
        return json({
          actions: [
            {
              ...action,
              status:
                mode === 'done' ? 'done' : mode === 'failed' ? 'failed' : 'pending_confirmation',
            },
          ],
        });
      if (p === '/api/requests') return json({ requests: [] });
      if (p.includes('/messages')) return json({ error: 'not found' }, 404);
      if (p === '/api/manager/requests')
        return json({
          requests: [
            {
              id: 'r',
              kind: 'absence',
              payload: { kind: 'vacation' },
              user_id: 'i',
              user_name: 'Анна',
              status: 'pending_approval',
              approver_role: 'manager',
              created_at: '2026-11-16',
            },
          ],
        });
      return json({});
    })
  );
});
async function login() {
  render(<App />);
  await userEvent.click(screen.getByRole('button', { name: /Анна/ }));
  await screen.findByRole('button', { name: 'Открыть меню' });
}
test('a failed action is not shown as success or raw internal error', async () => {
  await login();
  await userEvent.type(screen.getByLabelText('Сообщение'), 'Доступ');
  await userEvent.click(screen.getByRole('button', { name: 'Отправить' }));
  await userEvent.click(await screen.findByRole('button', { name: 'Подтвердить отправку' }));
  await screen.findByText('Ошибка', { exact: true });
  expect(screen.queryByText(/raw SQL/)).not.toBeInTheDocument();
  expect(screen.queryByText('Выполнено', { exact: true })).not.toBeInTheDocument();
});
test('lost confirmation response reconciles state without retrying POST', async () => {
  mode = 'lost';
  await login();
  await userEvent.type(screen.getByLabelText('Сообщение'), 'Доступ');
  await userEvent.click(screen.getByRole('button', { name: 'Отправить' }));
  await userEvent.click(await screen.findByRole('button', { name: 'Подтвердить отправку' }));
  await screen.findByText('Выполнено', { exact: true });
  expect(postCalls).toBe(2);
  expect(screen.queryByRole('button', { name: 'Подтвердить отправку' })).not.toBeInTheDocument();
});
test('foreign-role route never loads restricted data', async () => {
  await login();
  location.hash = 'approvals';
  await screen.findByText('Раздел недоступен для вашей роли. Выберите раздел в меню.');
  expect(readCalls).not.toContain('/api/manager/requests');
});
test('403 reads show a meaningful error and can be refreshed', async () => {
  mode = 'forbidden';
  await login();
  await userEvent.click(screen.getByRole('link', { name: 'План адаптации' }));
  await screen.findByText('У вашей роли нет доступа к этой операции.');
  mode = 'ok';
  await userEvent.click(screen.getByRole('button', { name: 'Обновить данные' }));
  await screen.findByText('План пока не назначен. Обратитесь к наставнику.');
});
test('switching profile removes all previous chat and pending actions', async () => {
  await login();
  await userEvent.type(screen.getByLabelText('Сообщение'), 'Секретный вопрос');
  await userEvent.click(screen.getByRole('button', { name: 'Отправить' }));
  await screen.findByRole('button', { name: 'Подтвердить отправку' });
  await userEvent.click(screen.getByRole('button', { name: 'Сменить пользователя →' }));
  await userEvent.click(screen.getByRole('button', { name: /Борис/ }));
  await screen.findByRole('button', { name: 'Открыть меню' });
  expect(screen.queryByText('Секретный вопрос')).not.toBeInTheDocument();
  expect(screen.queryByText('private-resource')).not.toBeInTheDocument();
});
test('failed chat remains a visible error after refreshing history', async () => {
  await login();
  vi.stubGlobal(
    'fetch',
    vi.fn(async (p: string) =>
      p === '/api/chat'
        ? json({ error: 'unavailable' }, 502)
        : p === '/api/actions'
          ? json({ actions: [] })
          : json(progress)
    )
  );
  await userEvent.type(screen.getByLabelText('Сообщение'), 'Вопрос');
  await userEvent.click(screen.getByRole('button', { name: 'Отправить' }));
  await waitFor(() =>
    expect(screen.getByRole('alert')).toHaveTextContent('Сервис временно недоступен')
  );
});

test('a conflict during manager decision remains visible after queue refresh', async () => {
  role = 'manager';
  await login();
  await userEvent.click(screen.getByRole('link', { name: 'Согласования' }));
  await screen.findByRole('button', { name: 'Одобрить заявку' });
  vi.stubGlobal(
    'fetch',
    vi.fn(async (_p: string, init: RequestInit) =>
      init.method === 'POST' ? json({ error: 'conflict' }, 409) : json({ requests: [] })
    )
  );
  await userEvent.click(screen.getByRole('button', { name: 'Одобрить заявку' }));
  await screen.findByText('Входящих заявок по выбранным условиям нет.');
  expect(screen.getByRole('alert')).toHaveTextContent('Состояние уже изменилось');
});

test('first conversation is indexed if navigation happens while reply is in flight', async () => {
  await login();
  let release: (v: Response) => void = () => {};
  const original = fetch;
  vi.stubGlobal(
    'fetch',
    vi.fn((p: string, init: RequestInit) =>
      p === '/api/chat'
        ? new Promise<Response>(resolve => {
            release = resolve;
          })
        : original(p, init)
    )
  );
  await userEvent.type(screen.getByLabelText('Сообщение'), 'Долгий вопрос');
  await userEvent.click(screen.getByRole('button', { name: 'Отправить' }));
  await userEvent.click(screen.getByRole('link', { name: 'План адаптации' }));
  await screen.findByRole('heading', { name: 'Большой старт. Маленькими шагами.' });
  release(
    json({
      conversation_id: 'late-id',
      reply: 'Поздний ответ',
      tools_used: [],
      pending_actions: [],
    })
  );
  await screen.findByRole('link', { name: 'Долгий вопрос' });
  expect(location.hash).toBe('#plan');
});
test('pending actions are visible after first chat response is lost', async () => {
  await login();
  const original = fetch;
  vi.stubGlobal(
    'fetch',
    vi.fn((p: string, init: RequestInit) =>
      p === '/api/chat' ? Promise.reject(Error('lost response')) : original(p, init)
    )
  );
  await userEvent.type(screen.getByLabelText('Сообщение'), 'Создать доступ');
  await userEvent.click(screen.getByRole('button', { name: 'Отправить' }));
  await screen.findByRole('alert');
  expect(screen.getByRole('button', { name: 'Подтвердить отправку' })).toBeInTheDocument();
});
