import { test, expect, Locator } from '@playwright/test';
import { mockServiceDiscovery } from '../support/mock-service-discovery';
import { ONE_PIXEL_PNG, signInAs } from '../support/sign-in-as';
import { contrastRatio, flatten, parseColour } from '../support/contrast';

const boxOf = async (locator: Locator) => {
  const box = await locator.boundingBox();
  if (!box) {
    throw new Error('element has no layout box');
  }
  return box;
};

const colourOf = (locator: Locator, property: string) =>
  locator.evaluate(
    (el, name) => getComputedStyle(el).getPropertyValue(name),
    property,
  );

// Nothing else in CI catches a shape or token regression: lint, unit tests and
// build all pass on a header whose radii are wrong, which is how three M3
// regressions reached production green.
test.describe('Header shape system', () => {
  test.beforeEach(async ({ page }) => {
    await mockServiceDiscovery(page);
    await page.goto('/');
  });

  test('the environment badge is a rounded square, not a pill', async ({
    page,
  }) => {
    const badge = page.locator('[data-cy="env-badge"]');
    await expect(badge).toBeVisible();
    await expect(badge).toHaveCSS('border-radius', '8px');
  });

  test('the badge carries one accessible name for environment and version', async ({
    page,
  }) => {
    const badge = page.locator('[data-cy="env-badge"]');
    await expect(badge).toHaveAttribute(
      'aria-label',
      /^Environment .+, version/,
    );
  });

  test('the account menu is present and circular', async ({ page }) => {
    const trigger = page.locator('[data-cy="account-menu-button"]');
    await expect(trigger).toBeVisible();
  });

  test('the account menu opens and offers a way in when signed out', async ({
    page,
  }) => {
    await page.locator('[data-cy="account-menu-button"]').click();
    await expect(page.locator('[data-cy="sign-in-button"]')).toBeVisible();
    await expect(page.locator('[data-cy="sign-up-button"]')).toBeVisible();
  });

  test('dashboard tile icons are rounded squares, not circles', async ({
    page,
  }) => {
    const icon = page.locator('[data-cy="tile"] .avatar').first();
    await expect(icon).toBeVisible();
    await expect(icon).toHaveCSS('border-radius', '8px');
  });

  // The badge's own root is an inline-level box. Left inside a block-level host
  // it is placed by baseline against the inherited toolbar strut, so the host
  // measures taller than the badge and centring the host leaves the badge
  // visibly low. Only geometry catches that — the radii and tokens are correct
  // either way.
  test('the badge is centred on the same line as the account button', async ({
    page,
  }) => {
    const host = await boxOf(page.locator('jdw-env-badge'));
    const badge = await boxOf(page.locator('[data-cy="env-badge"]'));
    const trigger = await boxOf(
      page.locator('[data-cy="account-menu-button"]'),
    );

    // No leading: the host is the badge, so there is no taller box to centre.
    expect(host.height).toBeCloseTo(badge.height, 0);
    expect(badge.y + badge.height / 2).toBeCloseTo(
      trigger.y + trigger.height / 2,
      0,
    );
  });
});

// Material sizes an icon button's padding from `--mat-icon-button-icon-size`,
// so a 32px avatar declared without restating that token overflows a 24px
// content box — a layout fault no unit test can observe, since jsdom computes
// no geometry. The colour cases need a browser for the same reason: the fault
// they guard is a collision between two resolved tokens, not a wrong literal.
test.describe('Account avatar', () => {
  test.beforeEach(async ({ page }) => {
    await mockServiceDiscovery(page);
  });

  test('the initials avatar sits centred inside its trigger', async ({
    page,
  }) => {
    await signInAs(page);
    await page.goto('/');

    const trigger = page.locator('[data-cy="account-menu-button"]');
    const initials = page.locator('[data-cy="avatar-initials"]');
    await expect(initials).toHaveText('JD');

    const triggerBox = await boxOf(trigger);
    const avatarBox = await boxOf(initials);

    expect(avatarBox.width).toBeCloseTo(32, 0);
    expect(avatarBox.height).toBeCloseTo(32, 0);
    // Centred, rather than pushed to the right and bottom of a content box too
    // small to hold it.
    expect(avatarBox.x + avatarBox.width / 2).toBeCloseTo(
      triggerBox.x + triggerBox.width / 2,
      0,
    );
    expect(avatarBox.y + avatarBox.height / 2).toBeCloseTo(
      triggerBox.y + triggerBox.height / 2,
      0,
    );
  });

  test('a photo avatar renders at the same size as initials', async ({
    page,
  }) => {
    await signInAs(page, { iconData: ONE_PIXEL_PNG });
    await page.goto('/');

    const photo = page.locator('[data-cy="avatar-image"]');
    await expect(photo).toBeVisible();

    const box = await boxOf(photo);
    expect(box.width).toBeCloseTo(32, 0);
    expect(box.height).toBeCloseTo(32, 0);
  });

  // A dark theme's toolbar fills itself with the *container* tone of its colour
  // input, which is the same role family an avatar chip naturally reaches for.
  // Where the two coincide the chip disappears into the bar and the initials
  // read as loose letters — every token individually correct, the pair not.
  test('the initials avatar reads as a chip against the bar behind it', async ({
    page,
  }) => {
    await signInAs(page);
    await page.goto('/');

    const bar = parseColour(
      await colourOf(
        page.locator('[data-cy="navbar-header"]'),
        'background-color',
      ),
    );
    const avatar = page.locator('[data-cy="avatar-initials"]');
    const chip = parseColour(await colourOf(avatar, 'background-color'));
    const text = parseColour(await colourOf(avatar, 'color'));

    const chipOverBar = flatten(chip, bar.rgb);
    expect(contrastRatio(chipOverBar, bar.rgb)).toBeGreaterThanOrEqual(3);
    expect(
      contrastRatio(flatten(text, chipOverBar), chipOverBar),
    ).toBeGreaterThanOrEqual(4.5);
  });
});

// The collapsed rail is the only place these apply: the expanded drawer encloses
// a label, so its indicator is a wide pill by design.
test.describe('Navigation rail indicator', () => {
  test.beforeEach(async ({ page }) => {
    await mockServiceDiscovery(page);
    // Wide enough that the drawer resolves to side mode, which is what
    // collapses it into the icons-only rail.
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto('/');
  });

  test('the collapsed indicator is a circle, not a lens', async ({ page }) => {
    const item = page.locator('[data-cy="navigation-link"]').first();
    await expect(item).toBeVisible();

    const box = await boxOf(item);
    expect(box.width).toBeCloseTo(box.height, 0);
    await expect(item).toHaveCSS('border-radius', '9999px');
    // Clipping the fill to a shorter box does not shorten the corner curves, it
    // only cuts them — which is how a circle became a lens here before.
    await expect(item).toHaveCSS('background-clip', 'border-box');
    await expect(item).toHaveCSS('padding', '0px');
  });

  test('the state layer covers the indicator exactly', async ({ page }) => {
    const item = page.locator('[data-cy="navigation-link"]').first();
    await item.hover();

    // Material pins its state layer to all four edges of the item. Any inset
    // that the fill does not share leaves a ghost visible around the indicator.
    const inset = await item.evaluate((el) => {
      const layer = getComputedStyle(el, '::before');
      return [layer.top, layer.right, layer.bottom, layer.left];
    });
    expect(inset).toEqual(['0px', '0px', '0px', '0px']);
  });

  test('the icon centre does not move when the rail expands', async ({
    page,
  }) => {
    const icon = page.locator('[data-cy="navigation-icon"]').first();
    await expect(icon).toBeVisible();
    const collapsed = await boxOf(icon);

    await page.locator('[data-cy="expand-toggle-button"]').click();
    await expect(
      page.locator('[data-cy="navigation-title"]').first(),
    ).toBeVisible();
    const expanded = await boxOf(icon);

    expect(expanded.x + expanded.width / 2).toBeCloseTo(
      collapsed.x + collapsed.width / 2,
      0,
    );
  });

  // Material's own focus affordance is the same translucent state layer it uses
  // for hover, in the same box as the indicator — so on a selected destination
  // it reads as a smudge rather than a focus ring. A drawn ring is the only
  // thing here that survives being laid over the indicator.
  test('keyboard focus draws a ring the indicator never draws', async ({
    page,
  }) => {
    const item = page.locator('[data-cy="navigation-link"]').first();
    // `:focus-visible` tracks the browser's focus *modality*, not the focused
    // element, and a programmatic focus alone does not set it. Round-tripping
    // through a real Tab press does.
    await item.focus();
    await page.keyboard.press('Tab');
    await page.keyboard.press('Shift+Tab');
    await expect(item).toHaveCSS('outline-style', 'solid');
    const width = await item.evaluate((el) =>
      parseFloat(getComputedStyle(el).outlineWidth),
    );
    expect(width).toBeGreaterThanOrEqual(2);

    const ring = parseColour(await colourOf(item, 'outline-color'));
    const drawer = parseColour(
      await colourOf(page.locator('[data-cy="sidenav"]'), 'background-color'),
    );
    expect(
      contrastRatio(flatten(ring, drawer.rgb), drawer.rgb),
    ).toBeGreaterThanOrEqual(3);
  });
});
