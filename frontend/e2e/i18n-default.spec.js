const { expect, test } = require('@playwright/test');
const { installMockBackend } = require('./fixtures/mock-backend');

const LANGUAGE_STORAGE_KEY = 'octolink.lang';

// Locks the boss's 2026-06-30 decision: the console default is English (WS-33),
// which supersedes WS-30's original "default Chinese" DoD. Guards against the
// default silently flipping again (it has flip-flopped zh<->en once before).
test.describe('i18n default language', () => {
  test('first visit with no saved preference renders English', async ({ page }) => {
    // Ensure a clean slate: no localStorage choice and no cookie before the app boots.
    await page.addInitScript((key) => {
      try {
        window.localStorage.removeItem(key);
        document.cookie = `${key}=; Max-Age=0; path=/`;
      } catch (err) {
        /* storage may be unavailable in some contexts; ignore */
      }
    }, LANGUAGE_STORAGE_KEY);

    await installMockBackend(page);
    await page.goto('/auth/login');

    // English UI strings, not the Chinese catalog.
    await expect(page.getByLabel('Email Address')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Continue' })).toBeVisible();
    await expect(page.getByText('邮箱地址')).toHaveCount(0);

    // Document language reflects the English default.
    await expect(page.locator('html')).toHaveAttribute('lang', 'en');
  });

  test('a saved Chinese preference still wins over the English default', async ({ page }) => {
    // The default only applies when there is no preference; an explicit choice persists.
    await page.addInitScript((key) => {
      window.localStorage.setItem(key, 'zh');
    }, LANGUAGE_STORAGE_KEY);

    await installMockBackend(page);
    await page.goto('/auth/login');

    await expect(page.locator('html')).toHaveAttribute('lang', 'zh');
    await expect(page.getByText('邮箱地址')).toBeVisible();
  });
});
