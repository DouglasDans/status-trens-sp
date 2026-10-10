import { logoTint } from './operators';

describe('logoTint', () => {
  it('turns logos fully white in dark mode', () => {
    expect(logoTint('cptm', 'dark')).toBe('#FFFFFF');
    expect(logoTint('motiva', 'dark')).toBe('#FFFFFF');
    expect(logoTint('viamobilidade', 'dark')).toBe('#FFFFFF');
  });

  it('keeps the Metrô logo in its own colors in dark mode', () => {
    expect(logoTint('metro', 'dark')).toBeNull();
  });

  it('keeps every logo in its own colors in light mode', () => {
    expect(logoTint('cptm', 'light')).toBeNull();
    expect(logoTint('metro', 'light')).toBeNull();
  });
});
