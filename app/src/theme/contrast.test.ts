import { badgeInk, contrastRatio, lineInk, needsOutline } from './contrast';

describe('contrastRatio', () => {
  it('is 21 between black and white', () => {
    expect(contrastRatio('#000000', '#FFFFFF')).toBeCloseTo(21, 0);
  });

  it('is 1 for the same color', () => {
    expect(contrastRatio('#219896', '#219896')).toBeCloseTo(1, 5);
  });
});

describe('badgeInk', () => {
  it('uses white text on dark line colors', () => {
    expect(badgeInk('#00378c')).toBe('#FFFFFF'); // Linha 1 Azul
    expect(badgeInk('#1f2086')).toBe('#FFFFFF'); // Linha 12 Safira
  });

  it('uses dark text on light line colors', () => {
    expect(badgeInk('#efba00')).toBe('#151515'); // Linha 4 Amarela
  });
});

describe('needsOutline', () => {
  it('outlines a badge that blends into the surface', () => {
    expect(needsOutline('#848d90', '#1C1D1F')).toBe(false);
    expect(needsOutline('#1f2086', '#1C1D1F')).toBe(true); // Safira on dark surface
  });

  it('does not outline a badge with enough contrast', () => {
    expect(needsOutline('#00378c', '#FFFFFF')).toBe(false);
  });
});

describe('lineInk', () => {
  it('uses the official text color when the backend sends one', () => {
    expect(lineInk('#00a88e', '#ffffff')).toBe('#ffffff'); // Esmeralda: white by the signs, not by contrast
  });

  it('falls back to contrast when the line has no official text color', () => {
    expect(lineInk('#efba00', '')).toBe('#151515');
  });
});
