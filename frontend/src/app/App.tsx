import { useEffect } from 'react';
import { useAppState } from '@/state/useAppState';
import { AppProvider } from '@/state/useAppState';
import { AppLayout } from './AppLayout';
import { routeIntent } from '@/utils/intentRouter';
import { nextIndex } from '@/state/reducer';
import type { AbsencePayload, AccessPayload, Request } from '@/state/types';

function GlobalEventHandlers() {
  const {
    addMessage,
    setView,
    addRequest,
    updateRequest,
    planStatuses,
    requests,
    serviceOffline,
    setServiceOffline,
  } = useAppState();

  useEffect(() => {
    const handleFormSubmit = (e: CustomEvent<{ formData: Record<string, string>; messageId: string }>) => {
      const { formData } = e.detail;
      const isAccess = 'resource' in formData;

      // Create request object
      const key = crypto.randomUUID ? crypto.randomUUID() : Date.now() + '-' + Math.random();
      const expires = Date.now() + 600000;
      let consumed = false;

      const request: Request = {
        key,
        payload: isAccess ? formData as unknown as AccessPayload : formData as unknown as AbsencePayload,
        type: isAccess ? 'access' : 'absence',
        title: isAccess ? `Доступ к ${formData.resource}` : `Отгул · ${formData.from} — ${formData.to}`,
        detail: isAccess ? `Чтение · ${formData.reason}` : `Отгул · ${formData.comment || 'Без комментария'}`,
        status: 'Отправка',
      };

      addRequest(request);

      // Simulate execution
      const execute = () => {
        if (consumed) return;
        if (Date.now() > expires) {
          // Expired - would need to re-prepare
          return;
        }
        consumed = true;

        updateRequest(key, { status: 'Отправка', busy: true });

        setTimeout(() => {
          if (serviceOffline) {
            updateRequest(key, {
              status: 'Ошибка',
              error: 'Сервис недоступен. Заявка не отправлена. Включи сервис и повтори попытку.',
              busy: false,
            });
          } else {
            const idx = requests.length;
            const newId = `DEMO-${String(idx + 1).padStart(3, '0')}`;
            updateRequest(key, {
              status: 'На согласовании',
              id: newId,
              error: '',
              busy: false,
            });
            // Add link to requests view
            addMessage({
              id: crypto.randomUUID ? crypto.randomUUID() : Date.now().toString(),
              role: 'assistant',
              content: `Заявка ${newId} · На согласовании. Имитация сервиса сохранила заявку. Это ещё не одобрение. Следующий шаг — решение руководителя.`,
              actions: [{ label: 'Открыть мои заявки →', action: () => setView('requests') }],
            });
          }
        }, 700);
      };

      execute();
    };

    const handleAskPrompt = (e: CustomEvent<string>) => {
      const prompt = e.detail;
      const nextTaskIdx = nextIndex(planStatuses);
      const nextTaskText = nextTaskIdx < 0 ? 'Все шаги завершены. Обсуди с наставником дальнейшие цели.' : `Следующий шаг: ${[
        'Познакомиться с наставником',
        'Представиться команде',
        'Получить доступ к репозиторию',
        'Настроить окружение',
        'Изучить процессы команды',
        'Обсудить цели стажировки',
        'Выбрать первую задачу',
        'Договориться о первом ревью',
      ][nextTaskIdx]}`;

      const result = routeIntent(prompt, nextTaskText, setView);

      const assistantMessage = {
        id: crypto.randomUUID ? crypto.randomUUID() : Date.now().toString(),
        role: 'assistant' as const,
        content: result.message,
      } as import('@/state/types').Message;

      if (result.sourceCard) {
        assistantMessage.sourceCard = result.sourceCard;
      }

      if (result.showForm) {
        assistantMessage.form = {
          type: result.showForm,
          title: result.showForm === 'absence' ? 'Оформить отгул' : 'Запросить доступ к репозиторию',
          fields: result.showForm === 'absence'
            ? [
                { name: 'from', label: 'Первый день', type: 'date' as const, value: '', required: true, min: new Date().toISOString().split('T')[0] },
                { name: 'to', label: 'Последний день', type: 'date' as const, value: '', required: true, min: new Date().toISOString().split('T')[0] },
                { name: 'comment', label: 'Комментарий (необязательно)', type: 'textarea' as const, value: '', required: false, maxLength: 200 },
              ]
            : [
                { name: 'resource', label: 'Учебный репозиторий', type: 'text' as const, value: 'backend-onboarding', required: true, readonly: true },
                { name: 'reason', label: 'Обоснование', type: 'textarea' as const, value: '', required: true, maxLength: 200 },
              ],
        };
      }

      if (result.actions) {
        assistantMessage.actions = result.actions;
      }

      addMessage(assistantMessage);
    };

    const handleSetServiceOffline = (e: Event) => { setServiceOffline((e as CustomEvent<boolean>).detail); };

    window.addEventListener('form-submit', handleFormSubmit as EventListener);
    window.addEventListener('ask-prompt', handleAskPrompt as EventListener);
    window.addEventListener('set-service-offline', handleSetServiceOffline as EventListener);

    return () => {
      window.removeEventListener('form-submit', handleFormSubmit as EventListener);
      window.removeEventListener('ask-prompt', handleAskPrompt as EventListener);
      window.removeEventListener('set-service-offline', handleSetServiceOffline as EventListener);
    };
  }, [addMessage, setView, addRequest, updateRequest, planStatuses, requests, serviceOffline, setServiceOffline]);

  return null;
}

export function App() {
  return (
    <AppProvider>
      <AppLayout />
      <GlobalEventHandlers />
    </AppProvider>
  );
}