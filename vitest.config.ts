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
      include: ['src/**/*.ts'],
      reporter: ['text', 'json', 'html', 'lcov'],
      exclude: ['node_modules/', 'src/**/*.test.ts', 'src/__fixtures__/', 'src/types/'],
      thresholds: {
        lines: 85,
        functions: 85,
        branches: 75,
        statements: 85,
      },
    },
  },
  resolve: {
    alias: {
      '@': '/src',
    },
  },
});
