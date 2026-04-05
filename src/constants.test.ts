/**
 * Shared Constants Tests
 *
 * Trace:
 *   spec_id: SPEC-UTILS-001, SPEC-REFACTOR-P2-LOG-001
 *   task_id: TASK-010, TASK-REFACTOR-P2-002
 */

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import {
  CHARGE_AMOUNT,
  COST_PER_GAME,
  GAMES_PER_PURCHASE,
  MIN_DEPOSIT_AMOUNT,
  PENSION_RESERVE_COST,
  TOTAL_PURCHASE_COST,
  WEEKLY_COMBINED_REQUIRED_BALANCE,
} from './constants';

describe('TEST-UTILS-002: Shared business constants', () => {
  it('should expose required constants with expected values', () => {
    expect(MIN_DEPOSIT_AMOUNT).toBe(5000);
    expect(CHARGE_AMOUNT).toBe(50000);
    expect(GAMES_PER_PURCHASE).toBe(5);
    expect(COST_PER_GAME).toBe(1000);
    expect(TOTAL_PURCHASE_COST).toBe(5000);
    expect(PENSION_RESERVE_COST).toBe(5000);
    expect(WEEKLY_COMBINED_REQUIRED_BALANCE).toBe(10000);
  });
});

describe('TEST-REFACTOR-P2-LOG-001: DEBUG constant for conditional logging', () => {
  beforeEach(() => {
    vi.stubEnv('DEBUG', '');
  });

  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it('should default to false when DEBUG env is not set', async () => {
    const { DEBUG } = await import('./constants');
    expect(typeof DEBUG).toBe('boolean');
    expect(DEBUG).toBe(false);
  });
});
