import type { Conversation, Role } from '../api/types';
export type Page = 'chat' | 'plan' | 'requests' | 'mentor' | 'approvals' | 'metrics' | 'materials';
export const labels: Record<Page, string> = {
  chat: 'Мой buddy',
  plan: 'План адаптации',
  requests: 'Мои заявки',
  mentor: 'Кабинет наставника',
  approvals: 'Согласования',
  metrics: 'Метрики',
  materials: 'Рабочие материалы',
};
export const roleLabels: Record<Role, string> = {
  intern: 'Стажёр',
  mentor: 'Наставник',
  manager: 'Руководитель',
  hr_bp: 'HR BP',
};
export const allowed: Record<Role, Page[]> = {
  intern: ['chat', 'plan', 'requests', 'materials'],
  mentor: ['mentor', 'chat', 'materials'],
  manager: ['approvals', 'metrics', 'chat', 'materials'],
  hr_bp: ['metrics', 'chat', 'materials'],
};
export const profiles = [
  { name: 'Анна Стажёр', token: 'dev-intern-1', role: 'intern' },
  { name: 'Борис Стажёр', token: 'dev-intern-2', role: 'intern' },
  { name: 'Михаил Наставник', token: 'dev-mentor', role: 'mentor' },
  { name: 'Мария Руководитель', token: 'dev-manager', role: 'manager' },
  { name: 'Ольга HR', token: 'dev-hr', role: 'hr_bp' },
] as const;
export const statusLabels: Record<string, string> = {
  not_started: 'Не начато',
  in_progress: 'В работе',
  waiting: 'Ожидает подтверждения наставником',
  done: 'Выполнено',
  pending_confirmation: 'Ожидает подтверждения',
  running: 'Выполняется',
  failed: 'Ошибка',
  cancelled: 'Отменено',
  denied: 'Отказано',
  pending_approval: 'На согласовании',
  approved: 'Одобрена',
  rejected: 'Отклонена',
};
export function route(): { page: string; conversation?: string } {
  const [page = 'chat', conversation] = location.hash.slice(1).split('/');
  return { page, conversation };
}
export function conversations(user: string): Conversation[] {
  try {
    const value: unknown = JSON.parse(localStorage.getItem('buddy-conversations:' + user) || '[]');
    return Array.isArray(value)
      ? value.filter(
          (c): c is Conversation => !!c && typeof c.id === 'string' && typeof c.title === 'string'
        )
      : [];
  } catch {
    return [];
  }
}
export function remember(user: string, entry: Conversation) {
  const list = conversations(user);
  const next = list.some(c => c.id === entry.id) ? list : [entry, ...list];
  try {
    localStorage.setItem('buddy-conversations:' + user, JSON.stringify(next));
  } catch {
    /* History index is optional when browser storage is unavailable. */
  }
  return next;
}
export function sessionToken() {
  try {
    return sessionStorage.getItem('buddy-token') || '';
  } catch {
    return '';
  }
}
export function saveToken(token: string) {
  try {
    if (token) sessionStorage.setItem('buddy-token', token);
    else sessionStorage.removeItem('buddy-token');
  } catch {
    /* The active session can still live in memory. */
  }
}
