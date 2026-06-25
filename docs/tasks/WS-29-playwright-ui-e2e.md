# WS-29 Playwright UI E2E Test Cases

## Environment confirmation

- Runtime browser capability confirmed with headless Chromium via `npx --yes playwright screenshot`.
- Project-local Playwright runner is installed in `frontend/` as `@playwright/test`.
- Default local command starts the Next.js frontend on `127.0.0.1:3100` and uses mocked REST responses for deterministic UI coverage.

## Commands

```bash
cd frontend
npm run test:e2e
```

For an already deployed UI, set `PLAYWRIGHT_BASE_URL` and provide matching backend data or extend the fixtures:

```bash
cd frontend
PLAYWRIGHT_BASE_URL=http://39.97.250.156 npm run test:e2e
```

Fix-dependent regression guards are parked behind an explicit flag until the sister fixes land:

```bash
cd frontend
npm run test:e2e:fix-dependent
```

## Test cases

| ID | Area | Scenario | Expected result | Current automation |
| --- | --- | --- | --- | --- |
| WS29-AUTH-001 | Login | Valid credentials | User lands on dashboard and session is authenticated | Active |
| WS29-AUTH-002 | Login | Invalid credentials | Error is shown and session is not authenticated | Active |
| WS29-DEV-001 | Devices | Backend returns three devices | UI renders exactly three body rows and the pagination count is 3 | Active |
| WS29-DEV-002 | Devices | Dataset includes empty SN and unowned devices | Rows render and action controls remain available | Active |
| WS29-DEV-003 | Devices | Delete normal device | DELETE request uses the selected SN and the row disappears | Active |
| WS29-DEV-004 | Devices | Delete unowned device | DELETE request uses the unowned device SN and the row disappears | Active |
| WS29-DEV-005 | Devices | Attempt delete on empty-SN device | UI must not emit `DELETE /api/device?id=` | Fix-dependent guard |
| WS29-TENANT-001 | Tenant isolation | Login as tenant-a then tenant-b | Each tenant sees only its own rendered device rows | Active |
| WS29-BRAND-001 | Branding | Login page render | OctoLink title, favicon, and logos render; no Oktopus text/assets | Active |
| WS29-BRAND-002 | Branding | Dashboard/device page render | OctoLink title, logos, powered-by footer render; no Oktopus text/assets | Active |

## Notes

- The active suite is intentionally mock-backed so render-layer regressions run before a full cloud stack is available.
- The empty-SN delete guard documents the expected behavior and should be enabled in CI after Cloud_Guru's fix prevents invalid delete requests.
