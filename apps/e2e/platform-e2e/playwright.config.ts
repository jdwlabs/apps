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
  // CI was swept on the actual ubuntu-latest (4-vCPU) hosted runner rather
  // than reusing the workstation numbers, since those speak to a different
  // machine and the four-process layout this replaced would not have
  // transferred either. 1/2/4/6/8 all ran green with no connection resets or
  // navigation timeouts. No failure threshold was reached: fullyParallel is
  // off (Playwright default, unset by both this config and the nx preset),
  // so within a shard tests in the same file run serially and real
  // concurrency against the server is bounded by the file count in that
  // shard — 3, with the suite's current 6 spec files over 2 shards — not by
  // this number. 8 is set to match the proven local value and give the
  // suite headroom to grow before the next file-count/shard-count change
  // makes this worth re-sweeping.
  workers: process.env['CI'] ? 8 : 8,
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
