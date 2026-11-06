import { defineConfig } from 'vitest/config';

// `.refs/` holds pinned upstream baselines for diffing. It must never be
// test-discovered: its copies would run twice (vitest matches CLI filters by
// substring) and execute pre-fork code.
export default defineConfig({
  test: {
    exclude: [
      '**/node_modules/**',
      '**/dist/**',
      '.refs/**',
    ],
  },
});
