import { test, expect } from '@playwright/test';
import { login } from '../helpers/auth';

test.describe('Form Editor', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test.skip('editor page loads for a form', async ({ page }) => {
    // TODO: Navigate to an uploaded form's editor
    // Verify canvas: div#form-canvas
    // Verify fields panel: ul#fields-list-ul
    // Verify version history: ul#version-history
  });

  test.skip('can drag a field to a new position', async ({ page }) => {
    // TODO: Drag field, save, reload, verify position
  });

  test.skip('can save field placements', async ({ page }) => {
    // TODO: Make changes, click save, verify version incremented
  });
});
