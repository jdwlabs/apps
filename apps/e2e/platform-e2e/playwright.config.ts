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
  // One worker per static server. Each is a single process, and every
  // navigation pulls a remote entry plus its chunks through one of them;
  // Playwright's local default of one worker per two cores was enough to have
  // them reset connections outright. CI already pins a single worker, so this
  // only bounds local runs.
  workers: process.env['CI'] ? 1 : 4,
  use: {
    baseURL,
    trace: 'on-first-retry',
  },
  ...(useDeployedEnvironment
    ? {}
    : {
        // `serve-built` rather than `serve-static`: the latter kicks off its own
        // build, and the Angular builder empties the output directory before
        // rewriting it. The server is already accepting connections by then, so
        // Playwright's readiness probe passes against the *previous* build and
        // tests then run against a directory being deleted underneath them —
        // measured as index.html vanishing for ~15s mid-run. The e2e target
        // already depends on its dependencies' builds, so the output is present
        // and current before any of this starts.
        webServer: [
          {
            command: 'pnpm exec nx run container:serve-built',
            url: 'http://localhost:4200',
            reuseExistingServer: !process.env['CI'],
            cwd: workspaceRoot,
            timeout: 180_000,
          },
          {
            command: 'pnpm exec nx run authui:serve-built',
            url: 'http://localhost:4201',
            reuseExistingServer: !process.env['CI'],
            cwd: workspaceRoot,
            timeout: 180_000,
          },
          {
            command: 'pnpm exec nx run usersui:serve-built',
            url: 'http://localhost:4202',
            reuseExistingServer: !process.env['CI'],
            cwd: workspaceRoot,
            timeout: 180_000,
          },
          {
            command: 'pnpm exec nx run rolesui:serve-built',
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
