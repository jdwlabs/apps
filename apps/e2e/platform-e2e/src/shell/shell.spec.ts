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

  // Matched on the tile rather than on loose page text: a tile's title is a
  // prefix of its own description ("Auth" / "Authentication"), so a text match
  // resolves to both the link and the body copy.
  test('dashboard shows nav tiles for each MF', async ({ page }) => {
    await page.goto('/');
    for (const title of ['Auth', 'Users', 'Roles']) {
      const tile = page.locator(`[data-cy="${title.toLowerCase()}-tile"]`);
      await expect(tile).toBeVisible();
      await expect(tile.locator('[data-cy="tile-link"]')).toHaveText(title);
    }
  });

  test('fallback renders when navigating to unknown route', async ({
    page,
  }) => {
    await page.goto('/unknown-route-xyz');
    await expect(page).toHaveURL('/');
  });
});

// Not about the status code so much as the second request: a lone `%` used to
// reject out of the static server's async handler and terminate the process,
// which serves every origin the suite addresses.
test('malformed percent-escape is refused without stopping the server', async ({
  request,
}) => {
  expect((await request.get('/%', { maxRedirects: 0 })).status()).toBe(400);
  expect((await request.get('/')).status()).toBe(200);
});
