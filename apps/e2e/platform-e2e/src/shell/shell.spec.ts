import { test, expect } from '@playwright/test';
import { mockServiceDiscovery } from '../support/mock-service-discovery';

test.describe('Platform Shell', () => {
  test.beforeEach(async ({ page }) => {
    await mockServiceDiscovery(page);
  });

  test('loads dashboard', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveURL('/');
  });

  test('header renders with app title', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('jdw-header')).toBeVisible();
  });

  // Matching on text alone is ambiguous here: a tile's title is a substring of
  // its own description ("Auth" of "Authentication"), so the title has to be
  // read from the tile that carries it.
  test('dashboard shows nav tiles for each MF', async ({ page }) => {
    await page.goto('/');
    for (const title of ['Auth', 'Users', 'Roles']) {
      const tile = page.locator(`[data-cy="${title.toLowerCase()}-tile"]`);
      await expect(tile.locator('[data-cy="tile-link"]')).toHaveText(title);
    }
  });

  // A remote whose entry point cannot be fetched does not raise: the shell
  // swaps in its fallback page, and the breakage surfaces as a dozen unrelated
  // missing-element failures elsewhere. Named here, it stays one failure.
  test('each micro-frontend mounts instead of the shell fallback', async ({
    page,
  }) => {
    for (const path of ['/auth/sign-in', '/users/account', '/roles/role']) {
      await page.goto(path);
      await expect(
        page.locator('jdw-fallback'),
        `${path} served the shell fallback, so its remote never loaded`,
      ).toHaveCount(0);
    }
  });

  test('fallback renders when navigating to unknown route', async ({
    page,
  }) => {
    await mockServiceDiscovery(page);
    await page.goto('/unknown-route-xyz');
    await expect(page).toHaveURL('/');
  });
});
