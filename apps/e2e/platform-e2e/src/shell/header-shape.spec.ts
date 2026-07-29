import { test, expect } from '@playwright/test';
import { mockServiceDiscovery } from '../support/mock-service-discovery';

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
