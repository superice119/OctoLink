const { expect, test } = require('@playwright/test');
const { installMockBackend } = require('./fixtures/mock-backend');
const { seedAuthenticatedSession } = require('./helpers/auth');

async function openDevices(page, tenantId = 'tenant-a') {
  await installMockBackend(page);
  await seedAuthenticatedSession(page, {
    email: `${tenantId}@example.com`,
    tenantId
  });
  await page.goto('/devices');
  await expect(page.locator('main').getByText('Devices', { exact: true })).toBeVisible();
}

test.describe('device list and deletion UI', () => {
  test('renders one table row for each backend device, including unowned and empty-SN rows', async ({ page }) => {
    await openDevices(page);

    await expect(page.locator('tbody tr')).toHaveCount(3);
    await expect(page.getByText('TENANT-A-001')).toBeVisible();
    await expect(page.getByText('UNOWNED-001')).toBeVisible();
    await expect(page.getByRole('row').filter({ hasText: 'Bridge' }).filter({ hasText: '0.9.0' })).toBeVisible();
    await expect(page.getByText('1–3 of 3')).toBeVisible();
  });

  test('deletes a normal device and refreshes the rendered row count', async ({ page }) => {
    const state = await installMockBackend(page);
    await seedAuthenticatedSession(page);
    await page.goto('/devices');

    const row = page.getByRole('row').filter({ hasText: 'TENANT-A-001' });
    await row.locator('button').last().click();
    await expect(page.getByText('Are you sure you want to remove')).toBeVisible();
    await page.getByRole('button', { name: /Apply/i }).click();

    await expect(page.getByText('TENANT-A-001')).toHaveCount(0);
    await expect(page.locator('tbody tr')).toHaveCount(2);
    expect(state.deleteRequests).toEqual(
      expect.arrayContaining([expect.objectContaining({ id: 'TENANT-A-001' })])
    );
  });

  test('allows deleting an unowned device from the UI', async ({ page }) => {
    const state = await installMockBackend(page);
    await seedAuthenticatedSession(page);
    await page.goto('/devices');

    const row = page.getByRole('row').filter({ hasText: 'UNOWNED-001' });
    await row.locator('button').last().click();
    await page.getByRole('button', { name: /Apply/i }).click();

    await expect(page.getByText('UNOWNED-001')).toHaveCount(0);
    expect(state.deleteRequests).toEqual(
      expect.arrayContaining([expect.objectContaining({ id: 'UNOWNED-001' })])
    );
  });

  test('does not emit DELETE /api/device?id= for an empty-SN device', async ({ page }) => {
    test.skip(!process.env.E2E_RUN_FIX_DEPENDENT, 'Enable after the empty-SN delete fix is merged.');

    const state = await installMockBackend(page);
    await seedAuthenticatedSession(page);
    await page.goto('/devices');

    const row = page.getByRole('row').filter({ hasText: 'Bridge' }).filter({ hasText: '0.9.0' });
    await row.locator('button').last().click();
    await page.getByRole('button', { name: /Apply/i }).click();

    expect(state.deleteRequests).not.toEqual(
      expect.arrayContaining([expect.objectContaining({ id: '' })])
    );
  });
});
