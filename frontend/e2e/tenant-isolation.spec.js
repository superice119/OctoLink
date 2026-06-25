const { expect, test } = require('@playwright/test');
const { installMockBackend } = require('./fixtures/mock-backend');
const { loginThroughUi } = require('./helpers/auth');

test.describe('tenant isolation in rendered device list', () => {
  test.beforeEach(async ({ page }) => {
    await installMockBackend(page);
  });

  test('tenant-a sees only tenant-a devices and tenant-b sees only tenant-b devices', async ({ page }) => {
    await loginThroughUi(page, 'tenant-a@example.com', 'correct-password');
    await page.goto('/devices');

    await expect(page.getByText('TENANT-A-001')).toBeVisible();
    await expect(page.getByText('UNOWNED-001')).toBeVisible();
    await expect(page.getByText('TENANT-B-001')).toHaveCount(0);

    await page.evaluate(() => {
      window.sessionStorage.clear();
      window.localStorage.clear();
    });

    await loginThroughUi(page, 'tenant-b@example.com', 'correct-password');
    await page.goto('/devices');

    await expect(page.getByText('TENANT-B-001')).toBeVisible();
    await expect(page.getByText('TENANT-A-001')).toHaveCount(0);
    await expect(page.getByText('UNOWNED-001')).toHaveCount(0);
  });
});
