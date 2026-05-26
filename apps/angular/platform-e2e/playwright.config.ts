import { defineConfig, devices } from '@playwright/test';
import { nxE2EPreset } from '@nx/playwright/preset';
import { workspaceRoot } from '@nx/devkit';

const baseURL = process.env['BASE_URL'] ?? 'http://localhost:4200';

export default defineConfig({
  ...nxE2EPreset(__filename, { testDir: './src' }),
  use: {
    baseURL,
    trace: 'on-first-retry',
  },
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
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
