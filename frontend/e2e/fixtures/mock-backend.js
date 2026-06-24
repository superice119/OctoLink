const transparentPng = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==',
  'base64'
);

const tenantDevices = {
  'tenant-a': [
    {
      SN: 'TENANT-A-001',
      Alias: 'Tenant A gateway',
      ProductClass: 'Router',
      Vendor: 'OctoLink',
      Version: '1.0.0',
      Status: 2,
      Cwmp: 0,
      tenant_id: 'tenant-a'
    },
    {
      SN: 'UNOWNED-001',
      Alias: 'Unowned lab device',
      ProductClass: 'Gateway',
      Vendor: 'OctoLink',
      Version: '1.0.1',
      Status: 0,
      Cwmp: 0,
      tenant_id: null
    },
    {
      SN: '',
      Alias: 'Empty serial device',
      ProductClass: 'Bridge',
      Vendor: 'OctoLink',
      Version: '0.9.0',
      Status: 0,
      Cwmp: 0,
      tenant_id: null
    }
  ],
  'tenant-b': [
    {
      SN: 'TENANT-B-001',
      Alias: 'Tenant B gateway',
      ProductClass: 'Router',
      Vendor: 'OctoLink',
      Version: '2.0.0',
      Status: 2,
      Cwmp: 0,
      tenant_id: 'tenant-b'
    }
  ]
};

function tokenFor({ email, role = 'super_admin', tenantId = 'tenant-a' }) {
  const encode = (value) => Buffer.from(JSON.stringify(value)).toString('base64');

  return `${encode({ alg: 'none', typ: 'JWT' })}.${encode({
    email,
    role,
    tenant_id: tenantId
  })}.signature`;
}

function tenantFromRequest(request) {
  const authorization = request.headers().authorization || '';
  const token = authorization.replace(/^Bearer\s+/i, '');

  try {
    const payload = JSON.parse(Buffer.from(token.split('.')[1], 'base64').toString('utf8'));
    return payload.tenant_id || 'tenant-a';
  } catch {
    return 'tenant-a';
  }
}

function deviceListFor(request, state) {
  const url = new URL(request.url());
  const tenantId = tenantFromRequest(request);
  let devices = [...(state.devicesByTenant[tenantId] || [])];
  const id = url.searchParams.get('id');

  if (id !== null) {
    const device = devices.find((candidate) => candidate.SN === id);
    return device || {};
  }

  return {
    devices,
    total: devices.length,
    page: 0,
    pages: 0
  };
}

async function installMockBackend(page, options = {}) {
  const state = {
    devicesByTenant: structuredClone(options.devicesByTenant || tenantDevices),
    rejectedLogins: options.rejectedLogins || new Set(['wrong@example.com']),
    deleteRequests: []
  };

  await page.route('**/images/logo.png', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'image/png',
      body: transparentPng
    });
  });

  await page.route('**/api/auth/admin/exists', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: 'true' });
  });

  await page.route('**/api/auth/login', async (route, request) => {
    const body = request.postDataJSON();

    if (state.rejectedLogins.has(body.email)) {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'invalid credentials' })
      });
      return;
    }

    const tenantId = body.email.includes('tenant-b') ? 'tenant-b' : 'tenant-a';
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(tokenFor({ email: body.email, tenantId }))
    });
  });

  await page.route('**/api/info/general', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        StatusCount: { Online: 1, Offline: 1 },
        ProductClassCount: [{ productClass: 'Router', count: 1 }],
        VendorsCount: [{ vendor: 'OctoLink', count: 1 }],
        StompRtt: '+10ms',
        MqttRtt: '+12ms',
        WebsocketsRtt: '+8ms'
      })
    });
  });

  await page.route('**/api/device/filterOptions', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        productClasses: ['Router', 'Gateway', 'Bridge'],
        vendors: ['OctoLink'],
        versions: ['1.0.0', '1.0.1', '2.0.0'],
        models: ['Router', 'Gateway', 'Bridge']
      })
    });
  });

  await page.route('**/api/device?**', async (route, request) => {
    if (request.method() === 'DELETE') {
      const url = new URL(request.url());
      const tenantId = tenantFromRequest(request);
      const id = url.searchParams.get('id');
      state.deleteRequests.push({ id, url: request.url() });

      state.devicesByTenant[tenantId] = (state.devicesByTenant[tenantId] || []).filter(
        (device) => device.SN !== id
      );

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ deleted: id })
      });
      return;
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(deviceListFor(request, state))
    });
  });

  return state;
}

module.exports = {
  installMockBackend,
  tenantDevices,
  tokenFor
};
