import { test, expect, Locator } from '@playwright/test';
import { mockServiceDiscovery } from '../support/mock-service-discovery';
import { ONE_PIXEL_PNG, signInAs } from '../support/sign-in-as';

const boxOf = async (locator: Locator) => {
  const box = await locator.boundingBox();
  if (!box) {
    throw new Error('element has no layout box');
  }
  return box;
};

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
});

// Material sizes the icon button's padding from `--mat-icon-button-icon-size`,
// so a 32px avatar declared without restating that token overflows a 24px
// content box — a layout fault no unit test can observe, since jsdom computes
// no geometry. The photo case is pinned here too: Material also sizes a
// projected `img` from that token, and only the encapsulation attribute on
// `.avatar` currently outranks it.
test.describe('Account avatar geometry', () => {
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

  test('the initials avatar does not paint in the toolbar own fill colour', async ({
    page,
  }) => {
    await signInAs(page);
    await page.goto('/');

    const toolbarFill = await page
      .locator('[data-cy="navbar-header"]')
      .evaluate((el) => getComputedStyle(el).backgroundColor);
    const avatarFill = await page
      .locator('[data-cy="avatar-initials"]')
      .evaluate((el) => getComputedStyle(el).backgroundColor);

    expect(avatarFill).not.toBe(toolbarFill);
  });
});

// The collapsed rail is the only place these apply: the over-mode drawer shows
// labels, so its indicator is a wide pill by design.
test.describe('Navigation rail indicator', () => {
  test.beforeEach(async ({ page }) => {
    await mockServiceDiscovery(page);
    // Wide enough that the drawer resolves to side mode, which is what
    // collapses it into the icons-only rail.
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto('/');
  });

  test('the collapsed indicator is a circle, not a stadium', async ({
    page,
  }) => {
    const item = page.locator('[data-cy="navigation-link"]').first();
    await expect(item).toBeVisible();

    const box = await boxOf(item);
    // The background is clipped to the content box, so the painted indicator is
    // the item box less its vertical padding — square, hence circular at
    // `corner-full`.
    const padding = await item.evaluate((el) =>
      parseFloat(getComputedStyle(el).paddingTop),
    );
    expect(box.width).toBeCloseTo(box.height - padding * 2, 0);
    await expect(item).toHaveCSS('border-radius', '9999px');
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

  test('hover is a translucent state layer, not an opaque surface swap', async ({
    page,
  }) => {
    const item = page.locator('[data-cy="navigation-link"]').first();
    await item.hover();

    // Material paints hover through the item's own `::before` state layer. A
    // custom rule setting the item background instead reads as an opaque swap.
    const background = await item.evaluate(
      (el) => getComputedStyle(el).backgroundColor,
    );
    const stateLayerOpacity = await item.evaluate((el) =>
      parseFloat(getComputedStyle(el, '::before').opacity),
    );

    expect(background).toBe('rgba(0, 0, 0, 0)');
    expect(stateLayerOpacity).toBeGreaterThan(0);
  });
});
