// Client for the backend GET /status. Contract documented in the repo CLAUDE.md.

export type State = 'normal' | 'reduzida' | 'parcial' | 'paralisada' | 'encerrada' | 'desconhecido';

export type LineStatus = {
  code: number;
  name: string;
  colorHex: string;
  operator: string; // id, empty when the backend has no operator for the line
  operatorName: string;
  status: State;
  label: string;
  description: string;
};

export type StatusResponse = {
  updatedAt: string;
  source: string;
  stale: boolean;
  lines: LineStatus[];
};

const TIMEOUT_MS = 10_000;

export async function fetchStatus(baseUrl: string, fetchImpl: typeof fetch = fetch): Promise<StatusResponse> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), TIMEOUT_MS);
  try {
    const response = await fetchImpl(`${baseUrl.replace(/\/+$/, '')}/status`, { signal: controller.signal });
    if (!response.ok) {
      throw new Error(`GET /status failed with HTTP ${response.status}`);
    }
    const body = await response.json();
    if (!Array.isArray(body?.lines)) {
      throw new Error('GET /status returned an unexpected body');
    }
    return body as StatusResponse;
  } finally {
    clearTimeout(timer);
  }
}
