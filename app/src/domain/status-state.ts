import type { StatusResponse } from '@/api/status';

export type StatusState = {
  data?: StatusResponse;
  loading: boolean; // first load, nothing to show yet
  refreshing: boolean; // manual pull to refresh
  error: boolean; // last attempt failed, or the backend is serving stale data
};

export type StatusAction = { type: 'refresh' } | { type: 'success'; data: StatusResponse } | { type: 'failure' };

export const initialStatusState: StatusState = { loading: true, refreshing: false, error: false };

export function statusReducer(state: StatusState, action: StatusAction): StatusState {
  switch (action.type) {
    case 'refresh':
      return { ...state, refreshing: true };
    case 'success':
      return { data: action.data, loading: false, refreshing: false, error: action.data.stale };
    case 'failure':
      return { data: state.data, loading: false, refreshing: false, error: true };
  }
}
