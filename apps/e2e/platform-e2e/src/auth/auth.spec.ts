import { test, expect } from '@playwright/test';
import { mockServiceDiscovery } from '../support/mock-service-discovery';

test.describe('Auth', () => {
  test.beforeEach(async ({ page }) => {
    await mockServiceDiscovery(page);
  });

  test.describe('Sign In', () => {
    test.beforeEach(async ({ page }) => {
      await page.goto('/auth/sign-in');
    });

    test('renders sign-in form', async ({ page }) => {
      await expect(page.locator('[data-cy="title"]')).toContainText('Sign In');
      await expect(page.locator('[data-cy="email-sign-in"]')).toBeVisible();
    });

    test('sign-up link navigates to sign-up', async ({ page }) => {
      await page.locator('[data-cy="sign-up-link"]').click();
      await expect(page).toHaveURL(/sign-up/);
    });
  });

  test.describe('Sign Up', () => {
    test.beforeEach(async ({ page }) => {
      await page.goto('/auth/sign-up');
    });

    test('renders sign-up form', async ({ page }) => {
      await expect(page.locator('[data-cy="title"]')).toContainText('Sign Up');
      await expect(
        page.locator('[data-cy="email-address-field"]'),
      ).toBeVisible();
      await expect(page.locator('[data-cy="password-field"]')).toBeVisible();
      await expect(
        page.locator('[data-cy="confirm-password-field"]'),
      ).toBeVisible();
      await expect(page.locator('[data-cy="submit-button"]')).toBeDisabled();
    });

    // Focused rather than clicked: an outline field's un-floated label sits over
    // its own input and forwards the click through its `for`, but where the
    // label is wider than half the input — as it is once a suffix button
    // narrows the box — it covers the centre point a click is aimed at, and
    // Playwright refuses the click. Touching the control is what these
    // assertions are about, and focus is the direct way to say so.
    test('shows validation errors on blur', async ({ page }) => {
      await page.locator('[data-cy="email-address-field"] input').focus();
      await page.locator('[data-cy="email-address-field"] input').blur();
      await expect(
        page.locator('[data-cy="email-address-field"]'),
      ).toContainText('Email is required');

      await page.locator('[data-cy="password-field"] input').focus();
      await page.locator('[data-cy="password-field"] input').blur();
      await expect(page.locator('[data-cy="password-field"]')).toContainText(
        'Password is required',
      );
    });

    test('submit enables when form is valid', async ({ page }) => {
      await page
        .locator('[data-cy="email-address-field"] input')
        .fill('test@example.com');
      await page.locator('[data-cy="password-field"] input').fill('Password1!');
      await page
        .locator('[data-cy="confirm-password-field"] input')
        .fill('Password1!');
      await expect(page.locator('[data-cy="submit-button"]')).toBeEnabled();
    });
  });

  test.describe('Forbidden', () => {
    test('renders forbidden page', async ({ page }) => {
      await page.goto('/auth/forbidden');
      await expect(page.locator('body')).toBeVisible();
    });
  });
});
