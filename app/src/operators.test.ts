import { logoFor } from './operators';

describe('logoFor', () => {
  it('uses the original logo, untinted, in light mode', () => {
    expect(logoFor('cptm', 'light')).toEqual({ source: expect.anything(), tint: null });
    expect(logoFor('metro', 'light')?.tint).toBeNull();
  });

  it('turns logos without a dark variant fully white in dark mode', () => {
    expect(logoFor('cptm', 'dark')?.tint).toBe('#FFFFFF');
    expect(logoFor('motiva', 'dark')?.tint).toBe('#FFFFFF');
  });

  it('uses the dark variant, untinted, when the operator has one', () => {
    const light = logoFor('metro', 'light');
    const dark = logoFor('metro', 'dark');

    expect(dark?.tint).toBeNull();
    expect(dark?.source).not.toBe(light?.source);
  });

  it('returns nothing for an operator without logo', () => {
    expect(logoFor('nova', 'dark')).toBeNull();
  });
});
