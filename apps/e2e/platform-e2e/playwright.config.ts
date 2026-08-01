import { defineConfig, devices } from '@playwright/test';
import { nxE2EPreset } from '@nx/playwright/preset';
import { workspaceRoot } from '@nx/devkit';

const baseURL = process.env['BASE_URL'] ?? 'http://localhost:4200';

// An explicit BASE_URL means the platform is already running somewhere real
// (e.g. staging via the deployments repo pipeline) — spinning up local static
// servers would be wasted work and could shadow the environment under test.
const useDeployedEnvironment = !!process.env['BASE_URL'];

export default defineConfig({
  ...nxE2EPreset(__filename, { testDir: './src' }),
  // The remotes are served by four single-process static servers, and every
  // navigation pulls a remote entry plus its chunks through them. Playwright's
  // local default of one worker per two cores saturates them: connections are
  // reset mid-navigation and the route renders empty. One worker per server
  // holds. CI already pins a single worker, so this only bounds local runs.
  workers: process.env['CI'] ? 1 : 4,
  use: {
    baseURL,
    trace: 'on-first-retry',
  },
  ...(useDeployedEnvironment
    ? {}
    : {
        webServer: [
          {
            command: 'pnpm exec nx run container:serve-static',
            url: 'http://localhost:4200',
            reuseExistingServer: !process.env['CI'],
            cwd: workspaceRoot,
            timeout: 180_000,
          },
          {
            command: 'pnpm exec nx run authui:serve-static',
            url: 'http://localhost:4201',
            reuseExistingServer: !process.env['CI'],
            cwd: workspaceRoot,
            timeout: 180_000,
          },
          {
            command: 'pnpm exec nx run usersui:serve-static',
            url: 'http://localhost:4202',
            reuseExistingServer: !process.env['CI'],
            cwd: workspaceRoot,
            timeout: 180_000,
          },
          {
            command: 'pnpm exec nx run rolesui:serve-static',
            url: 'http://localhost:4203',
            reuseExistingServer: !process.env['CI'],
            cwd: workspaceRoot,
            timeout: 180_000,
          },
        ],
      }),
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
