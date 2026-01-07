/**
 * Logger utility tests
 */

import { afterEach, describe, expect, it, vi } from "vitest";

describe("logger", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    vi.resetModules();
    vi.unmock("../constants");
  });

  it("should suppress debug logs when DEBUG is false", async () => {
    vi.doMock("../constants", () => ({ DEBUG: false }));
    const consoleLogSpy = vi.spyOn(console, "log").mockImplementation(() => undefined);

    const { logger } = await import("./logger");

    logger.debug("test");

    expect(consoleLogSpy).not.toHaveBeenCalled();
  });

  it("should log info even when DEBUG is false", async () => {
    vi.doMock("../constants", () => ({ DEBUG: false }));
    const consoleLogSpy = vi.spyOn(console, "log").mockImplementation(() => undefined);

    const { logger } = await import("./logger");

    logger.info("test");

    expect(consoleLogSpy).toHaveBeenCalledTimes(1);
  });

  it("should log errors regardless of DEBUG", async () => {
    vi.doMock("../constants", () => ({ DEBUG: false }));
    const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => undefined);

    const { logger } = await import("./logger");

    logger.error("failure");

    expect(consoleErrorSpy).toHaveBeenCalledTimes(1);
  });
});
