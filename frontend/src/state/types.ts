export type ViewId = 'home' | 'plan' | 'materials' | 'requests';

export type TaskStatus = 'todo' | 'running' | 'done';

export interface Task {
  id: number;
  title: string;
  phase: string;
}

export interface Request {
  key: string;
  id?: string;
  type: 'absence' | 'access';
  title: string;
  detail: string;
  payload: AbsencePayload | AccessPayload;
  status: 'Отправка' | 'На согласовании' | 'Ошибка';
  error?: string;
  busy?: boolean;
}

export interface AbsencePayload {
  from: string;
  to: string;
  comment: string;
}

export interface AccessPayload {
  resource: string;
  reason: string;
}

export interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  sourceCard?: SourceCardData;
  form?: FormData;
  actions?: ActionButton[];
}

export interface SourceCardData {
  key: 'absence' | 'access' | 'readme' | 'mentor' | 'plan';
  title: string;
  quote: string;
}

export interface FormData {
  type: 'absence' | 'access';
  title: string;
  fields: FormField[];
}

export interface FormField {
  name: string;
  label: string;
  type: 'text' | 'textarea' | 'date';
  value: string;
  required?: boolean;
  readonly?: boolean;
  min?: string;
  maxLength?: number;
}

export interface ActionButton {
  label: string;
  action: () => void;
  variant?: 'primary' | 'text';
}

export interface Chat {
  id: string;
  title: string;
  messages: Message[];
}

export interface AppState {
  chats: Chat[];
  activeChatId: string | null;
  currentView: ViewId;
  sidebarOpen: boolean;
  messages: Message[];
  welcomeVisible: boolean;
  planStatuses: TaskStatus[];
  requests: Request[];
  serviceOffline: boolean;
}

export type AppAction =
  | { type: 'SELECT_CHAT'; payload: string }
  | { type: 'SET_VIEW'; payload: ViewId }
  | { type: 'TOGGLE_SIDEBAR' }
  | { type: 'CLOSE_SIDEBAR' }
  | { type: 'ADD_MESSAGE'; payload: Message }
  | { type: 'CLEAR_MESSAGES' }
  | { type: 'SET_WELCOME_VISIBLE'; payload: boolean }
  | { type: 'UPDATE_TASK_STATUS'; payload: { index: number; status: TaskStatus } }
  | { type: 'ADD_REQUEST'; payload: Request }
  | { type: 'UPDATE_REQUEST'; payload: { key: string; updates: Partial<Request> } }
  | { type: 'SET_SERVICE_OFFLINE'; payload: boolean }
  | { type: 'LOAD_STATE'; payload: AppState }
  | { type: 'RESET_STATE' };