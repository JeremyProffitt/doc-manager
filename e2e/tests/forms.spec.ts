import { test, expect } from '@playwright/test';
import { login } from '../helpers/auth';
import path from 'path';

test.describe('Form Management', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('forms page loads', async ({ page }) => {
    await page.goto('/forms');
    await expect(page.locator('text=Form Library')).toBeVisible();
  });

  test('upload a PDF form', async ({ page }) => {
    await page.goto('/forms');
    // TODO: Implement full upload flow test
    // Click upload button
    // Select sample-form.pdf
    // Wait for upload complete
    // Verify form appears in library
  });

  test('delete a form', async ({ page }) => {
    // TODO: Create a form first, then delete it
  });
});
