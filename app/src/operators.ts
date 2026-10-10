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
