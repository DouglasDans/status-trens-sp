import { fetchStatus, type StatusResponse } from './status';

const body: StatusResponse = {
  updatedAt: '2026-10-08T01:30:38Z',
  source: 'proximotrem',
  stale: false,
  lines: [
    {
      code: 9,
      name: 'Esmeralda',
      colorHex: '#219896',
      operator: 'viamobilidade',
      operatorName: 'ViaMobilidade',
      status: 'reduzida',
      label: 'Atividade Programada',
      description: 'Maiores intervalos.',
    },
  ],
};

function fakeFetch(response: Partial<Response>) {
  return jest.fn().mockResolvedValue({ ok: true, status: 200, json: async () => body, ...response });
}

describe('fetchStatus', () => {
  it('requests GET /status from the base URL and returns the body', async () => {
    const fetchImpl = fakeFetch({});

    const result = await fetchStatus('http://192.168.0.10:8080', fetchImpl);

    expect(fetchImpl).toHaveBeenCalledWith('http://192.168.0.10:8080/status', expect.anything());
    expect(result).toEqual(body);
  });

  it('accepts a base URL with a trailing slash', async () => {
    const fetchImpl = fakeFetch({});

    await fetchStatus('http://localhost:8080/', fetchImpl);

    expect(fetchImpl).toHaveBeenCalledWith('http://localhost:8080/status', expect.anything());
  });

  it('fails on a non-2xx response', async () => {
    const fetchImpl = fakeFetch({ ok: false, status: 503 });

    await expect(fetchStatus('http://localhost:8080', fetchImpl)).rejects.toThrow('503');
  });

  it('fails when the body has no lines', async () => {
    const fetchImpl = fakeFetch({ json: async () => ({ error: 'x' }) });

    await expect(fetchStatus('http://localhost:8080', fetchImpl)).rejects.toThrow();
  });

  it('fails when the network fails', async () => {
    const fetchImpl = jest.fn().mockRejectedValue(new TypeError('Network request failed'));

    await expect(fetchStatus('http://localhost:8080', fetchImpl)).rejects.toThrow('Network request failed');
  });
});
