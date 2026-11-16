import { render, screen, waitFor, within, cleanup } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, expect, test, vi } from 'vitest';
import { App } from './App';
const pending = {
  id: 'action-1',
  tool: 'create_absence_request',
  status: 'pending_confirmation',
  args: { kind: 'vacation', date_from: '2030-01-10', date_to: '2030-01-11' },
  created_at: '2026-11-16',
  summary: 'Отпуск',
};
let actions = [pending];
let calls: { path: string; body: Record<string, unknown> | undefined }[] = [];
beforeEach(() => {
  actions = [{ ...pending }];
  calls = [];
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: string, init: RequestInit) => {
      const path = String(input);
      const body = init.body ? JSON.parse(String(init.body)) : undefined;
      calls.push({ path, body });
      let value: unknown = {};
      if (path === '/api/progress')
        value = { steps: [], done: 0, total: 0, percent: 0, next: null, waiting: [] };
      else if (path === '/api/me') value = { user_id: 'intern-1', role: 'intern', team: 'Команда' };
      else if (path === '/api/chat')
        value = {
          conversation_id: 'conversation-1',
          reply: 'Проверьте заявку',
          tools_used: ['create_absence_request'],
          pending_actions: actions,
        };
      else if (path.endsWith('/confirm')) {
        actions = [{ ...pending, status: 'done' }];
        value = actions[0];
      } else if (path === '/api/actions') value = { actions };
      else if (path === '/api/requests') value = { requests: [] };
      else if (path.includes('/messages'))
        value = {
          messages: [
            { id: 'm1', role: 'user', content: 'Сохранённый вопрос', created_at: '2026-11-16' },
          ],
        };
      return new Response(JSON.stringify(value), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      });
    })
  );
});
async function login() {
  render(<App />);
  await userEvent.click(screen.getByRole('button', { name: /Анна/ }));
  await screen.findByRole('button', { name: 'Открыть меню' });
}
test('mobile navigation and per-user server history survive remount', async () => {
  await login();
  await userEvent.type(screen.getByRole('textbox', { name: 'Сообщение' }), 'Оформи отпуск');
  await userEvent.click(screen.getByRole('button', { name: 'Отправить' }));
  await screen.findByText('Проверьте заявку');
  const link = screen.getByRole('link', { name: 'Оформи отпуск' });
  expect(link).toHaveAttribute('href', '#chat/conversation-1');
  cleanup();
  render(<App />);
  await screen.findByText('Сохранённый вопрос');
  expect(calls.some(c => c.path === '/api/conversations/conversation-1/messages')).toBe(true);
});
test('confirm updates the chat card and prevents duplicate execution', async () => {
  await login();
  await userEvent.type(screen.getByRole('textbox', { name: 'Сообщение' }), 'Оформи отпуск');
  await userEvent.click(screen.getByRole('button', { name: 'Отправить' }));
  const confirm = await screen.findByRole('button', { name: 'Подтвердить отправку' });
  await userEvent.dblClick(confirm);
  await waitFor(() =>
    expect(screen.queryByRole('button', { name: 'Подтвердить отправку' })).not.toBeInTheDocument()
  );
  expect(calls.filter(c => c.path.endsWith('/confirm'))).toHaveLength(1);
  expect(screen.getByText('Выполнено')).toBeInTheDocument();
});
test('requests expose server status filters and human readable payloads', async () => {
  await login();
  await userEvent.click(screen.getByRole('link', { name: 'Мои заявки' }));
  await userEvent.click(await screen.findByRole('tab', { name: 'Действия' }));
  await userEvent.click(screen.getByRole('tab', { name: 'Ожидают подтверждения' }));
  expect(screen.getByText('10.01.2030 — 11.01.2030')).toBeInTheDocument();
  expect(
    within(screen.getByRole('tabpanel')).getByRole('button', { name: 'Отменить' })
  ).toBeInTheDocument();
});
