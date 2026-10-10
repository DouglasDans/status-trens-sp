import AsyncStorage from '@react-native-async-storage/async-storage';
import { fireEvent, render, screen } from '@testing-library/react-native';
import { ScrollView } from 'react-native';

import type { LineStatus, StatusResponse } from '@/api/status';

import { StatusScreen } from './status-screen';

jest.mock('expo-router', () => ({ Link: ({ children }: { children: React.ReactNode }) => children }));

function line(code: number, name: string, status: LineStatus['status'], label = '', description = ''): LineStatus {
  return { code, name, colorHex: '#219896', operator: 'viamobilidade', operatorName: 'ViaMobilidade', status, label, description };
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

    expect(await screen.findByText('Operação encerrada — volta às 4h40')).toBeOnTheScreen();
  });
});
