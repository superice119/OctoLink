const { expect, test } = require('@playwright/test');
const { installMockBackend } = require('./fixtures/mock-backend');
const { seedAuthenticatedSession } = require('./helpers/auth');

// Regression coverage for WS-35: three layout bugs reported on a 13.3"
// 1920x1080 panel at Windows 150% scale (~1280px CSS viewport).
//   1. Docs nav item overlapped / click-hijacked by the absolutely
//      positioned "Powered by" footer.
//   2. Card title text overlapping the 56x56 avatar icon.
//   3. Four overview cards crammed into one row in the lg band, which
//      narrowed each card enough to trigger bug #2.

const NARROW = { width: 1280, height: 640 }; // genuinely short, forces nav scroll
const WIDE = { width: 1920, height: 1080 };

function intersects(a, b) {
  return !(
    a.x + a.width <= b.x ||
    b.x + b.width <= a.x ||
    a.y + a.height <= b.y ||
    b.y + b.height <= a.y
  );
}

async function openOverview(page) {
  await installMockBackend(page);
  await seedAuthenticatedSession(page);
  await page.goto('/');
  // Cards only render once general info resolves.
  await expect(page.getByText('STOMP Connection')).toBeVisible();
}

async function assertNoTitleIconOverlap(page, titleText) {
  const card = page.locator('.MuiCard-root').filter({ hasText: titleText });
  await expect(card).toHaveCount(1);
  const titleBox = await card.getByText(titleText).boundingBox();
  const avatarBox = await card.locator('.MuiAvatar-root').boundingBox();
  expect(titleBox, `title box for "${titleText}"`).not.toBeNull();
  expect(avatarBox, `avatar box for "${titleText}"`).not.toBeNull();
  expect(
    intersects(titleBox, avatarBox),
    `"${titleText}" title must not overlap its icon at ${NARROW.width}px`
  ).toBe(false);
}

test.describe('WS-35 narrow-screen dashboard layout', () => {
  test.use({ viewport: NARROW });

  test('Docs nav item is visible and clickable (not covered by footer)', async ({ page }) => {
    await openOverview(page);

    const docs = page.getByRole('link', { name: 'Docs' });
    await expect(docs).toBeVisible();
    // trial click runs Playwright's occlusion hit-test without navigating to
    // the external wiki — fails if the footer overlays the Docs item.
    await docs.click({ trial: true });
  });

  test('overview card titles do not overlap their icons', async ({ page }) => {
    await openOverview(page);

    for (const title of [
      'Total Devices',
      'STOMP Connection',
      'MQTT Connection',
      'WebSockets Connection'
    ]) {
      await assertNoTitleIconOverlap(page, title);
    }
  });
});

test.describe('WS-35 wide-screen no-regression', () => {
  test.use({ viewport: WIDE });

  test('the four overview cards still share a single row at 1920px', async ({ page }) => {
    await openOverview(page);

    const titles = [
      'Total Devices',
      'STOMP Connection',
      'MQTT Connection',
      'WebSockets Connection'
    ];
    const tops = [];
    for (const title of titles) {
      const card = page.locator('.MuiCard-root').filter({ hasText: title });
      const box = await card.boundingBox();
      expect(box, `card box for "${title}"`).not.toBeNull();
      tops.push(box.y);
    }
    const spread = Math.max(...tops) - Math.min(...tops);
    expect(spread, 'all four cards should align on one row').toBeLessThan(5);
  });
});
