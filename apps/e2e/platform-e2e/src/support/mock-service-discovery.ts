import { Page } from '@playwright/test';

const MOCK_ROUTES = [
  {
    path: 'auth',
    remoteName: 'authui',
    moduleName: './Routes',
    url: 'http://localhost:4201/remoteEntry.js',
    icon: 'login',
    title: 'Auth',
    description: 'Authentication',
  },
  {
    path: 'users',
    remoteName: 'usersui',
    moduleName: './Routes',
    url: 'http://localhost:4202/remoteEntry.js',
    icon: 'people',
    title: 'Users',
    description: 'User management',
  },
  {
    path: 'roles',
    remoteName: 'rolesui',
    moduleName: './Routes',
    url: 'http://localhost:4203/remoteEntry.js',
    icon: 'admin_panel_settings',
    title: 'Roles',
    description: 'Role management',
  },
];

export async function mockServiceDiscovery(page: Page): Promise<void> {
  await page.route('**/api/micro-frontends', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_ROUTES),
    });
  });
}
