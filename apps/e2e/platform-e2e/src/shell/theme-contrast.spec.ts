import { test, expect, Page } from '@playwright/test';
import { mockServiceDiscovery } from '../support/mock-service-discovery';
import { contrastRatio, flatten, parseColour } from '../support/contrast';

const THEMES = ['blue-slate', 'user-custom-light', 'user-custom-dark'];

// Six colours spanning the hue circle. A user can pick any of them, so each
// has to clear the floors — the design's original failure was a set of chips
// that passed only because one particular default green happened to sit where
// it did.
const USER_COLOURS = [
  '#7dcf2a',
  '#2f5fa8',
  '#a8622f',
  '#c2185b',
  '#00897b',
  '#5d4037',
];

// Storage is the only channel that survives the reload, so this exercises the
// same restore path a returning user gets rather than a shortcut past it. That
// matters: writing `data-theme` directly would have kept passing while nothing
// injected the service that reads the stored choice at boot.
const applyTheme = async (page: Page, theme: string, colour?: string) => {
  await page.evaluate(
    ([id, hex]) => {
      if (hex) {
        localStorage.setItem('jdw.theme.colour', hex);
      }
      localStorage.setItem('jdw.theme', id);
    },
    [theme, colour ?? ''],
  );
  await page.reload();
  await expect(page.locator('[data-cy="env-badge"]')).toBeVisible();
  await expect(page.locator('html')).toHaveAttribute('data-theme', theme);

  // The seed reaches the page as published custom properties, so this is the
  // only observable proof it was read back at all. A customisable theme that
  // published nothing would leave every role on its compiled fallback and still
  // render a perfectly legible page.
  const published = await page.evaluate(() =>
    document.documentElement.style.getPropertyValue('--primary-500'),
  );
  if (colour) {
    expect(published).toMatch(/^#[0-9a-f]{6}$/i);
  }
  return published;
};

const read = (page: Page, selector: string, property: string) =>
  page
    .locator(selector)
    .first()
    .evaluate(
      (el, name) => getComputedStyle(el).getPropertyValue(name),
      property,
    );

/**
 * Contrast between two computed colours, compositing any alpha over the one
 * behind it.
 *
 * Both operands name their own property because the pairing that means
 * anything is the one where the two colours actually meet. The badge halves
 * each paint an opaque fill, so a chip's label never touches the toolbar: in
 * the dark theme the version fill sits 1.31:1 from the bar while its label sits
 * 7.23:1 from that same bar. Measuring a label against a background two layers
 * back moves independently of what a reader sees, so it can neither pass nor
 * fail for the right reason.
 */
const ratio = async (
  page: Page,
  foreground: [string, string],
  background: [string, string],
) => {
  const back = parseColour(await read(page, ...background));
  const front = parseColour(await read(page, ...foreground));
  return contrastRatio(flatten(front, back.rgb), back.rgb);
};

const ENV_FILL: [string, string] = [
  '[data-cy="env-segment"]',
  'background-color',
];
const TOOLBAR_FILL: [string, string] = [
  '[data-cy="navbar-header"]',
  'background-color',
];

test.describe('Theme contrast floors', () => {
  test.beforeEach(async ({ page }) => {
    await mockServiceDiscovery(page);
    await page.goto('/');
  });

  for (const theme of THEMES) {
    // The chip is a non-text component under WCAG 2.1 SC 1.4.11, so what has to
    // clear 3:1 is its fill against the bar it sits on — that is the edge a
    // reader picks the chip out by.
    test(`${theme}: the environment segment clears the non-text floor`, async ({
      page,
    }) => {
      await applyTheme(page, theme);
      expect(await ratio(page, ENV_FILL, TOOLBAR_FILL)).toBeGreaterThanOrEqual(
        3,
      );
    });

    test(`${theme}: the version segment clears the text floor`, async ({
      page,
    }) => {
      await applyTheme(page, theme);
      expect(
        await ratio(
          page,
          ['[data-cy="version-segment"]', 'color'],
          ['[data-cy="version-segment"]', 'background-color'],
        ),
      ).toBeGreaterThanOrEqual(4.5);
    });

    test(`${theme}: the active destination separates from the drawer`, async ({
      page,
    }) => {
      await applyTheme(page, theme);
      const item = page.locator('[data-cy="navigation-link"]').first();
      await item.click();
      await expect(item).toHaveClass(/(^|\s)active(\s|$)/);
      // The click leaves the pointer on the item, and `.active:hover` blends a
      // state layer into the fill this measures.
      await page.mouse.move(600, 700);

      const indicator = parseColour(
        await item.evaluate((el) => getComputedStyle(el).backgroundColor),
      );
      const drawer = parseColour(
        await page
          .locator('mat-sidenav')
          .first()
          .evaluate((el) => getComputedStyle(el).backgroundColor),
      );
      expect(
        contrastRatio(flatten(indicator, drawer.rgb), drawer.rgb),
      ).toBeGreaterThanOrEqual(3);
    });
  }

  // Six seeds are only a matrix if they produce six themes. Should the stored
  // colour stop reaching the service — a renamed key, a rejected value, a
  // publish that no longer writes — every case below silently collapses into
  // six readings of the default seed and stays green. The seed moves the chip
  // pairing by about 1%, far too little for anyone to notice that by eye, so
  // the distinctness has to be asserted rather than inferred.
  test('each user colour publishes a palette of its own', async ({ page }) => {
    const published = new Set<string>();
    for (const colour of USER_COLOURS) {
      published.add(await applyTheme(page, 'user-custom-light', colour));
    }
    expect(published.size).toBe(USER_COLOURS.length);
  });

  // The customisable themes recolour the toolbar from the user's seed while the
  // environment chips keep the fixed roles that carry their meaning, so every
  // seed is a fresh pairing rather than a variation on one.
  for (const colour of USER_COLOURS) {
    test(`a user colour of ${colour} keeps the environment chip legible`, async ({
      page,
    }) => {
      await applyTheme(page, 'user-custom-light', colour);
      expect(await ratio(page, ENV_FILL, TOOLBAR_FILL)).toBeGreaterThanOrEqual(
        3,
      );
    });
  }
});
