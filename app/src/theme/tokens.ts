// Visual tokens from the Claude Design reference ("Status Trens SP v2").
import type { State } from '@/api/status';

export type Tone = 'green' | 'yellow' | 'red' | 'gray';

const light = {
  bg: '#F4F4F1',
  surface: '#FFFFFF',
  text: '#151515',
  muted: '#5C5C58',
  border: '#ECEBE6',
  dash: '#CFCEC8',
  skel: '#E9E8E3',
  banner: '#E9E9E4',
  outline: 'rgba(0,0,0,.28)',
  grayBadge: '#8E8E8A',
  tones: {
    green: { fill: '#1F8A47', glyph: '#FFFFFF', label: '#1A7A3E' },
    yellow: { fill: '#E5AE00', glyph: '#151515', label: '#7A5400' },
    red: { fill: '#D0342C', glyph: '#FFFFFF', label: '#B3241D' },
    gray: { fill: '#8A8A86', glyph: '#FFFFFF', label: '#5F5F5B' },
  },
};

export type Theme = typeof light;

const dark: Theme = {
  bg: '#111213',
  surface: '#1C1D1F',
  text: '#EDEDEA',
  muted: '#A3A39E',
  border: '#2A2B2E',
  dash: '#45464A',
  skel: '#27282B',
  banner: '#26272A',
  outline: 'rgba(255,255,255,.45)',
  grayBadge: '#4D4E52',
  tones: {
    green: { fill: '#4CC27A', glyph: '#0F1A13', label: '#5ACB85' },
    yellow: { fill: '#F2C440', glyph: '#151515', label: '#F2C440' },
    red: { fill: '#FF6B60', glyph: '#1A0D0C', label: '#FF8077' },
    gray: { fill: '#8F8F8B', glyph: '#151515', label: '#A3A39E' },
  },
};

export const themes = { light, dark };

export const fonts = {
  regular: 'IBMPlexSans_400Regular',
  medium: 'IBMPlexSans_500Medium',
  semibold: 'IBMPlexSans_600SemiBold',
  bold: 'IBMPlexSans_700Bold',
  mono: 'IBMPlexMono_500Medium',
  monoSemibold: 'IBMPlexMono_600SemiBold',
};

// 16x16 stroke paths from the design.
export const icons = {
  check: 'M4 8.5l2.6 2.5L12 5.5',
  bang: 'M8 4.2v4.6M8 11.6v.2',
  split: 'M3.5 8h3M9.5 8h3',
  x: 'M5.2 5.2l5.6 5.6M10.8 5.2l-5.6 5.6',
  moon: 'M10.8 4.6a3.9 3.9 0 1 0 .8 5.6 3.2 3.2 0 0 1-.8-5.6z',
  q: 'M6 6.3a2 2 0 1 1 2.8 1.8c-.6.3-.8.7-.8 1.3M8 11.6v.2',
  info: 'M8 7.4v4M8 4.6v.2',
};

export const star = 'M12 3.5l2.6 5.3 5.8.8-4.2 4.1 1 5.8L12 16.8l-5.2 2.7 1-5.8-4.2-4.1 5.8-.8z';

// Color comes with an icon and a label: status never depends on color alone.
export const stateVisual: Record<State, { tone: Tone; icon: string; label: string }> = {
  normal: { tone: 'green', icon: icons.check, label: 'Operação Normal' },
  reduzida: { tone: 'yellow', icon: icons.bang, label: 'Velocidade Reduzida' },
  parcial: { tone: 'yellow', icon: icons.split, label: 'Operação Parcial' },
  paralisada: { tone: 'red', icon: icons.x, label: 'Paralisada' },
  encerrada: { tone: 'gray', icon: icons.moon, label: 'Operação Encerrada' },
  desconhecido: { tone: 'gray', icon: icons.q, label: 'Status desconhecido' },
};
