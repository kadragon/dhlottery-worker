import { afterEach, beforeEach, describe, expect, it, type Mock, vi } from 'vitest';

vi.mock('./index', () => ({
  runWorkflow: vi.fn(),
}));

vi.mock('./utils/env', () => ({
  validateEnv: vi.fn(),
}));

vi.mock('./utils/logger', () => ({
  logger: {
    info: vi.fn(),
    error: vi.fn(),
  },
}));

describe('GitHub Actions entrypoint - run.ts', () => {
  let exitSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
    exitSpy = vi.spyOn(process, 'exit').mockImplementation((() => undefined) as never);
  });

  afterEach(() => {
    exitSpy.mockRestore();
  });

  it('exits with code 0 when env validation and workflow both succeed', async () => {
    const { runWorkflow } = await import('./index');
    const { validateEnv } = await import('./utils/env');
    const { logger } = await import('./utils/logger');

    (validateEnv as Mock).mockImplementation(() => undefined);
    (runWorkflow as Mock).mockResolvedValue(undefined);

    await import('./run');

    await vi.waitFor(() => {
      expect(runWorkflow).toHaveBeenCalledTimes(1);
      expect(logger.info).toHaveBeenCalledWith('Starting lottery workflow');
      expect(logger.info).toHaveBeenCalledWith('Lottery workflow completed successfully');
      expect(process.exit).toHaveBeenCalledWith(0);
    });
  });

  it('logs and exits with code 1 when env validation throws', async () => {
    const { validateEnv } = await import('./utils/env');
    const { logger } = await import('./utils/logger');

    (validateEnv as Mock).mockImplementation(() => {
      throw new Error('missing env');
    });

    await import('./run');

    await vi.waitFor(() => {
      expect(logger.error).toHaveBeenCalledWith('Lottery workflow failed', {
        error: 'missing env',
      });
      expect(process.exit).toHaveBeenCalledWith(1);
    });
  });

  it('logs and exits with code 1 when workflow throws', async () => {
    const { runWorkflow } = await import('./index');
    const { validateEnv } = await import('./utils/env');
    const { logger } = await import('./utils/logger');

    (validateEnv as Mock).mockImplementation(() => undefined);
    (runWorkflow as Mock).mockRejectedValue(new Error('workflow failed'));

    await import('./run');

    await vi.waitFor(() => {
      expect(logger.error).toHaveBeenCalledWith('Lottery workflow failed', {
        error: 'workflow failed',
      });
      expect(process.exit).toHaveBeenCalledWith(1);
    });
  });

  it('stringifies non-Error throws in log payload', async () => {
    const { runWorkflow } = await import('./index');
    const { validateEnv } = await import('./utils/env');
    const { logger } = await import('./utils/logger');

    (validateEnv as Mock).mockImplementation(() => undefined);
    (runWorkflow as Mock).mockRejectedValue('fatal string');

    await import('./run');

    await vi.waitFor(() => {
      expect(logger.error).toHaveBeenCalledWith('Lottery workflow failed', {
        error: 'fatal string',
      });
      expect(process.exit).toHaveBeenCalledWith(1);
    });
  });
});
