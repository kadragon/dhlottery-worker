/**
 * GitHub Actions Entry Point
 *
 * Trace:
 *   spec_id: SPEC-GHACTION-001
 *   task_id: TASK-GHACTION-001
 */

import { runWorkflow } from './index';
import { validateEnv } from './utils/env';

/**
 * Log level type
 */
type LogLevel = 'info' | 'error';

/**
 * Structured logging helper for GitHub Actions
 * Outputs JSON-formatted logs with timestamp
 *
 * @param level - Log level (info or error)
 * @param message - Log message
 * @param details - Optional additional details to include in log
 */
function log(level: LogLevel, message: string, details?: Record<string, unknown>): void {
  const logEntry = {
    level,
    message,
    ...details,
    timestamp: new Date().toISOString(),
  };

  const output = JSON.stringify(logEntry);

  if (level === 'error') {
    console.error(output);
  } else {
    console.log(output);
  }
}

/**
 * Main entry point for GitHub Actions workflow
 * Validates environment variables and runs the lottery workflow
 */
async function main(): Promise<void> {
  try {
    // Validate all required environment variables
    validateEnv();

    log('info', 'Starting lottery workflow');

    // Run the workflow
    await runWorkflow();

    log('info', 'Lottery workflow completed successfully');

    process.exit(0);
  } catch (error) {
    log('error', 'Lottery workflow failed', {
      error: error instanceof Error ? error.message : String(error),
    });

    process.exit(1);
  }
}

// Run main function
main();
