// WCAG 2.1 relative luminance and contrast ratio. Colours arrive from
// `getComputedStyle`, so the parser has to accept whatever the browser chose to
// serialise — `rgb()`, `rgba()` and the `color(srgb ...)` form Chromium emits
// for anything that went through `color-mix`.
export type Rgb = [number, number, number];

export type Colour = { rgb: Rgb; alpha: number };

export function parseColour(value: string): Colour {
  // A custom property keeps whatever notation it was authored in — the theme
  // seeds are hex — while `color` and `background-color` are always serialised
  // as a colour function. Comparing a resolved role against a computed colour
  // means both notations arrive at this parser.
  const hex = value.trim().match(/^#([0-9a-f]{3}|[0-9a-f]{6})$/i);
  if (hex) {
    const digits =
      hex[1].length === 3
        ? hex[1].replace(/./g, (digit) => digit + digit)
        : hex[1];
    return {
      rgb: [
        parseInt(digits.slice(0, 2), 16),
        parseInt(digits.slice(2, 4), 16),
        parseInt(digits.slice(4, 6), 16),
      ],
      alpha: 1,
    };
  }
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

/**
 * Gap between the strongest and weakest channel — zero for any grey, and the
 * only property some of these assertions need. Contrast alone cannot express
 * "this segment carries a hue": in a palette whose families are one grey ramp,
 * two roles can be equally legible and still be the same colour.
 */
export function channelSpread(rgb: Rgb): number {
  return Math.max(...rgb) - Math.min(...rgb);
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
