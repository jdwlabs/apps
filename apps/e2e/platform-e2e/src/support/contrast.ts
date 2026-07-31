// WCAG 2.1 relative luminance and contrast ratio. Colours arrive from
// `getComputedStyle`, so the parser has to accept whatever the browser chose to
// serialise — `rgb()`, `rgba()` and the `color(srgb ...)` form Chromium emits
// for anything that went through `color-mix`.
type Rgb = [number, number, number];

export type Colour = { rgb: Rgb; alpha: number };

export function parseColour(value: string): Colour {
  const srgb = value.match(/^color\(srgb\s+([^)]+)\)$/);
  if (srgb) {
    const parts = srgb[1]
      .split(/[\s/]+/)
      .filter(Boolean)
      .map(Number);
    return {
      rgb: [parts[0] * 255, parts[1] * 255, parts[2] * 255],
      alpha: parts[3] ?? 1,
    };
  }
  const rgb = value.match(/^rgba?\(([^)]+)\)$/);
  if (!rgb) {
    throw new Error(`unrecognised colour: ${value}`);
  }
  const parts = rgb[1]
    .split(/[,\s/]+/)
    .filter(Boolean)
    .map(Number);
  return { rgb: [parts[0], parts[1], parts[2]], alpha: parts[3] ?? 1 };
}

/** Composites a possibly translucent colour over an opaque one. */
export function flatten(colour: Colour, backdrop: Rgb): Rgb {
  return colour.rgb.map(
    (channel, i) => channel * colour.alpha + backdrop[i] * (1 - colour.alpha),
  ) as Rgb;
}

export function contrastRatio(a: Rgb, b: Rgb): number {
  const [lighter, darker] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (lighter + 0.05) / (darker + 0.05);
}

function luminance([r, g, b]: Rgb): number {
  return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b);
}

function channel(value: number): number {
  const c = value / 255;
  return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
}
