import { describe, expect, it } from 'vitest';
import { contrastRatio, onColor, toneSetFrom } from './on-color.util';

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
