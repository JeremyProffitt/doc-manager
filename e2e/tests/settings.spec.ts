import { test, expect } from '@playwright/test';
import { login } from '../helpers/auth';

test.describe('Settings', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('settings page shows default fields', async ({ page }) => {
    await page.goto('/settings/fields');
    await expect(page.locator('text=Name')).toBeVisible();
    await expect(page.locator('text=Business')).toBeVisible();
    await expect(page.locator('text=Phone Number')).toBeVisible();
  });

  test('can add a custom field', async ({ page }) => {
    // TODO: Add field, verify it appears
  });
});
