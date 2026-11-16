export function date(value?: string) {
  if (!value) return '—';
  const d = new Date(value);
  return Number.isNaN(d.getTime())
    ? '—'
    : new Intl.DateTimeFormat('ru-RU', { timeZone: 'UTC' }).format(d);
}
const names: Record<string, string> = {
  day_off: 'Отгул',
  vacation: 'Отпуск',
  sick: 'Больничный',
  read: 'Чтение',
  write: 'Запись',
  admin: 'Администрирование',
  manager: 'Руководитель',
  project_owner: 'Владелец проекта',
  devops: 'DevOps-инженер',
};
export const label = (v: string) => names[v] || v;
