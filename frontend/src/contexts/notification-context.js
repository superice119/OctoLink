import { createContext, useContext, useEffect, useReducer, useCallback } from 'react';
import PropTypes from 'prop-types';
import io from 'socket.io-client';
import { useAuth } from 'src/hooks/use-auth';

const NotificationContext = createContext({ undefined });

const ACTIONS = {
  ADD: 'ADD',
  SET: 'SET',
  MARK_READ: 'MARK_READ',
  MARK_ALL_READ: 'MARK_ALL_READ',
  CLEAR: 'CLEAR',
};

function reducer(state, action) {
  switch (action.type) {
    case ACTIONS.ADD:
      return {
        ...state,
        items: [action.payload, ...state.items],
        unread: state.unread + 1,
        total: state.total + 1,
      };
    case ACTIONS.SET:
      return {
        ...state,
        items: action.payload.notifications || [],
        total: action.payload.total || 0,
        unread: action.payload.unread || 0,
      };
    case ACTIONS.MARK_READ:
      return {
        ...state,
        items: state.items.map((item) => (
          action.payload.includes(item.id) ? { ...item, read: true } : item
        )),
        unread: Math.max(0, state.unread - action.payload.length),
      };
    case ACTIONS.MARK_ALL_READ:
      return {
        ...state,
        items: state.items.map((n) => ({ ...n, read: true })),
        unread: 0,
      };
    case ACTIONS.CLEAR:
      return { items: [], total: 0, unread: 0 };
    default:
      return state;
  }
}

export const NotificationProvider = ({ children }) => {
  const auth = useAuth();
  const [state, dispatch] = useReducer(reducer, { items: [], total: 0, unread: 0 });

  const fetchHistory = useCallback(async (page = 1, pageSize = 20) => {
    try {
      const token = typeof window !== 'undefined' ? localStorage.getItem('token') : '';
      const res = await fetch(
        `${process.env.NEXT_PUBLIC_REST_ENDPOINT || ''}/api/notifications?page=${page}&page_size=${pageSize}`,
        { headers: { Authorization: token, 'Content-Type': 'application/json' } }
      );
      if (res.ok) {
        const data = await res.json();
        dispatch({ type: ACTIONS.SET, payload: data });
      }
    } catch (e) {
      console.error('[Notifications] fetchHistory error:', e);
    }
  }, []);

  const markAllRead = useCallback(async () => {
    try {
      const token = typeof window !== 'undefined' ? localStorage.getItem('token') : '';
      const res = await fetch(`${process.env.NEXT_PUBLIC_REST_ENDPOINT || ''}/api/notifications/read`, {
        method: 'PUT',
        headers: { Authorization: token, 'Content-Type': 'application/json' },
        body: JSON.stringify({ all: true }),
      });
      if (res.ok) {
        dispatch({ type: ACTIONS.MARK_ALL_READ });
      }
    } catch (e) {
      console.error('[Notifications] markAllRead error:', e);
    }
  }, []);

  const clearAll = useCallback(async () => {
    try {
      const token = typeof window !== 'undefined' ? localStorage.getItem('token') : '';
      const res = await fetch(`${process.env.NEXT_PUBLIC_REST_ENDPOINT || ''}/api/notifications`, {
        method: 'DELETE',
        headers: { Authorization: token },
      });
      if (res.ok) {
        dispatch({ type: ACTIONS.CLEAR });
      }
    } catch (e) {
      console.error('[Notifications] clearAll error:', e);
    }
  }, []);

  useEffect(() => {
    if (!auth.isAuthenticated) {
      return undefined;
    }

    fetchHistory();
    const token = typeof window !== 'undefined' ? localStorage.getItem('token') : '';
    const socket = io(process.env.NEXT_PUBLIC_WS_ENDPOINT || 'http://localhost:5000', {
      auth: { token: token || '' },
    });
    socket.on('connect_error', (err) => {
      console.error('[Notifications] Socket connect error:', err.message);
    });
    socket.on('usp_notify', (notification) => {
      console.log('[Notifications] Real-time notify:', notification);
      dispatch({ type: ACTIONS.ADD, payload: { ...notification, read: false } });
    });

    return () => {
      socket.disconnect();
    };
  }, [auth.isAuthenticated, fetchHistory]);

  return (
    <NotificationContext.Provider value={{ ...state, fetchHistory, markAllRead, clearAll }}>
      {children}
    </NotificationContext.Provider>
  );
};

NotificationProvider.propTypes = { children: PropTypes.node };
export const useNotifications = () => useContext(NotificationContext);
export default NotificationContext;
