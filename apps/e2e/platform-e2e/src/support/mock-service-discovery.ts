import { Page } from '@playwright/test';

const MOCK_ROUTES = [
  {
    path: 'auth',
    remoteName: 'authui',
    moduleName: './Routes',
    origin: 'http://localhost:4201',
    icon: 'login',
    title: 'Auth',
    description: 'Authentication',
  },
  {
    path: 'users',
    remoteName: 'usersui',
    moduleName: './Routes',
    origin: 'http://localhost:4202',
    icon: 'people',
    title: 'Users',
    description: 'User management',
  },
  {
    path: 'roles',
    remoteName: 'rolesui',
    moduleName: './Routes',
    origin: 'http://localhost:4203',
    icon: 'admin_panel_settings',
    title: 'Roles',
    description: 'Role management',
  },
];

/**
 * Resolves a remote's entry URL from the manifest the build emits beside it.
 *
 * The entry filename is a build-output detail, not a contract: Module
 * Federation renamed it from `remoteEntry.js` to `remoteEntry.mjs`, and the
 * literal that used to sit here kept serving a URL nothing could load. Reading
 * the manifest ties the mock to whatever the build actually produced, so the
 * next rename cannot silently desync it.
 */
async function resolveRemoteEntry(page: Page, origin: string): Promise<string> {
  const manifestUrl = `${origin}/mf-manifest.json`;
  const response = await page.request.get(manifestUrl);
  if (!response.ok()) {
    throw new Error(
      `Could not read ${manifestUrl} (HTTP ${response.status()}) — the remote's build output is missing, so its entry URL cannot be resolved.`,
    );
  }

  const { name, path } = (await response.json()).metaData.remoteEntry;
  if (!name) {
    throw new Error(`${manifestUrl} declares no remote entry filename.`);
  }
  return [origin, path, name].filter(Boolean).join('/');
}

/**
 * One resolution per origin for the life of a worker, sharing the in-flight
 * promise so parallel tests do not each issue their own request.
 *
 * The filename cannot change mid-run, and re-reading it put three manifest
 * fetches on the hot path of every test — enough concurrent requests across
 * workers for the static server to start resetting connections. A rejection is
 * evicted so one failed read does not poison the rest of the worker's tests.
 */
const remoteEntryUrls = new Map<string, Promise<string>>();

function remoteEntryUrl(page: Page, origin: string): Promise<string> {
  let pending = remoteEntryUrls.get(origin);
  if (!pending) {
    pending = resolveRemoteEntry(page, origin).catch((error) => {
      remoteEntryUrls.delete(origin);
      throw error;
    });
    remoteEntryUrls.set(origin, pending);
  }
  return pending;
}

export async function mockServiceDiscovery(page: Page): Promise<void> {
  // BASE_URL targets a deployed environment whose real service-discovery
  // backend serves /api/micro-frontends — mocking it there would hide the
  // exact integration these post-deploy runs exist to verify.
  if (process.env['BASE_URL']) {
    return;
  }

  const routes = await Promise.all(
    MOCK_ROUTES.map(async ({ origin, ...route }) => ({
      ...route,
      url: await remoteEntryUrl(page, origin),
    })),
  );

  await page.route('**/api/micro-frontends', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(routes),
    });
  });
}
