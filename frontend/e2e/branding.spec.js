const { expect, test } = require('@playwright/test');
const { installMockBackend } = require('./fixtures/mock-backend');
const { seedAuthenticatedSession } = require('./helpers/auth');

test.describe('OctoLink branding', () => {
  test('renders OctoLink login branding and no Oktopus remnants', async ({ page }) => {
    await installMockBackend(page);
    await page.goto('/auth/login');

    await expect(page).toHaveTitle(/Login \| OctoLink/);
    await expect(page.locator('link[rel="icon"][href="/favicon.ico"]')).toHaveCount(1);
    await expect(page.locator('img[src*="/images/logo.png"]')).toBeVisible();
    await expect(page.locator('img[alt="OctoLink logo"]')).toBeVisible();
    await expect(page.getByText(/Oktopus/i)).toHaveCount(0);
    await expect(page.locator('[src*="oktopus" i], [href*="oktopus" i], [alt*="oktopus" i]')).toHaveCount(0);
  });

  test('renders OctoLink dashboard branding and no Oktopus remnants', async ({ page }) => {
    await installMockBackend(page);
    await seedAuthenticatedSession(page);
    await page.goto('/devices');

    await expect(page).toHaveTitle(/OctoLink/);
    await expect(page.locator('img[src*="/images/logo.png"]')).toBeVisible();
    await expect(page.locator('img[alt="OctoLink logo"]')).toBeVisible();
    await expect(page.getByText('Powered by')).toBeVisible();
    await expect(page.getByText(/Oktopus/i)).toHaveCount(0);
    await expect(page.locator('[src*="oktopus" i], [href*="oktopus" i], [alt*="oktopus" i]')).toHaveCount(0);
  });
});
