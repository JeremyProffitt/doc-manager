import { test, expect } from '@playwright/test';
import { login } from '../helpers/auth';

test.describe('Version History', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('version history shows after save', async ({ page }) => {
    // TODO: Open editor, save, verify version list
  });

  test('can revert to a previous version', async ({ page }) => {
    // TODO: Create multiple versions, revert, verify
  });
});
