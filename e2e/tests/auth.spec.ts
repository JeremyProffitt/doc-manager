import { test, expect } from '@playwright/test';
import { login } from '../helpers/auth';

test.describe('Authentication', () => {
  test('login with valid credentials', async ({ page }) => {
    await login(page);
    await expect(page).toHaveURL('/');
    await expect(page.locator('text=Forms')).toBeVisible();
  });

  test('login with wrong password shows error', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[name="email"]', 'proffitt.jeremy@gmail.com');
    await page.fill('input[name="password"]', 'wrongpassword');
    await page.click('button[type="submit"]');
    await expect(page.locator('text=Invalid email or password')).toBeVisible();
  });

  test('login with non-existent email shows same error', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[name="email"]', 'nobody@example.com');
    await page.fill('input[name="password"]', 'somepassword');
    await page.click('button[type="submit"]');
    await expect(page.locator('text=Invalid email or password')).toBeVisible();
  });

  test('logout redirects to login', async ({ page }) => {
    await login(page);
    await page.click('text=Logout');
    await expect(page).toHaveURL(/\/login/);
  });

  test('accessing protected route without auth redirects to login', async ({ page }) => {
    await page.goto('/forms');
    await expect(page).toHaveURL(/\/login/);
  });
});
