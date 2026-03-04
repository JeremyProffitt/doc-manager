import { test, expect } from '@playwright/test';
import { login } from '../helpers/auth';

test.describe('Settings', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('settings page shows field configuration', async ({ page }) => {
    await page.goto('/settings/fields');
    await expect(page.locator('h1')).toContainText('Field Settings');
    await expect(page.locator('div#fieldsContainer')).toBeVisible();
  });

  test('settings page shows default seeded fields', async ({ page }) => {
    await page.goto('/settings/fields');
    // Check that at least some default field names appear as input values
    const fieldInputs = page.locator('input.field-name');
    await expect(fieldInputs.first()).toBeVisible();
  });

  test.skip('can add a custom field', async ({ page }) => {
    // TODO: Add field, verify it appears
  });
});
