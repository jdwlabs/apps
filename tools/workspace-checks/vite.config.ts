/// <reference types="vitest" />
import { defineConfig } from 'vite';
import tsconfigPaths from 'vite-tsconfig-paths';

export default defineConfig({
  plugins: [tsconfigPaths()],
  test: {
    globals: true,
    environment: 'node',
    include: ['**/*.spec.ts'],
    reporters: ['default'],
    passWithNoTests: true,
    // Building the project graph walks the whole workspace and can shell out to
    // plugins; the default 5s is not enough on a cold daemon.
    testTimeout: 120_000,
    coverage: {
      reportsDirectory: '../../coverage/tools/workspace-checks',
    },
  },
});
