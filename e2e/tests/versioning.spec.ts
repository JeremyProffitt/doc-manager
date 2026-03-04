import { test, expect } from '@playwright/test';
import { login } from '../helpers/auth';

test.describe('Version History', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test.skip('version history shows after save', async ({ page }) => {
    // TODO: Open editor, save, verify ul#version-history has items
  });

  test.skip('can revert to a previous version', async ({ page }) => {
    // TODO: Create multiple versions, revert, verify
  });
});
