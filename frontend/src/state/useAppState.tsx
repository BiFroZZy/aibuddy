import React, { useEffect, useCallback, ReactNode, useSyncExternalStore } from 'react';
import type { AppState, AppAction, ViewId, Message, Request, TaskStatus } from './types';
import { initialState, appReducer, loadState } from './reducer';

// Create a simple store with subscribe/notify pattern
function createStore() {
  let state = loadState();
  const listeners = new Set<() => void>();

  return {
    getState: () => state,
    setState: (newState: AppState) => {
      state = newState;
      listeners.forEach((l) => l());
    },
    dispatch: (action: AppAction) => {
      state = appReducer(state, action);
      listeners.forEach((l) => l());
    },
    subscribe: (listener: () => void) => {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
  };
}

// Global store instance
const store = createStore();

// AppProvider component that initializes the store and provides escape key handling
export function AppProvider({ children }: { children: ReactNode }) {
  // Use a dummy state to trigger re-renders when store changes
  const [, forceUpdate] = React.useState(0);

  useEffect(() => {
    const unsubscribe = store.subscribe(() => {
      forceUpdate((n) => n + 1);
    });
    return unsubscribe;
  }, []);

  // Handle Escape key to close sidebar
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        const currentState = store.getState();
        if (currentState.sidebarOpen) {
          store.dispatch({ type: 'CLOSE_SIDEBAR' });
        }
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, []);

  return <React.Fragment>{children}</React.Fragment>;
}

function useStoreSelector<T>(selector: (state: AppState) => T): T {
  return useSyncExternalStore(
    store.subscribe,
    () => selector(store.getState()),
    () => selector(initialState)
  );
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAppState() {
  const state = useStoreSelector((s) => s);

  const setView = useCallback((view: ViewId) => store.dispatch({ type: 'SET_VIEW', payload: view }), []);
  const toggleSidebar = useCallback(() => store.dispatch({ type: 'TOGGLE_SIDEBAR' }), []);
  const closeSidebar = useCallback(() => store.dispatch({ type: 'CLOSE_SIDEBAR' }), []);
  const addMessage = useCallback((message: Message) => store.dispatch({ type: 'ADD_MESSAGE', payload: message }), []);
  const clearMessages = useCallback(() => store.dispatch({ type: 'CLEAR_MESSAGES' }), []);
  const setWelcomeVisible = useCallback((visible: boolean) => store.dispatch({ type: 'SET_WELCOME_VISIBLE', payload: visible }), []);
  const updateTaskStatus = useCallback((index: number, status: TaskStatus) => store.dispatch({ type: 'UPDATE_TASK_STATUS', payload: { index, status } }), []);
  const addRequest = useCallback((request: Request) => store.dispatch({ type: 'ADD_REQUEST', payload: request }), []);
  const updateRequest = useCallback((key: string, updates: Partial<Request>) => store.dispatch({ type: 'UPDATE_REQUEST', payload: { key, updates } }), []);
  const setServiceOffline = useCallback((offline: boolean) => store.dispatch({ type: 'SET_SERVICE_OFFLINE', payload: offline }), []);

  return {
    ...state,
    setView,
    toggleSidebar,
    closeSidebar,
    addMessage,
    clearMessages,
    setWelcomeVisible,
    updateTaskStatus,
    addRequest,
    updateRequest,
    setServiceOffline,
  };
}