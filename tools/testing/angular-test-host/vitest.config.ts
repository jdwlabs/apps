import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    // Recorded calls outlive the test that made them by default, so a
    // `toHaveBeenCalledWith` on a shared mock can be satisfied by an earlier
    // test's call and assert nothing about its own subject.
    clearMocks: true,
  },
});
