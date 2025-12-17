/**
 * Main Orchestration Test Suite
 *
 * Trace:
 *   spec_id: SPEC-ORCH-001
 *   task_id: TASK-008
 */

import { describe, it, expect, beforeEach, vi, type Mock } from "vitest";
import type { ExecutionContext, ScheduledController } from "@cloudflare/workers-types";
import type { HttpClient } from "./types/http.types";
import type { PurchaseOutcome } from "./types/purchase.types";

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
const { checkWinning } = await import("./dhlottery/check");
const { sendNotification } = await import("./notify/telegram");

type WorkerEnv = {
  USER_ID: string;
  PASSWORD: string;
  TELEGRAM_BOT_TOKEN: string;
  TELEGRAM_CHAT_ID: string;
};

describe("Main Orchestration - scheduled handler", () => {
  let mockClient: HttpClient;
  let env: WorkerEnv;

  beforeEach(() => {
    mockClient = {
      cookies: {},
      fetch: vi.fn(),
      getCookieHeader: vi.fn(),
      clearCookies: vi.fn(),
    } as unknown as HttpClient;

    env = {
      USER_ID: "test-user",
      PASSWORD: "test-pass",
      TELEGRAM_BOT_TOKEN: "test-token",
      TELEGRAM_CHAT_ID: "test-chat",
    };

    (createHttpClient as Mock).mockReturnValue(mockClient);
    (login as Mock).mockResolvedValue(undefined);
    (checkDeposit as Mock).mockResolvedValue(true);
    (purchaseLottery as Mock).mockResolvedValue({ success: true } as PurchaseOutcome);
    (checkWinning as Mock).mockResolvedValue([]);
    vi.clearAllMocks();
  });

  it("TEST-ORCH-001: should run the workflow from scheduled handler", async () => {
    const { default: worker } = await import("./index");

    const ctx = {
      waitUntil: vi.fn(),
    };

    await worker.scheduled({} as ScheduledController, env, ctx as unknown as ExecutionContext);

    expect(ctx.waitUntil).toHaveBeenCalledTimes(1);
    const promise = ctx.waitUntil.mock.calls[0][0] as Promise<void>;
    await promise;

    expect(createHttpClient).toHaveBeenCalledTimes(1);
    expect(login).toHaveBeenCalledTimes(1);
    expect(checkDeposit).toHaveBeenCalledTimes(1);
    expect(purchaseLottery).toHaveBeenCalledTimes(1);
    expect(checkWinning).toHaveBeenCalledTimes(1);
  });

  it("TEST-ORCH-002: should skip purchase when deposit is insufficient", async () => {
    (checkDeposit as Mock).mockResolvedValue(false);

    const { runWorkflow } = await import("./index");

    await runWorkflow(env, new Date("2025-12-15T00:00:00.000Z"));

    expect(login).toHaveBeenCalledTimes(1);
    expect(checkDeposit).toHaveBeenCalledTimes(1);
    expect(purchaseLottery).not.toHaveBeenCalled();
    expect(checkWinning).toHaveBeenCalledTimes(1);
  });

  it("TEST-ORCH-003: should notify and stop on fatal precondition errors", async () => {
    (login as Mock).mockRejectedValue(new Error("boom"));

    const { runWorkflow } = await import("./index");

    await runWorkflow(env, new Date("2025-12-15T00:00:00.000Z"));

    expect(sendNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        type: "error",
        title: expect.stringContaining("Orchestration"),
      }),
      env
    );
    expect(checkDeposit).not.toHaveBeenCalled();
    expect(purchaseLottery).not.toHaveBeenCalled();
  });
});
