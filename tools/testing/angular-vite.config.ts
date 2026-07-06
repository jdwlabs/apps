/// <reference types="vitest" />
import angular from '@analogjs/vite-plugin-angular';
import { defineConfig } from 'vite';
import tsconfigPaths from 'vite-tsconfig-paths';

/**
 * Shared Vitest config for Angular projects. Each project's own
 * `vite.config.ts` is a thin wrapper that supplies its coverage output dir --
 * kept per-project (not hoisted to workspace root) because the Nx executor
 * (`@analogjs/vitest-angular:test`) resolves `configFile` relative to the
 * project root, so every project needs its own config entry point.
 */
export function createAngularViteConfig(coverageDirectory: string) {
  return defineConfig({
    plugins: [tsconfigPaths(), angular()],
    test: {
      globals: true,
      environment: 'jsdom',
      setupFiles: ['src/test-setup.ts'],
      include: ['src/**/*.spec.ts'],
      reporters: ['default'],
      // Some util libs have no spec files yet (mirrors Jest's
      // passWithNoTests, which the prior jest.config.ts relied on).
      passWithNoTests: true,
      coverage: {
        reportsDirectory: coverageDirectory,
      },
    },
  });
}
