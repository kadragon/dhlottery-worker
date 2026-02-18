/**
 * Main Orchestration Test Suite
 *
 * Trace:
 *   spec_id: SPEC-ORCH-001
 *   task_id: TASK-008, TASK-011
 */

import { describe, it, expect, beforeEach, vi, afterEach, type Mock } from "vitest";
import type { HttpClient, PurchaseOutcome } from "./types";

vi.mock("./client/http", () => ({
  createHttpClient: vi.fn(),
}));

vi.mock("./dhlottery/auth", () => ({
  login: vi.fn(),
}));

vi.mock("./dhlottery/charge", () => ({
  checkDeposit: vi.fn(),
}));

vi.mock("./dhlottery/buy", () => ({
  purchaseLottery: vi.fn(),
}));

vi.mock("./dhlottery/pension-reserve", () => ({
  reservePensionNextWeek: vi.fn(),
}));

vi.mock("./dhlottery/check", () => ({
  checkWinning: vi.fn(),
}));

vi.mock("./notify/telegram", () => ({
  sendNotification: vi.fn().mockResolvedValue(undefined),
}));

const { createHttpClient } = await import("./client/http");
const { login } = await import("./dhlottery/auth");
const { checkDeposit } = await import("./dhlottery/charge");
const { purchaseLottery } = await import("./dhlottery/buy");
const { reservePensionNextWeek } = await import("./dhlottery/pension-reserve");
const { checkWinning } = await import("./dhlottery/check");
const { sendNotification } = await import("./notify/telegram");

describe("Main Orchestration - runWorkflow", () => {
  let mockClient: HttpClient;

  beforeEach(() => {
    // Mock process.env
    vi.stubEnv('USER_ID', 'test-user');
    vi.stubEnv('PASSWORD', 'test-pass');
    vi.stubEnv('TELEGRAM_BOT_TOKEN', 'test-token');
    vi.stubEnv('TELEGRAM_CHAT_ID', 'test-chat');

    mockClient = {
      cookies: {},
      fetch: vi.fn(),
      getCookieHeader: vi.fn(),
      clearCookies: vi.fn(),
    } as unknown as HttpClient;

    (createHttpClient as Mock).mockReturnValue(mockClient);
    (login as Mock).mockResolvedValue(undefined);
    (checkDeposit as Mock).mockResolvedValue(true);
    (reservePensionNextWeek as Mock).mockResolvedValue({
      status: "success",
      success: true,
      skipped: false,
    });
    (purchaseLottery as Mock).mockResolvedValue({ success: true } as PurchaseOutcome);
    (checkWinning as Mock).mockResolvedValue([]);
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it("TEST-ORCH-001: should run the complete workflow", async () => {
    const { runWorkflow } = await import("./index");

    await runWorkflow(new Date("2025-12-15T00:00:00.000Z"));

    expect(createHttpClient).toHaveBeenCalledTimes(1);
    expect(login).toHaveBeenCalledTimes(1);
    expect(checkDeposit).toHaveBeenCalledWith(expect.anything(), 10000);
    expect(reservePensionNextWeek).toHaveBeenCalledTimes(1);
    expect(purchaseLottery).toHaveBeenCalledTimes(1);
    expect(checkWinning).toHaveBeenCalledTimes(1);
  });

  it("TEST-ORCH-002: should skip purchase when deposit is insufficient", async () => {
    (checkDeposit as Mock).mockResolvedValue(false);

    const { runWorkflow } = await import("./index");

    await runWorkflow(new Date("2025-12-15T00:00:00.000Z"));

    expect(login).toHaveBeenCalledTimes(1);
    expect(checkDeposit).toHaveBeenCalledTimes(1);
    expect(reservePensionNextWeek).not.toHaveBeenCalled();
    expect(purchaseLottery).not.toHaveBeenCalled();
    expect(checkWinning).toHaveBeenCalledTimes(1);
  });

  it("TEST-ORCH-003: should notify and stop on fatal precondition errors", async () => {
    (login as Mock).mockRejectedValue(new Error("boom"));

    const { runWorkflow } = await import("./index");

    await runWorkflow(new Date("2025-12-15T00:00:00.000Z"));

    expect(sendNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        type: "error",
        title: expect.stringContaining("Orchestration"),
      }),
    );
    expect(checkDeposit).not.toHaveBeenCalled();
    expect(purchaseLottery).not.toHaveBeenCalled();
  });

  it("should notify and return early when checkDeposit throws", async () => {
    (checkDeposit as Mock).mockRejectedValue(new Error("deposit failed"));

    const { runWorkflow } = await import("./index");

    await expect(runWorkflow(new Date("2025-12-15T00:00:00.000Z"))).resolves.toBeUndefined();

    expect(sendNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        type: "error",
        title: "Orchestration Error",
        message: expect.stringContaining("deposit failed"),
      }),
    );
    expect(purchaseLottery).not.toHaveBeenCalled();
    expect(checkWinning).not.toHaveBeenCalled();
  });

  it("should continue to buy when pension reserve returns failure outcome", async () => {
    (reservePensionNextWeek as Mock).mockResolvedValue({
      status: "failure",
      success: false,
      skipped: false,
      error: "reserve failed",
    });

    const { runWorkflow } = await import("./index");

    await expect(runWorkflow(new Date("2025-12-15T00:00:00.000Z"))).resolves.toBeUndefined();

    expect(reservePensionNextWeek).toHaveBeenCalledTimes(1);
    expect(purchaseLottery).toHaveBeenCalledTimes(1);
    expect(checkWinning).toHaveBeenCalledTimes(1);
  });

  it("should notify when buy throws and not continue to checkWinning", async () => {
    (purchaseLottery as Mock).mockRejectedValue(new Error("buy failed"));

    const { runWorkflow } = await import("./index");

    await expect(runWorkflow(new Date("2025-12-15T00:00:00.000Z"))).resolves.toBeUndefined();

    expect(sendNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        type: "error",
        title: "Orchestration Error",
        message: expect.stringContaining("buy failed"),
      }),
    );
    expect(checkWinning).not.toHaveBeenCalled();
  });

  it("should notify when checkWinning throws", async () => {
    (checkWinning as Mock).mockRejectedValue(new Error("winning failed"));

    const { runWorkflow } = await import("./index");

    await expect(runWorkflow(new Date("2025-12-15T00:00:00.000Z"))).resolves.toBeUndefined();

    expect(sendNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        type: "error",
        title: "Orchestration Error",
        message: expect.stringContaining("winning failed"),
      }),
    );
  });

  it("should stringify non-Error failures in orchestration notifications", async () => {
    (login as Mock).mockRejectedValue("string failure");

    const { runWorkflow } = await import("./index");

    await expect(runWorkflow(new Date("2025-12-15T00:00:00.000Z"))).resolves.toBeUndefined();

    expect(sendNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        type: "error",
        title: "Orchestration Error",
        message: expect.stringContaining("string failure"),
      }),
    );
  });
});
