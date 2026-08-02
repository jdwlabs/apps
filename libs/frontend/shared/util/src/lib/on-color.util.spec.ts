import { describe, expect, it } from 'vitest';
import { Hct, argbFromHex } from '@material/material-color-utilities';
import {
  complementOf,
  contrastRatio,
  onColor,
  toneSetFrom,
} from './on-color.util';

// Sampled across the hue circle plus the two cases the design measured:
// the legacy user-custom green, and a mid-grey, which is the worst case for
// picking a foreground because it is nearly equidistant from both endpoints.
const SAMPLES = [
  '#7dcf2a',
  '#777777',
  '#2f5fa8',
  '#a8622f',
  '#ffffff',
  '#000000',
  '#ffb4ab',
  '#123456',
];

describe('contrastRatio', () => {
  it('is 21 for black on white', () => {
    expect(contrastRatio('#000000', '#ffffff')).toBeCloseTo(21, 2);
  });

  it('is 1 for a colour against itself', () => {
    expect(contrastRatio('#7dcf2a', '#7dcf2a')).toBeCloseTo(1, 5);
  });

  it('is symmetric', () => {
    expect(contrastRatio('#2f5fa8', '#ffffff')).toBeCloseTo(
      contrastRatio('#ffffff', '#2f5fa8'),
      5,
    );
  });
});

describe('onColor', () => {
  it.each(SAMPLES)('clears the text floor on %s', (background) => {
    expect(
      contrastRatio(onColor(background), background),
    ).toBeGreaterThanOrEqual(4.5);
  });

  it.each(SAMPLES)('clears a non-text floor on %s', (background) => {
    expect(
      contrastRatio(onColor(background, 3), background),
    ).toBeGreaterThanOrEqual(3);
  });
});

describe('complementOf', () => {
  const hctOf = (hex: string) => Hct.fromInt(argbFromHex(hex));
  const hueGap = (a: number, b: number) => {
    const raw = Math.abs(a - b) % 360;
    return raw > 180 ? 360 - raw : raw;
  };

  // A byte rotation of the RGB channels looks like a complement and is not: it
  // moves a saturated hue about 120 degrees and leaves any grey exactly where
  // it started. Measured in HCT so the assertion is about hue rather than about
  // whichever sRGB triple happens to encode it.
  it.each(['#7dcf2a', '#2f5fa8', '#a8622f', '#c2185b', '#00897b'])(
    'sits opposite %s on the hue circle',
    (seed) => {
      const source = hctOf(seed);
      const rotated = hctOf(complementOf(seed));
      expect(hueGap(source.hue, rotated.hue)).toBeCloseTo(180, 0);
    },
  );

  it.each(['#7dcf2a', '#2f5fa8', '#c2185b'])(
    'holds the tone of %s so the pairing stays usable at the same lightness',
    (seed) => {
      expect(hctOf(complementOf(seed)).tone).toBeCloseTo(hctOf(seed).tone, 0);
    },
  );

  // Documented property, not an oversight: a colour with no chroma has no hue
  // to rotate, so it is its own complement. A user who picks grey gets a grey
  // theme, which is the honest answer rather than a hue invented for them.
  it.each(['#ffffff', '#000000'])('returns %s unchanged', (grey) => {
    expect(complementOf(grey)).toBe(grey);
  });

  it('leaves a mid grey visually where it was', () => {
    const rotated = complementOf('#808080');
    expect(hctOf(rotated).chroma).toBeLessThan(3);
    expect(contrastRatio(rotated, '#808080')).toBeCloseTo(1, 1);
  });
});

describe('toneSetFrom', () => {
  // The defect this exists to prevent: the user-custom themes aliased
  // primary-container to primary, which flattened tone 90/30 onto a saturated
  // tone and made on-primary-container meaningless.
  it.each(SAMPLES)('keeps container distinct from base for %s', (seed) => {
    const light = toneSetFrom(seed, 'light');
    expect(light.container).not.toBe(light.base);
  });

  it.each(SAMPLES)(
    'pairs every role with a legible foreground for %s',
    (seed) => {
      for (const type of ['light', 'dark'] as const) {
        const set = toneSetFrom(seed, type);
        expect(contrastRatio(set.on, set.base)).toBeGreaterThanOrEqual(4.5);
        expect(
          contrastRatio(set.onContainer, set.container),
        ).toBeGreaterThanOrEqual(4.5);
      }
    },
  );

  it('puts the light container above the light base in luminance', () => {
    const set = toneSetFrom('#2f5fa8', 'light');
    expect(contrastRatio(set.container, '#000000')).toBeGreaterThan(
      contrastRatio(set.base, '#000000'),
    );
  });
});
