/**
 * Shared Constants Tests
 *
 * Trace:
 *   spec_id: SPEC-UTILS-001
 *   task_id: TASK-010
 */

import { describe, expect, it } from 'vitest';

import {
  CHARGE_AMOUNT,
  COST_PER_GAME,
  GAMES_PER_PURCHASE,
  MIN_DEPOSIT_AMOUNT,
  TOTAL_PURCHASE_COST,
} from './constants';

describe('TEST-UTILS-002: Shared business constants', () => {
  it('should expose required constants with expected values', () => {
    expect(MIN_DEPOSIT_AMOUNT).toBe(5000);
    expect(CHARGE_AMOUNT).toBe(50000);
    expect(GAMES_PER_PURCHASE).toBe(5);
    expect(COST_PER_GAME).toBe(1000);
    expect(TOTAL_PURCHASE_COST).toBe(5000);
  });
});

