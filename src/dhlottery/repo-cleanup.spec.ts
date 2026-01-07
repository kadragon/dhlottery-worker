/**
 * Repository Hygiene Tests
 *
 * Trace:
 *   task_id: TASK-REFACTOR-P0-CLEAN-001
 */

import { readdirSync } from "node:fs";
import { describe, expect, it } from "vitest";

describe("Repository hygiene", () => {
  it("TEST-REFACTOR-P0-CLEAN-001: should not contain .bak files in src/dhlottery", () => {
    const entries = readdirSync(__dirname);
    const bakFiles = entries.filter((name) => /\.bak\d*$/.test(name));

    expect(bakFiles).toEqual([]);
  });
});
