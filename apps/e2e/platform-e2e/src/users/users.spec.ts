import { test, expect } from '@playwright/test';
import { mockServiceDiscovery } from '../support/mock-service-discovery';

test.describe('Users', () => {
  test.beforeEach(async ({ page }) => {
    await mockServiceDiscovery(page);
  });

  test('dashboard renders', async ({ page }) => {
    await page.goto('/users');
    await expect(page.locator('body')).toBeVisible();
  });

  test('users list page renders', async ({ page }) => {
    await page.goto('/users/users');
    await expect(page.locator('body')).toBeVisible();
  });

  test.describe('Profile Form', () => {
    test.beforeEach(async ({ page }) => {
      // No backend runs in this suite, so the real "no profile yet" 404
      // (a text/plain ResourceNotFoundException body, per the
      // profile-service contract) is mocked rather than left to a bare
      // connection failure, which the app must not read as "no profile".
      await page.route('**/api/profiles/by-user/*', async (route) => {
        await route.fulfill({
          status: 404,
          contentType: 'text/plain',
          body: 'Profile not found with id new',
        });
      });
      await page.goto('/users/profile/new');
    });

    test('renders in create mode', async ({ page }) => {
      await expect(page.locator('[data-cy="title"]')).toContainText(
        'Add Profile',
      );
      await expect(page.locator('[data-cy="submit-button"]')).toBeDisabled();
      await expect(page.locator('[data-cy="reset-button"]')).toBeEnabled();
    });

    test('shows validation errors on blur', async ({ page }) => {
      await page.locator('[data-cy="first-name-field"] input').click();
      await page.locator('[data-cy="first-name-field"] input').blur();
      await expect(page.locator('[data-cy="first-name-field"]')).toContainText(
        'First name is required',
      );

      await page.locator('[data-cy="last-name-field"] input').click();
      await page.locator('[data-cy="last-name-field"] input').blur();
      await expect(page.locator('[data-cy="last-name-field"]')).toContainText(
        'Last name is required',
      );
    });
  });

  test.describe('Account Form', () => {
    test.beforeEach(async ({ page }) => {
      await page.goto('/users/account');
    });

    test('renders form', async ({ page }) => {
      await expect(page.locator('[data-cy="title"]')).toContainText('Account');
      await expect(
        page.locator('[data-cy="email-address-field"]'),
      ).toBeVisible();
      await expect(page.locator('[data-cy="password-field"]')).toBeVisible();
      await expect(page.locator('[data-cy="submit-button"]')).toBeDisabled();
    });
  });
});
