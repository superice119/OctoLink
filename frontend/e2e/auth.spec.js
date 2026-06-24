const { expect, test } = require('@playwright/test');
const { installMockBackend } = require('./fixtures/mock-backend');
const { loginThroughUi } = require('./helpers/auth');

test.describe('login flow', () => {
  test.beforeEach(async ({ page }) => {
    await installMockBackend(page);
  });

  test('accepts valid credentials and redirects to the dashboard', async ({ page }) => {
    await loginThroughUi(page, 'tenant-a@example.com', 'correct-password');

    await expect(page).toHaveURL(/\/$/);
    await expect(page.getByText('Total Devices')).toBeVisible();
    await expect(page.getByText('Devices Type')).toBeVisible();
    await expect(page.evaluate(() => window.sessionStorage.getItem('authenticated'))).resolves.toBe('true');
  });

  test('rejects invalid credentials without authenticating the session', async ({ page }) => {
    await loginThroughUi(page, 'wrong@example.com', 'bad-password');

    await expect(page.getByText('Please check your email and password')).toBeVisible();
    await expect(page).toHaveURL(/\/auth\/login$/);
    await expect(page.evaluate(() => window.sessionStorage.getItem('authenticated'))).resolves.not.toBe('true');
  });
});
