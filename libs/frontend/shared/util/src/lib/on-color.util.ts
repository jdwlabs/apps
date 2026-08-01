import {
  Hct,
  argbFromHex,
  hexFromArgb,
} from '@material/material-color-utilities';

export type ThemeType = 'light' | 'dark';

export interface ToneSet {
  readonly base: string;
  readonly on: string;
  readonly container: string;
  readonly onContainer: string;
}

// M3's tone assignments. A container sits close to the surface it fills by
// design and relies on hue for separation, which is why a seed with no hue
// cannot produce a usable one.
const BASE_TONE: Record<ThemeType, number> = { light: 40, dark: 80 };
const CONTAINER_TONE: Record<ThemeType, number> = { light: 90, dark: 30 };

const srgbChannel = (value: number): number => {
  const scaled = value / 255;
  return scaled <= 0.04045
    ? scaled / 12.92
    : Math.pow((scaled + 0.055) / 1.055, 2.4);
};

const relativeLuminance = (hex: string): number => {
  const argb = argbFromHex(hex);
  const r = (argb >> 16) & 0xff;
  const g = (argb >> 8) & 0xff;
  const b = argb & 0xff;
  return (
    0.2126 * srgbChannel(r) + 0.7152 * srgbChannel(g) + 0.0722 * srgbChannel(b)
  );
};

const atTone = (source: Hct, tone: number): string =>
  hexFromArgb(Hct.from(source.hue, source.chroma, tone).toInt());

export function contrastRatio(a: string, b: string): number {
  const first = relativeLuminance(a);
  const second = relativeLuminance(b);
  const lighter = Math.max(first, second);
  const darker = Math.min(first, second);
  return (lighter + 0.05) / (darker + 0.05);
}

// Prefers M3's tonal foreground so the pairing keeps the background's hue,
// and falls back to a pure endpoint only when the tonal one misses the floor.
// One endpoint always clears 4.5:1 in sRGB — the worst case is a mid grey,
// where black still reaches 4.69:1 — so this cannot fail to return a legible
// value for the floors this design system uses.
export function onColor(background: string, floor = 4.5): string {
  const source = Hct.fromInt(argbFromHex(background));
  const tonal = source.tone > 60 ? atTone(source, 10) : atTone(source, 100);
  if (contrastRatio(tonal, background) >= floor) {
    return tonal;
  }
  return contrastRatio('#ffffff', background) >=
    contrastRatio('#000000', background)
    ? '#ffffff'
    : '#000000';
}

export function toneSetFrom(seed: string, type: ThemeType): ToneSet {
  const source = Hct.fromInt(argbFromHex(seed));
  const base = atTone(source, BASE_TONE[type]);
  const container = atTone(source, CONTAINER_TONE[type]);
  return {
    base,
    on: onColor(base),
    container,
    onContainer: onColor(container),
  };
}
