import { test, expect } from '@playwright/test';
import { login } from '../helpers/auth';

test.describe('Form Population', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('preview shows customer data on form', async ({ page }) => {
    // TODO: Navigate to populate page, verify customer data visible
  });

  test('can download populated PDF', async ({ page }) => {
    // TODO: Click download, verify PDF file downloaded
  });

  test('can switch customers in preview', async ({ page }) => {
    // TODO: Change customer selector, verify data updates
  });
});
