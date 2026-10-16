import AsyncStorage from '@react-native-async-storage/async-storage';
import { act, fireEvent, render, screen } from '@testing-library/react-native';
import { AppState, ScrollView } from 'react-native';

import type { LineStatus, StatusResponse } from '@/api/status';
import type { PermissionState, Push } from '@/push';

import { StatusScreen } from './status-screen';

jest.mock('expo-router', () => ({ Link: ({ children }: { children: React.ReactNode }) => children }));

function line(code: number, name: string, status: LineStatus['status'], label = '', description = ''): LineStatus {
  return { code, name, colorHex: '#219896', textColorHex: '#ffffff', operator: 'viamobilidade', operatorName: 'ViaMobilidade', status, label, description };
}

const day: StatusResponse = {
  updatedAt: new Date(2026, 9, 7, 22, 21).toISOString(),
  source: 'proximotrem',
  stale: false,
  lines: [
    line(7, 'Rubi', 'normal', 'Operação Normal', 'Linha opera normalmente em toda sua extensão.'),
    line(9, 'Esmeralda', 'reduzida', 'Atividade Programada', 'Maiores intervalos entre Osasco e Varginha.'),
  ],
};

beforeEach(() => AsyncStorage.clear());

describe('StatusScreen', () => {
  it('shows every line with its status, description and update time', async () => {
    await render(<StatusScreen load={async () => day} />);

    expect(await screen.findByText('Esmeralda')).toBeOnTheScreen();
    expect(screen.getByText('Atividade Programada')).toBeOnTheScreen();
    expect(screen.getByText('Maiores intervalos entre Osasco e Varginha.')).toBeOnTheScreen();
    expect(screen.getByText('Atualizado às 22:21')).toBeOnTheScreen();
  });

  it('loads the status again when the user comes back to the app', async () => {
    const listen = jest.spyOn(AppState, 'addEventListener');
    const later = { ...day, updatedAt: new Date(2026, 9, 8, 7, 5).toISOString() };
    const load = jest.fn().mockResolvedValueOnce(day).mockResolvedValueOnce(later);
    await render(<StatusScreen load={load} />);
    expect(await screen.findByText('Atualizado às 22:21')).toBeOnTheScreen();

    const onChange = listen.mock.calls.find(([type]) => type === 'change')?.[1];
    await act(async () => onChange?.('active'));

    expect(await screen.findByText('Atualizado às 7:05')).toBeOnTheScreen();
  });

  it('invites the user to favorite when there are no favorites', async () => {
    await render(<StatusScreen load={async () => day} />);

    expect(await screen.findByText(/Toque na estrela/)).toBeOnTheScreen();
    expect(screen.getByText('TODAS AS LINHAS')).toBeOnTheScreen();
  });

  it('moves a line to favorites when its star is pressed', async () => {
    await render(<StatusScreen load={async () => day} />);

    await fireEvent.press(await screen.findByLabelText('Favoritar linha 9 Esmeralda'));

    expect(await screen.findByText('FAVORITAS')).toBeOnTheScreen();
    expect(screen.getByText('OUTRAS LINHAS')).toBeOnTheScreen();
    expect(screen.getByLabelText('Remover linha 9 Esmeralda dos favoritos')).toBeOnTheScreen();
    expect(JSON.parse((await AsyncStorage.getItem('favorites')) ?? '[]')).toEqual([9]);
  });

  it('scrolls to the top after favoriting, so the line does not vanish from view', async () => {
    const scrollTo = jest.spyOn(ScrollView.prototype, 'scrollTo');
    await render(<StatusScreen load={async () => day} />);

    await fireEvent.press(await screen.findByLabelText('Favoritar linha 9 Esmeralda'));

    expect(scrollTo).toHaveBeenCalledWith({ y: 0, animated: true });
  });

  it('does not scroll when removing a favorite', async () => {
    await AsyncStorage.setItem('favorites', '[9]');
    const scrollTo = jest.spyOn(ScrollView.prototype, 'scrollTo');
    await render(<StatusScreen load={async () => day} />);

    await fireEvent.press(await screen.findByLabelText('Remover linha 9 Esmeralda dos favoritos'));

    expect(scrollTo).not.toHaveBeenCalled();
  });

  it('keeps showing the last data with a warning when an update fails', async () => {
    const load = jest.fn().mockResolvedValueOnce(day).mockRejectedValueOnce(new Error('offline'));
    await render(<StatusScreen load={load} />);
    await screen.findByText('Esmeralda');

    await fireEvent.press(screen.getByText('Atualizado às 22:21'));

    expect(await screen.findByText('Não foi possível atualizar')).toBeOnTheScreen();
    expect(screen.getByText('Mostrando os dados das 22:21.')).toBeOnTheScreen();
    expect(screen.getByText('Esmeralda')).toBeOnTheScreen();
  });

  it('offers a retry when the first load fails', async () => {
    const load = jest.fn().mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce(day);
    await render(<StatusScreen load={load} />);

    await fireEvent.press(await screen.findByText('Tentar de novo'));

    expect(await screen.findByText('Esmeralda')).toBeOnTheScreen();
  });

  it('shows the operator logo when the app has one', async () => {
    await render(<StatusScreen load={async () => day} />);

    expect(await screen.findAllByLabelText('Operadora ViaMobilidade')).not.toHaveLength(0);
  });

  it('shows the operator name when the app has no logo for it', async () => {
    const data = { ...day, lines: [{ ...day.lines[0], operator: 'nova', operatorName: 'Nova Operadora' }] };
    await render(<StatusScreen load={async () => data} />);

    expect(await screen.findByText('Nova Operadora')).toBeOnTheScreen();
  });

  it('shows no operator badge for a line without operator', async () => {
    const data = { ...day, lines: [{ ...day.lines[0], name: 'Bronze', operator: '', operatorName: '' }] };
    await render(<StatusScreen load={async () => data} />);

    await screen.findByText('Bronze');
    expect(screen.queryByLabelText(/^Operadora/)).toBeNull();
  });

  it('shows the late night banner when every line is closed', async () => {
    const night = { ...day, lines: day.lines.map((l) => ({ ...l, status: 'encerrada' as const, label: 'Operação Encerrada' })) };
    await render(<StatusScreen load={async () => night} />);

    expect(await screen.findByText('Todas as linhas estão com operação encerrada')).toBeOnTheScreen();
    expect(screen.queryByText(/4h40/)).toBeNull();
  });

  describe('push', () => {
    function fakePush(permission: PermissionState = 'granted', answer: 'granted' | 'denied' = 'granted') {
      return {
        subscribe: jest.fn(async () => {}),
        unsubscribe: jest.fn(async () => {}),
        permission: jest.fn(async () => permission),
        requestPermission: jest.fn(async () => answer),
        openSettings: jest.fn(),
      } satisfies Push;
    }

    it('subscribes to a line when it is favorited', async () => {
      const push = fakePush();
      await render(<StatusScreen load={async () => day} push={push} />);

      await fireEvent.press(await screen.findByLabelText('Favoritar linha 9 Esmeralda'));

      expect(push.subscribe).toHaveBeenCalledWith(9);
    });

    it('unsubscribes from a line when it leaves the favorites', async () => {
      await AsyncStorage.setItem('favorites', '[9]');
      const push = fakePush();
      await render(<StatusScreen load={async () => day} push={push} />);

      await fireEvent.press(await screen.findByLabelText('Remover linha 9 Esmeralda dos favoritos'));

      expect(push.unsubscribe).toHaveBeenCalledWith(9);
    });

    it('subscribes again to every stored favorite on open', async () => {
      await AsyncStorage.setItem('favorites', '[7,9]');
      const push = fakePush();
      await render(<StatusScreen load={async () => day} push={push} />);

      await screen.findByLabelText('Remover linha 9 Esmeralda dos favoritos');
      expect(push.subscribe).toHaveBeenCalledWith(7);
      expect(push.subscribe).toHaveBeenCalledWith(9);
    });

    it('keeps the favorite when the subscription fails', async () => {
      const push = fakePush();
      push.subscribe.mockRejectedValueOnce(new Error('offline'));
      await render(<StatusScreen load={async () => day} push={push} />);

      await fireEvent.press(await screen.findByLabelText('Favoritar linha 9 Esmeralda'));

      expect(await screen.findByLabelText('Remover linha 9 Esmeralda dos favoritos')).toBeOnTheScreen();
    });

    it('explains the notifications on the first favorite, then asks Android', async () => {
      const push = fakePush('undetermined');
      await render(<StatusScreen load={async () => day} push={push} />);

      await fireEvent.press(await screen.findByLabelText('Favoritar linha 9 Esmeralda'));
      expect(await screen.findByText('Avisar quando suas linhas mudarem?')).toBeOnTheScreen();
      expect(push.requestPermission).not.toHaveBeenCalled();

      await fireEvent.press(screen.getByText('Continuar'));

      expect(push.requestPermission).toHaveBeenCalledTimes(1);
      expect(screen.queryByText('Avisar quando suas linhas mudarem?')).toBeNull();
    });

    it('keeps the favorite and asks nothing when the user says not now', async () => {
      const push = fakePush('undetermined');
      await render(<StatusScreen load={async () => day} push={push} />);

      await fireEvent.press(await screen.findByLabelText('Favoritar linha 9 Esmeralda'));
      await fireEvent.press(await screen.findByText('Agora não'));

      expect(push.requestPermission).not.toHaveBeenCalled();
      expect(screen.queryByText('Avisar quando suas linhas mudarem?')).toBeNull();
      expect(screen.getByLabelText('Remover linha 9 Esmeralda dos favoritos')).toBeOnTheScreen();
      expect(push.subscribe).toHaveBeenCalledWith(9);
    });

    it('does not explain anything when notifications are already allowed', async () => {
      await render(<StatusScreen load={async () => day} push={fakePush('granted')} />);

      await fireEvent.press(await screen.findByLabelText('Favoritar linha 9 Esmeralda'));

      expect(screen.queryByText('Avisar quando suas linhas mudarem?')).toBeNull();
    });

    it('warns that notifications are off once the user denies them', async () => {
      const push = fakePush('undetermined', 'denied');
      await render(<StatusScreen load={async () => day} push={push} />);

      await fireEvent.press(await screen.findByLabelText('Favoritar linha 9 Esmeralda'));
      await fireEvent.press(await screen.findByText('Continuar'));

      expect(await screen.findByText('Notificações desativadas')).toBeOnTheScreen();
    });

    it('sends the user to the settings from the denied banner', async () => {
      await AsyncStorage.setItem('favorites', '[9]');
      const push = fakePush('denied');
      await render(<StatusScreen load={async () => day} push={push} />);

      await fireEvent.press(await screen.findByText('Ativar nas configurações'));

      expect(push.openSettings).toHaveBeenCalled();
    });

    it('does not warn about notifications without favorites', async () => {
      await render(<StatusScreen load={async () => day} push={fakePush('denied')} />);

      await screen.findByText('Esmeralda');
      expect(screen.queryByText('Notificações desativadas')).toBeNull();
    });

    it('checks the permission again when the user comes back to the app', async () => {
      await AsyncStorage.setItem('favorites', '[9]');
      const listen = jest.spyOn(AppState, 'addEventListener');
      const push = fakePush('denied');
      await render(<StatusScreen load={async () => day} push={push} />);
      expect(await screen.findByText('Notificações desativadas')).toBeOnTheScreen();

      push.permission.mockResolvedValue('granted');
      const onChange = listen.mock.calls.find(([type]) => type === 'change')?.[1];
      await act(async () => onChange?.('active'));

      expect(screen.queryByText('Notificações desativadas')).toBeNull();
    });
  });
});
