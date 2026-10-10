import type { LineStatus } from '@/api/status';

import { formatClock, groupLines, isNight } from './lines';

function line(code: number, status: LineStatus['status'] = 'normal'): LineStatus {
  return { code, name: `L${code}`, colorHex: '#000000', textColorHex: '#ffffff', operator: 'metro', operatorName: 'Metrô', status, label: '', description: '' };
}

describe('groupLines', () => {
  const lines = [line(1), line(7), line(9), line(15)];

  it('puts favorites first and keeps catalog order inside each group', () => {
    const { favorites, others } = groupLines(lines, [9, 1]);

    expect(favorites.map((l) => l.code)).toEqual([1, 9]);
    expect(others.map((l) => l.code)).toEqual([7, 15]);
  });

  it('returns every line in others when there are no favorites', () => {
    const { favorites, others } = groupLines(lines, []);

    expect(favorites).toEqual([]);
    expect(others).toHaveLength(4);
  });

  it('ignores favorites that are not in the list', () => {
    const { favorites } = groupLines(lines, [99]);

    expect(favorites).toEqual([]);
  });
});

describe('isNight', () => {
  it('is true when every line is encerrada', () => {
    expect(isNight([line(1, 'encerrada'), line(2, 'encerrada')])).toBe(true);
  });

  it('is false when any line is operating', () => {
    expect(isNight([line(1, 'encerrada'), line(2, 'normal')])).toBe(false);
  });

  it('ignores lines without data when deciding', () => {
    expect(isNight([line(1, 'encerrada'), line(13, 'desconhecido')])).toBe(true);
  });

  it('is false for an empty list', () => {
    expect(isNight([])).toBe(false);
  });
});

describe('formatClock', () => {
  it('formats local time as H:MM, like the Android status bar', () => {
    expect(formatClock(new Date(2026, 9, 7, 22, 21))).toBe('22:21');
    expect(formatClock(new Date(2026, 9, 7, 2, 4))).toBe('2:04');
  });
});
