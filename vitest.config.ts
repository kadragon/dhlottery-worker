import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    /**
     * Trace:
     *   spec_id: SPEC-LOGGING-001
     *   task_id: TASK-012
     */
    globals: true,
    environment: 'node',
    include: ['src/**/*.test.ts'],
    silent: true,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: ['node_modules/', 'src/**/*.test.ts', 'src/__fixtures__/', 'src/types/'],
    },
  },
  resolve: {
    alias: {
      '@': '/src',
    },
  },
});
