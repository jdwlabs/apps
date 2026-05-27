import { test, expect } from '@playwright/test';
import { mockServiceDiscovery } from '../support/mock-service-discovery';

test.describe('Roles', () => {
  test.beforeEach(async ({ page }) => {
    await mockServiceDiscovery(page);
  });

  test('dashboard renders', async ({ page }) => {
    await page.goto('/roles');
    await expect(page.locator('body')).toBeVisible();
  });

  test('roles list page renders', async ({ page }) => {
    await page.goto('/roles/roles');
    await expect(page.locator('body')).toBeVisible();
  });

  test.describe('Create Role Form', () => {
    test.beforeEach(async ({ page }) => {
      await page.goto('/roles/role');
    });

    test('renders in create mode', async ({ page }) => {
      await expect(page.locator('[data-cy="title"]')).toContainText('Add Role');
      await expect(page.locator('[data-cy="name-field"]')).toBeVisible();
      await expect(page.locator('[data-cy="description-field"]')).toBeVisible();
      await expect(page.locator('[data-cy="submit-button"]')).toBeDisabled();
      await expect(page.locator('[data-cy="reset-button"]')).toBeEnabled();
    });

    test('shows validation errors on blur', async ({ page }) => {
      await page.locator('[data-cy="name-field"] input').click();
      await page.locator('[data-cy="name-field"] input').blur();
      await expect(page.locator('[data-cy="name-field"]')).toContainText(
        'Name is required',
      );

      await page.locator('[data-cy="description-field"] input').click();
      await page.locator('[data-cy="description-field"] input').blur();
      await expect(page.locator('[data-cy="description-field"]')).toContainText(
        'Description is required',
      );
    });

    test('submit enables when form is valid', async ({ page }) => {
      await page.locator('[data-cy="name-field"] input').fill('ADMIN');
      await page
        .locator('[data-cy="description-field"] input')
        .fill('Admin role');
      await expect(page.locator('[data-cy="submit-button"]')).toBeEnabled();
    });
  });
});
