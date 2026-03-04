import { test, expect } from '@playwright/test';
import { login } from '../helpers/auth';

test.describe('Smoke Tests', () => {
  test('redirects unauthenticated users to login', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveURL(/\/login/);
  });

  test('login page loads', async ({ page }) => {
    await page.goto('/login');
    await expect(page.locator('text=Doc-Manager')).toBeVisible();
    await expect(page.locator('input[name="email"]')).toBeVisible();
    await expect(page.locator('input[name="password"]')).toBeVisible();
    await expect(page.locator('button[type="submit"]')).toBeVisible();
  });

  test('can log in and see dashboard', async ({ page }) => {
    await login(page);
    await expect(page.locator('text=Dashboard')).toBeVisible();
  });
});
