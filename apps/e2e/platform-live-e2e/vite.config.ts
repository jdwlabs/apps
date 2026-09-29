import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    environment: 'node',
    include: ['src/harness/**/*.spec.ts', 'scripts/**/*.spec.ts'],
    reporters: ['default'],
  },
});
