import { test, expect } from '@playwright/test';
import { login } from '../helpers/auth';

test.describe('Customer Management', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('customer list shows seeded customers', async ({ page }) => {
    await page.goto('/customers');
    await expect(page.locator('h1')).toContainText('Customers');
    await expect(page.locator('text=John Smith')).toBeVisible();
    await expect(page.locator('text=Jane Doe')).toBeVisible();
  });

  test('can navigate to add customer form', async ({ page }) => {
    await page.goto('/customers');
    await page.click('a[href="/customers/new"]');
    await expect(page.locator('h1')).toContainText('Add Customer');
    await expect(page.locator('input[name="name"]')).toBeVisible();
    await expect(page.locator('input[name="business"]')).toBeVisible();
  });

  test.skip('edit a customer', async ({ page }) => {
    // TODO: Edit and verify
  });

  test.skip('delete a customer', async ({ page }) => {
    // TODO: Create test customer, delete, verify
  });
});
