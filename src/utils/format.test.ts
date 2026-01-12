/**
 * Number formatting utilities tests
 */

import { describe, it, expect } from "vitest";
import { formatKoreanNumber, formatCurrency } from "./format";

describe("format utilities", () => {
  it("should format numbers with ko-KR locale", () => {
    expect(formatKoreanNumber(5000)).toBe("5,000");
  });

  it("should format numbers with won suffix", () => {
    expect(formatCurrency(5000)).toBe("5,000원");
  });
});
