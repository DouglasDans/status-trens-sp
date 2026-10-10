import cptm from '../assets/images/operators/cptm.svg';
import metro from '../assets/images/operators/metro.svg';
import motiva from '../assets/images/operators/motiva.svg';
import viamobilidade from '../assets/images/operators/viamob.svg';

// Logos by operator id (ids come from the backend catalog). An operator
// without a logo here is shown by name; adding one is just a new entry.
export const operatorLogos: Record<string, number> = {
  cptm,
  metro,
  motiva,
  viamobilidade,
};

// Logos that keep their own colors in dark mode; every other logo is tinted
// fully white so it stays legible on the dark surface.
const keepColorsInDark = new Set(['metro']);

export function logoTint(operator: string, scheme: 'light' | 'dark'): string | null {
  return scheme === 'dark' && !keepColorsInDark.has(operator) ? '#FFFFFF' : null;
}
