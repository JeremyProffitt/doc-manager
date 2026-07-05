import { Page, expect } from '@playwright/test';

export async function login(page: Page) {
  const email = process.env.TEST_USER_EMAIL || 'proffitt.jeremy@gmail.com';
  const password = process.env.TEST_USER_PASSWORD;
  if (!password) {
    throw new Error('TEST_USER_PASSWORD environment variable is required');
  }

  await page.goto('/login');
  await page.fill('input[name="email"]', email);
  await page.fill('input[name="password"]', password);
  await page.click('button[type="submit"]');
  // After login, we land on the home page. URL may include stage prefix (e.g. /prod/)
  await page.waitForURL(/(\/|\/$)$/);
}
