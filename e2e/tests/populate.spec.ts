import { test, expect } from '@playwright/test';
import { login } from '../helpers/auth';

test.describe('Form Population', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test.skip('preview shows customer data on form', async ({ page }) => {
    // TODO: Navigate to populate page
    // Verify select#customer-selector visible
    // Verify customer data overlays on div#form-canvas
  });

  test.skip('can download populated PDF', async ({ page }) => {
    // TODO: Click a#download-btn, verify PDF file downloaded
  });

  test.skip('can switch customers in preview', async ({ page }) => {
    // TODO: Change select#customer-selector, verify data updates
  });
});
