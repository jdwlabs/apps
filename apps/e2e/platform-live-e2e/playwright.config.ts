import { defineConfig } from '@playwright/test';
import { resolveProfile } from './src/harness/profiles';
import { gateFilter } from './src/harness/tags';

const profile = resolveProfile(process.env);
const { grep, grepInvert } = gateFilter(profile);

export default defineConfig({
  testDir: './src/suites',
  testMatch: '**/*.gate.ts',
  fullyParallel: true,
  forbidOnly: true,
  retries: 1,
  workers: 4,
  timeout: 60_000,
  reporter: [
    ['list'],
    ['junit', { outputFile: '../../../dist/platform-live-e2e/junit.xml' }],
    ['json', { outputFile: '../../../dist/platform-live-e2e/results.json' }],
  ],
  outputDir: '../../../dist/platform-live-e2e/test-results',
  use: { trace: 'retain-on-failure' },
  projects: [
    { name: 'api-gate', testMatch: 'api-gate/**/*.gate.ts', grep, grepInvert },
  ],
});
