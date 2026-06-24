const { tokenFor } = require('../fixtures/mock-backend');

async function seedAuthenticatedSession(page, options = {}) {
  const {
    email = 'tenant-a@example.com',
    role = 'super_admin',
    tenantId = 'tenant-a'
  } = options;
  const token = tokenFor({ email, role, tenantId });

  await page.addInitScript(
    ({ email: seededEmail, seededToken }) => {
      window.sessionStorage.setItem('authenticated', 'true');
      window.sessionStorage.setItem('email', seededEmail);
      window.localStorage.setItem('token', seededToken);
    },
    { email, seededToken: token }
  );

  return token;
}

async function loginThroughUi(page, email, password) {
  await page.goto('/auth/login');
  await page.getByLabel('Email Address').fill(email);
  await page.getByLabel('Password').fill(password);
  await page.getByRole('button', { name: 'Continue' }).click();
}

module.exports = {
  loginThroughUi,
  seedAuthenticatedSession
};
