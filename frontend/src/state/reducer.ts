import type { AppState, AppAction, TaskStatus } from './types';

const TASKS = [
  'Познакомиться с наставником',
  'Представиться команде',
  'Получить доступ к репозиторию',
  'Настроить окружение',
  'Изучить процессы команды',
  'Обсудить цели стажировки',
  'Выбрать первую задачу',
  'Договориться о первом ревью',
];

export const initialState: AppState = {
  currentView: 'home',
  sidebarOpen: false,
  messages: [],
  welcomeVisible: true,
  planStatuses: TASKS.map((_, i) => (i < 2 ? 'done' : i === 2 ? 'running' : 'todo')),
  requests: [],
  serviceOffline: false,
};

export function loadState(): AppState {
  try {
    const stored = localStorage.getItem('buddy-demo-v2');
    if (stored) {
      const parsed = JSON.parse(stored);
      parsed.planStatuses ??= parsed.statuses;
      if (parsed && Array.isArray(parsed.planStatuses) && parsed.planStatuses.length === 8 && Array.isArray(parsed.requests)) {
        return {
          ...initialState,
          ...parsed,
          planStatuses: parsed.planStatuses.map((s: string) => ['done', 'running', 'todo'].includes(s) ? s : 'todo'),
        };
      }
    }
    // Try v1 migration
    const v1 = localStorage.getItem('buddy-demo-v1');
    if (v1) {
      const parsed = JSON.parse(v1);
      if (parsed && Array.isArray(parsed.done)) {
        const statuses: TaskStatus[] = TASKS.map((_, i) => (parsed.done.includes(i) ? 'done' : 'todo'));
        return {
          ...initialState,
          planStatuses: statuses,
          requests: Array.isArray(parsed.requests) ? parsed.requests : [],
        };
      }
    }
  } catch {
    // Ignore parse errors
  }
  return initialState;
}

export function saveState(state: AppState): void {
  try {
    localStorage.setItem('buddy-demo-v2', JSON.stringify({ planStatuses: state.planStatuses, requests: state.requests }));
  } catch {
    // Ignore storage errors
  }
}

export function nextIndex(statuses: TaskStatus[]): number {
  return statuses.findIndex((s) => s !== 'done');
}

export function appReducer(state: AppState, action: AppAction): AppState {
  let newState: AppState;

  switch (action.type) {
    case 'SET_VIEW':
      newState = { ...state, currentView: action.payload, sidebarOpen: false };
      break;

    case 'TOGGLE_SIDEBAR':
      newState = { ...state, sidebarOpen: !state.sidebarOpen };
      break;

    case 'CLOSE_SIDEBAR':
      newState = { ...state, sidebarOpen: false };
      break;

    case 'ADD_MESSAGE':
      newState = {
        ...state,
        messages: [...state.messages, action.payload],
        welcomeVisible: false,
      };
      break;

    case 'CLEAR_MESSAGES':
      newState = { ...state, messages: [], welcomeVisible: true };
      break;

    case 'SET_WELCOME_VISIBLE':
      newState = { ...state, welcomeVisible: action.payload };
      break;

    case 'UPDATE_TASK_STATUS': {
      const { index, status } = action.payload;
      const newStatuses = [...state.planStatuses];
      newStatuses[index] = status;
      newState = { ...state, planStatuses: newStatuses };
      break;
    }

    case 'ADD_REQUEST': {
      const newRequest = action.payload;
      newState = { ...state, requests: [...state.requests, newRequest] };
      break;
    }

    case 'UPDATE_REQUEST': {
      const { key, updates } = action.payload;
      newState = {
        ...state,
        requests: state.requests.map((r) => (r.key === key ? { ...r, ...updates } : r)),
      };
      break;
    }

    case 'SET_SERVICE_OFFLINE':
      newState = { ...state, serviceOffline: action.payload };
      break;

    case 'LOAD_STATE':
      newState = { ...state, ...action.payload };
      break;

    case 'RESET_STATE':
      newState = initialState;
      break;

    default:
      return state;
  }

  // Auto-save on every change
  saveState(newState);
  return newState;
}