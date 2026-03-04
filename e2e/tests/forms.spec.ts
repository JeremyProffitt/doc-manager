import { test, expect } from '@playwright/test';
import { login } from '../helpers/auth';

test.describe('Form Management', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('forms page loads', async ({ page }) => {
    await page.goto('/forms');
    await expect(page.locator('h1')).toContainText('Form Library');
  });

  test('forms page has upload area', async ({ page }) => {
    await page.goto('/forms');
    await expect(page.locator('div#drop-zone')).toBeVisible();
    await expect(page.locator('input#file-input')).toBeAttached();
  });

  test.skip('upload a PDF form', async ({ page }) => {
    // TODO: Implement full upload flow test with pre-signed URL
  });

  test.skip('delete a form', async ({ page }) => {
    // TODO: Create a form first, then delete it
  });
});
