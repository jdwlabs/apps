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

  test('dashboard shows nav tiles for each MF', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByText('Auth')).toBeVisible();
    await expect(page.getByText('Users')).toBeVisible();
    await expect(page.getByText('Roles')).toBeVisible();
  });

  test('fallback renders when navigating to unknown route', async ({
    page,
  }) => {
    await mockServiceDiscovery(page);
    await page.goto('/unknown-route-xyz');
    await expect(page).toHaveURL('/');
  });
});
