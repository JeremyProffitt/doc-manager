import { test, expect } from '@playwright/test';
import { login } from '../helpers/auth';

test.describe('Customer Management', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('customer list shows seeded customers', async ({ page }) => {
    await page.goto('/customers');
    await expect(page.locator('text=John Smith')).toBeVisible();
    await expect(page.locator('text=Jane Doe')).toBeVisible();
  });

  test('create a new customer', async ({ page }) => {
    await page.goto('/customers');
    await page.click('text=Add Customer');
    // TODO: Fill form and verify creation
  });

  test('edit a customer', async ({ page }) => {
    // TODO: Edit and verify
  });

  test('delete a customer', async ({ page }) => {
    // TODO: Create test customer, delete, verify
  });
});
