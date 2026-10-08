import type { StatusResponse } from '@/api/status';

import { initialStatusState, statusReducer } from './status-state';

const fresh: StatusResponse = { updatedAt: '2026-10-08T01:30:00Z', source: 'proximotrem', stale: false, lines: [] };
const stale: StatusResponse = { ...fresh, stale: true };

describe('statusReducer', () => {
  it('starts loading with no data', () => {
    expect(initialStatusState).toEqual({ loading: true, refreshing: false, error: false });
  });

  it('shows data after the first load', () => {
    const s = statusReducer(initialStatusState, { type: 'success', data: fresh });

    expect(s).toEqual({ data: fresh, loading: false, refreshing: false, error: false });
  });

  it('marks a manual refresh separately from the first load', () => {
    const loaded = statusReducer(initialStatusState, { type: 'success', data: fresh });

    const s = statusReducer(loaded, { type: 'refresh' });

    expect(s.refreshing).toBe(true);
    expect(s.loading).toBe(false);
    expect(s.data).toBe(fresh);
  });

  it('keeps the last data when a refresh fails', () => {
    const loaded = statusReducer(initialStatusState, { type: 'success', data: fresh });

    const s = statusReducer(statusReducer(loaded, { type: 'refresh' }), { type: 'failure' });

    expect(s).toEqual({ data: fresh, loading: false, refreshing: false, error: true });
  });

  it('reports an error when the first load fails', () => {
    const s = statusReducer(initialStatusState, { type: 'failure' });

    expect(s).toEqual({ data: undefined, loading: false, refreshing: false, error: true });
  });

  it('treats stale data from the backend as an error to show', () => {
    const s = statusReducer(initialStatusState, { type: 'success', data: stale });

    expect(s.error).toBe(true);
    expect(s.data).toBe(stale);
  });

  it('clears the error after a successful retry', () => {
    const failed = statusReducer(initialStatusState, { type: 'failure' });

    const s = statusReducer(failed, { type: 'success', data: fresh });

    expect(s.error).toBe(false);
  });
});
