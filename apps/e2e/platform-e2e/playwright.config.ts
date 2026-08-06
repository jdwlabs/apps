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
  // Local is set from a full-suite sweep on a 24-core workstation: green and
  // connection-error-free through 16, falling off at 24 on browser CPU
  // contention rather than on the server.
  //
  // CI is being raised incrementally and measured on the actual 4-vCPU hosted
  // runner — the workstation numbers above don't transfer, since the four
  // single-threaded static-server processes the CI value used to assume
  // became one single-threaded process. Placeholder during measurement.
  workers: process.env['CI'] ? 4 : 8,
  use: {
    baseURL,
    trace: 'on-first-retry',
  },
  ...(useDeployedEnvironment
    ? {}
    : {
        // One process serving the four build outputs on the four origins the
        // suite addresses, rather than a static-server task per app.
        //
        // Running them as Nx tasks put two failure modes on every run. The
        // executor resolves a *free* port instead of the configured one, so a
        // lingering listener silently moved a remote to a neighbouring port and
        // the tests addressed an origin nothing served. And the tasks are not
        // continuous, so Nx reported them successful and tore the servers down
        // partway through the suite — measured as three of the four exiting
        // ~40s in, with every test after that failing to connect.
        //
        // Serving the built output rather than a `serve-static`-style target is
        // deliberate: that kicks off its own build, and the Angular builder
        // empties the output directory before rewriting it. The server is
        // already accepting connections by then, so the readiness probe passes
        // against the *previous* build and tests run against a directory being
        // deleted underneath them — measured as index.html vanishing for ~15s
        // mid-run. The e2e target already depends on its dependencies' builds,
        // so the output is present and current before any of this starts.
        webServer: {
          command: 'node apps/e2e/platform-e2e/static-server.mjs',
          // The host is bound last, so this one probe covers all four origins.
          url: 'http://localhost:4200',
          reuseExistingServer: !process.env['CI'],
          cwd: workspaceRoot,
          timeout: 180_000,
        },
      }),
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
