export type Role = 'intern' | 'mentor' | 'manager' | 'hr_bp';
export interface Identity {
  user_id: string;
  role: Role;
  team: string;
}
export type StepStatus = 'not_started' | 'in_progress' | 'waiting' | 'done';
export interface Step {
  id: string;
  scenario_code: string;
  scenario_title: string;
  position: number;
  title: string;
  description: string;
  confirm_by: 'self' | 'mentor';
  status: StepStatus;
  updated_at?: string;
}
export interface Progress {
  steps: Step[];
  done: number;
  total: number;
  percent: number;
  next: Step | null;
  waiting: Step[];
}
export type ActionStatus =
  'pending_confirmation' | 'running' | 'done' | 'failed' | 'cancelled' | 'denied';
export interface Payload {
  kind?: string;
  date_from?: string;
  date_to?: string;
  reason?: string;
  project?: string;
  level?: string;
  justification?: string;
}
export interface Action {
  id: string;
  tool: string;
  args: Payload;
  result?: { request_id?: string; status?: string; approver_role?: string };
  status: ActionStatus;
  external_id?: string;
  error?: string;
  summary?: string;
  created_at: string;
}
export type RequestStatus = 'pending_approval' | 'approved' | 'rejected';
export interface Request {
  id: string;
  kind: 'absence' | 'access';
  user_id: string;
  user_name?: string;
  payload: Payload;
  status: RequestStatus;
  approver_role: string;
  decision_comment?: string;
  created_at: string;
  decided_at?: string;
}
export interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  created_at?: string;
  tools?: string[];
  actions?: Action[];
}
export interface ChatResponse {
  conversation_id: string;
  reply: string;
  tools_used: string[];
  pending_actions: Action[];
}
export interface Confirmation {
  intern_id: string;
  intern_name: string;
  step_id: string;
  step_title: string;
  since: string;
}
export interface Metrics {
  since: string;
  events: Record<string, number>;
  tools: Record<string, number>;
  active_users: number;
}
export interface Conversation {
  id: string;
  title: string;
}
