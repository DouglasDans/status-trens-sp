import cptm from '../assets/images/operators/cptm.svg';
import metroDark from '../assets/images/operators/metro-dark.svg';
import metro from '../assets/images/operators/metro.svg';
import motiva from '../assets/images/operators/motiva.svg';
import viamobilidade from '../assets/images/operators/viamob.svg';

type OperatorLogo = { light: number; dark?: number };

// Logos by operator id (ids come from the backend catalog). An operator
// without a logo here is shown by name; adding one is just a new entry.
// `dark` is an optional variant drawn for dark backgrounds.
const operatorLogos: Record<string, OperatorLogo> = {
  cptm: { light: cptm },
  metro: { light: metro, dark: metroDark }, // dark: official "traço grosso" signature
  motiva: { light: motiva },
  viamobilidade: { light: viamobilidade },
};

/**
 * Logo to show for an operator. In dark mode an operator with its own dark
 * variant uses it; any other logo is tinted fully white to stay legible.
 */
export function logoFor(operator: string, scheme: 'light' | 'dark'): { source: number; tint: string | null } | null {
  const logo = operatorLogos[operator];
  if (!logo) return null;
  if (scheme === 'light') return { source: logo.light, tint: null };
  if (logo.dark) return { source: logo.dark, tint: null };
  return { source: logo.light, tint: '#FFFFFF' };
}
