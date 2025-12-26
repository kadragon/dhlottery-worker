/**
 * GitHub Actions Entry Point
 *
 * Trace:
 *   spec_id: SPEC-GHACTION-001
 *   task_id: TASK-GHACTION-001
 */

import { validateEnv } from './utils/env';
import { runWorkflow } from './index';

/**
 * Main entry point for GitHub Actions workflow
 * Validates environment variables and runs the lottery workflow
 */
async function main(): Promise<void> {
  try {
    // Validate all required environment variables
    validateEnv();

    console.log(
      JSON.stringify({
        level: 'info',
        message: 'Starting lottery workflow',
        timestamp: new Date().toISOString(),
      })
    );

    // Run the workflow
    await runWorkflow();

    console.log(
      JSON.stringify({
        level: 'info',
        message: 'Lottery workflow completed successfully',
        timestamp: new Date().toISOString(),
      })
    );

    process.exit(0);
  } catch (error) {
    console.error(
      JSON.stringify({
        level: 'error',
        message: 'Lottery workflow failed',
        error: error instanceof Error ? error.message : String(error),
        timestamp: new Date().toISOString(),
      })
    );

    process.exit(1);
  }
}

// Run main function
main();
