import type { LineStatus } from '@/api/status';

/** Splits lines into favorites and the rest, keeping the backend (catalog) order. */
export function groupLines(lines: LineStatus[], favorites: number[]) {
  const set = new Set(favorites);
  return {
    favorites: lines.filter((l) => set.has(l.code)),
    others: lines.filter((l) => !set.has(l.code)),
  };
}

/** Late night: every line with data is closed. Lines without data don't count. */
export function isNight(lines: LineStatus[]) {
  const known = lines.filter((l) => l.status !== 'desconhecido');
  return known.length > 0 && known.every((l) => l.status === 'encerrada');
}

export function formatClock(date: Date) {
  return `${date.getHours()}:${String(date.getMinutes()).padStart(2, '0')}`;
}
