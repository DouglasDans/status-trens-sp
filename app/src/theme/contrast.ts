// WCAG contrast helpers, used to keep official line colors legible in both themes.

const INK_LIGHT = '#FFFFFF';
const INK_DARK = '#151515';

function luminance(hex: string) {
  const [r, g, b] = [1, 3, 5].map((i) => {
    const v = parseInt(hex.slice(i, i + 2), 16) / 255;
    return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

export function contrastRatio(a: string, b: string) {
  const [x, y] = [luminance(a), luminance(b)];
  return (Math.max(x, y) + 0.05) / (Math.min(x, y) + 0.05);
}

/** Text color for the line number on top of the line color. */
export function badgeInk(background: string) {
  return contrastRatio(background, INK_LIGHT) >= contrastRatio(background, INK_DARK) ? INK_LIGHT : INK_DARK;
}

/** A badge that blends into the surface gets an outline (e.g. Safira on dark). */
export function needsOutline(badge: string, surface: string) {
  return contrastRatio(badge, surface) < 3;
}
