/**
 * GitHub Actions Entry Point
 *
 * Trace:
 *   spec_id: SPEC-GHACTION-001
 *   task_id: TASK-GHACTION-001
 */

import { runWorkflow } from './index';
import { validateEnv } from './utils/env';
import { logger } from './utils/logger';

/**
 * Main entry point for GitHub Actions workflow
 * Validates environment variables and runs the lottery workflow
 */
async function main(): Promise<void> {
  try {
    // Validate all required environment variables
    validateEnv();

    logger.info('Starting lottery workflow');

    // Run the workflow
    await runWorkflow();

    logger.info('Lottery workflow completed successfully');

    process.exit(0);
  } catch (error) {
    logger.error('Lottery workflow failed', {
      error: error instanceof Error ? error.message : String(error),
    });

    process.exit(1);
  }
}

// Run main function
main();
