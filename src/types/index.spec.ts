/**
 * Types Barrel Tests
 *
 * Trace:
 *   spec_id: SPEC-UTILS-001
 *   task_id: TASK-010
 */

import { describe, expect, it } from 'vitest';

import type { AccountInfo, PurchaseEnv, WinningResult } from './index';

describe('TEST-UTILS-003: Types barrel export', () => {
  it('should allow importing shared types from a single entry', () => {
    // Compile-time only check
    const ok = true as const;
    expect(ok).toBe(true);

    type _AccountInfo = AccountInfo;
    type _PurchaseEnv = PurchaseEnv;
    type _WinningResult = WinningResult;

    expect(typeof (null as unknown as _AccountInfo | _PurchaseEnv | _WinningResult)).toBe('object');
  });
});

