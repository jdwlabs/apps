import { Page } from '@playwright/test';

const MOCK_ROUTES = [
  {
    path: 'auth',
    remoteName: 'authui',
    moduleName: './Routes',
    url: 'http://localhost:4201',
    icon: 'login',
    title: 'Auth',
    description: 'Authentication',
  },
  {
    path: 'users',
    remoteName: 'usersui',
    moduleName: './Routes',
    url: 'http://localhost:4202',
    icon: 'people',
    title: 'Users',
    description: 'User management',
  },
  {
    path: 'roles',
    remoteName: 'rolesui',
    moduleName: './Routes',
    url: 'http://localhost:4203',
    icon: 'admin_panel_settings',
    title: 'Roles',
    description: 'Role management',
  },
];

// Service discovery answers with bare origins, and module federation appends
// whichever entry filename the bundler emitted. A url that already names a file
// is forwarded untouched, so spelling one here pins the suite to one bundler's
// default — and the day that default moved, every remote 404ed and the shell
// quietly served its fallback page instead of failing.
const ORIGIN_ONLY = /^https?:\/\/[^/]+\/?$/;

const misdirected = MOCK_ROUTES.filter((route) => !ORIGIN_ONLY.test(route.url));
if (misdirected.length > 0) {
  const offenders = misdirected
    .map((route) => `${route.remoteName} -> ${route.url}`)
    .join(', ');
  throw new Error(
    `Mocked remote urls must be origins, not module federation entry points: ${offenders}`,
  );
}

const MOCK_REMOTES = Object.fromEntries(
  MOCK_ROUTES.map((route) => [route.remoteName, route.url]),
);

const json = (body: unknown) => ({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify(body),
});

export async function mockServiceDiscovery(page: Page): Promise<void> {
  // BASE_URL targets a deployed environment whose real service-discovery
  // backend serves these routes — mocking it there would hide the exact
  // integration these post-deploy runs exist to verify.
  if (process.env['BASE_URL']) {
    return;
  }
  // The shell will not bootstrap until /api/remotes settles, and nothing local
  // answers on the discovery host. Left unmocked, every navigation pays for
  // that request to fail before the page renders at all.
  await page.route('**/api/remotes', async (route) => {
    await route.fulfill(json(MOCK_REMOTES));
  });
  await page.route('**/api/micro-frontends', async (route) => {
    await route.fulfill(json(MOCK_ROUTES));
  });
}
