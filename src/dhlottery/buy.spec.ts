/**
 * Lottery Purchase Test Suite
 *
 * Trace:
 *   spec_id: SPEC-PURCHASE-001
 *   task_id: TASK-005, TASK-011
 *
 * Tests lottery purchase functionality following TDD RED-GREEN-REFACTOR cycle
 */

import { describe, it, expect, vi, beforeEach, type Mock } from "vitest";
import { purchaseLottery } from "./buy";
import type {
	PurchaseEnv,
	PurchaseReadyResponse,
	PurchaseResult,
  AccountInfo,
  HttpClient,
} from "../types";
import { PURCHASE_CONSTANTS } from "../types/purchase.types";

// Mock dependencies
vi.mock("../notify/telegram", () => ({
	sendNotification: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("./account", () => ({
	getAccountInfo: vi.fn(),
}));

// Import mocked functions after vi.mock
const { sendNotification } = await import("../notify/telegram");
const { getAccountInfo } = await import("./account");

describe("Lottery Purchase - TEST-PURCHASE-001: Purchase ready endpoint", () => {
	let mockEnv: PurchaseEnv;
	let mockFetch: Mock;

	beforeEach(() => {
		mockEnv = {
			DHLOTTERY_USER_ID: "testuser",
			DHLOTTERY_USER_PW: "testpass",
			TELEGRAM_BOT_TOKEN: "test-token",
			TELEGRAM_CHAT_ID: "test-chat",
		};

		mockFetch = vi.fn();
		global.fetch = mockFetch;
		vi.clearAllMocks();
	});

	it("should call purchase ready endpoint before purchase execution", async () => {
		const mockAccountInfo: AccountInfo = {
			balance: 50000,
			currentRound: 1203,
		};
		(getAccountInfo as Mock).mockResolvedValue(mockAccountInfo);

		const mockReadyResponse: PurchaseReadyResponse = {
			direct_yn: "N",
			ready_ip: "INTCOM2",
			ready_time: "0",
			ready_cnt: "0",
		};

		const mockPurchaseResult: PurchaseResult = {
			loginYn: "Y",
			result: {
				resultCode: "100",
				resultMsg: "Success",
			},
		};

		mockFetch
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockReadyResponse,
			} as Response)
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockPurchaseResult,
			} as Response);

		await purchaseLottery({} as HttpClient, mockEnv);

		expect(mockFetch).toHaveBeenCalledWith(
			"https://ol.dhlottery.co.kr/olotto/game/egovUserReadySocket.json",
			expect.objectContaining({
				method: "POST",
				headers: expect.objectContaining({
					"Content-Type": "application/json; charset=UTF-8",
				}),
			}),
		);
	});

	it("should succeed when ready endpoint returns valid response", async () => {
		const mockAccountInfo: AccountInfo = {
			balance: 50000,
			currentRound: 1203,
		};
		(getAccountInfo as Mock).mockResolvedValue(mockAccountInfo);

		const mockReadyResponse: PurchaseReadyResponse = {
			direct_yn: "N",
			ready_ip: "INTCOM2",
			ready_time: "0",
			ready_cnt: "0",
		};

		const mockPurchaseResult: PurchaseResult = {
			loginYn: "Y",
			result: {
				resultCode: "100",
				resultMsg: "Success",
			},
		};

		mockFetch
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockReadyResponse,
			} as Response)
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockPurchaseResult,
			} as Response);

		const result = await purchaseLottery({} as HttpClient, mockEnv);

		expect(result.success).toBe(true);
	});

	it("should use ready_ip value in purchase execution", async () => {
		const mockAccountInfo: AccountInfo = {
			balance: 50000,
			currentRound: 1203,
		};
		(getAccountInfo as Mock).mockResolvedValue(mockAccountInfo);

		const mockReadyResponse: PurchaseReadyResponse = {
			direct_yn: "N",
			ready_ip: "TESTSERVER",
			ready_time: "0",
			ready_cnt: "0",
		};

		const mockPurchaseResult: PurchaseResult = {
			loginYn: "Y",
			result: {
				resultCode: "100",
				resultMsg: "Success",
			},
		};

		mockFetch
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockReadyResponse,
			} as Response)
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockPurchaseResult,
			} as Response);

		await purchaseLottery({} as HttpClient, mockEnv);

		const execBuyCall = mockFetch.mock.calls.find((call) =>
			call[0].includes("execBuy.do"),
		);
		expect(execBuyCall).toBeDefined();
		expect(execBuyCall![1].body).toContain("direct=TESTSERVER");
	});
});

describe("Lottery Purchase - TEST-PURCHASE-002: Purchase execution with correct parameters", () => {
	let mockEnv: PurchaseEnv;
	let mockFetch: Mock;

	beforeEach(() => {
		mockEnv = {
			DHLOTTERY_USER_ID: "testuser",
			DHLOTTERY_USER_PW: "testpass",
			TELEGRAM_BOT_TOKEN: "test-token",
			TELEGRAM_CHAT_ID: "test-chat",
		};

		mockFetch = vi.fn();
		global.fetch = mockFetch;
		vi.clearAllMocks();
	});

	it("should execute purchase with 5 games", async () => {
		const mockAccountInfo: AccountInfo = {
			balance: 50000,
			currentRound: 1203,
		};
		(getAccountInfo as Mock).mockResolvedValue(mockAccountInfo);

		const mockReadyResponse: PurchaseReadyResponse = {
			direct_yn: "N",
			ready_ip: "INTCOM2",
			ready_time: "0",
			ready_cnt: "0",
		};

		const mockPurchaseResult: PurchaseResult = {
			loginYn: "Y",
			result: {
				resultCode: "100",
				resultMsg: "Success",
			},
		};

		mockFetch
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockReadyResponse,
			} as Response)
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockPurchaseResult,
			} as Response);

		await purchaseLottery({} as HttpClient, mockEnv);

		const execBuyCall = mockFetch.mock.calls.find((call) =>
			call[0].includes("execBuy.do"),
		);
		expect(execBuyCall).toBeDefined();
		expect(execBuyCall![1].body).toContain("gameCnt=5");
	});

	it("should use automatic number generation mode", async () => {
		const mockAccountInfo: AccountInfo = {
			balance: 50000,
			currentRound: 1203,
		};
		(getAccountInfo as Mock).mockResolvedValue(mockAccountInfo);

		const mockReadyResponse: PurchaseReadyResponse = {
			direct_yn: "N",
			ready_ip: "INTCOM2",
			ready_time: "0",
			ready_cnt: "0",
		};

		const mockPurchaseResult: PurchaseResult = {
			loginYn: "Y",
			result: {
				resultCode: "100",
				resultMsg: "Success",
			},
		};

		mockFetch
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockReadyResponse,
			} as Response)
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockPurchaseResult,
			} as Response);

		await purchaseLottery({} as HttpClient, mockEnv);

		const execBuyCall = mockFetch.mock.calls.find((call) =>
			call[0].includes("execBuy.do"),
		);
		expect(execBuyCall).toBeDefined();
		// genType "0" means automatic mode, arrGameChoiceNum null
		// Decode URL-encoded body to check JSON content
		const decodedBody = decodeURIComponent(execBuyCall![1].body);
		expect(decodedBody).toContain('"genType":"0"');
		expect(decodedBody).toContain('"arrGameChoiceNum":null');
	});

	it("should set total amount to 5,000 KRW", async () => {
		const mockAccountInfo: AccountInfo = {
			balance: 50000,
			currentRound: 1203,
		};
		(getAccountInfo as Mock).mockResolvedValue(mockAccountInfo);

		const mockReadyResponse: PurchaseReadyResponse = {
			direct_yn: "N",
			ready_ip: "INTCOM2",
			ready_time: "0",
			ready_cnt: "0",
		};

		const mockPurchaseResult: PurchaseResult = {
			loginYn: "Y",
			result: {
				resultCode: "100",
				resultMsg: "Success",
			},
		};

		mockFetch
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockReadyResponse,
			} as Response)
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockPurchaseResult,
			} as Response);

		await purchaseLottery({} as HttpClient, mockEnv);

		const execBuyCall = mockFetch.mock.calls.find((call) =>
			call[0].includes("execBuy.do"),
		);
		expect(execBuyCall).toBeDefined();
		expect(execBuyCall![1].body).toContain("nBuyAmount=5000");
	});
});

describe("Lottery Purchase - TEST-PURCHASE-003: Parse purchase result", () => {
	let mockEnv: PurchaseEnv;
	let mockFetch: Mock;

	beforeEach(() => {
		mockEnv = {
			DHLOTTERY_USER_ID: "testuser",
			DHLOTTERY_USER_PW: "testpass",
			TELEGRAM_BOT_TOKEN: "test-token",
			TELEGRAM_CHAT_ID: "test-chat",
		};

		mockFetch = vi.fn();
		global.fetch = mockFetch;
		vi.clearAllMocks();
	});

	it("should recognize successful purchase with result code 100", async () => {
		const mockAccountInfo: AccountInfo = {
			balance: 50000,
			currentRound: 1203,
		};
		(getAccountInfo as Mock).mockResolvedValue(mockAccountInfo);

		const mockReadyResponse: PurchaseReadyResponse = {
			direct_yn: "N",
			ready_ip: "INTCOM2",
			ready_time: "0",
			ready_cnt: "0",
		};

		const mockPurchaseResult: PurchaseResult = {
			loginYn: "Y",
			result: {
				resultCode: "100",
				resultMsg: "Success",
			},
		};

		mockFetch
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockReadyResponse,
			} as Response)
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockPurchaseResult,
			} as Response);

		const result = await purchaseLottery({} as HttpClient, mockEnv);

		expect(result.success).toBe(true);
		if (result.success) {
			expect(result.gameCount).toBe(PURCHASE_CONSTANTS.GAME_COUNT);
			expect(result.totalAmount).toBe(PURCHASE_CONSTANTS.TOTAL_COST);
			expect(result.roundNumber).toBe(1203);
		}
	});

	it("should include purchase details in successful result", async () => {
		const mockAccountInfo: AccountInfo = {
			balance: 50000,
			currentRound: 1203,
		};
		(getAccountInfo as Mock).mockResolvedValue(mockAccountInfo);

		const mockReadyResponse: PurchaseReadyResponse = {
			direct_yn: "N",
			ready_ip: "INTCOM2",
			ready_time: "0",
			ready_cnt: "0",
		};

		const mockPurchaseResult: PurchaseResult = {
			loginYn: "Y",
			result: {
				resultCode: "100",
				resultMsg: "Purchase successful",
			},
		};

		mockFetch
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockReadyResponse,
			} as Response)
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockPurchaseResult,
			} as Response);

		const result = await purchaseLottery({} as HttpClient, mockEnv);

		expect(result.success).toBe(true);
		if (result.success) {
			expect(result.message).toBeDefined();
			expect(result.purchaseDate).toBeDefined();
		}
	});
});

describe("Lottery Purchase - TEST-PURCHASE-004: Telegram success notification", () => {
	let mockEnv: PurchaseEnv;
	let mockFetch: Mock;

	beforeEach(() => {
		mockEnv = {
			DHLOTTERY_USER_ID: "testuser",
			DHLOTTERY_USER_PW: "testpass",
			TELEGRAM_BOT_TOKEN: "test-token",
			TELEGRAM_CHAT_ID: "test-chat",
		};

		mockFetch = vi.fn();
		global.fetch = mockFetch;
		vi.clearAllMocks();
	});

	it("should send success notification via Telegram on successful purchase", async () => {
		const mockAccountInfo: AccountInfo = {
			balance: 50000,
			currentRound: 1203,
		};
		(getAccountInfo as Mock).mockResolvedValue(mockAccountInfo);

		const mockReadyResponse: PurchaseReadyResponse = {
			direct_yn: "N",
			ready_ip: "INTCOM2",
			ready_time: "0",
			ready_cnt: "0",
		};

		const mockPurchaseResult: PurchaseResult = {
			loginYn: "Y",
			result: {
				resultCode: "100",
				resultMsg: "Success",
			},
		};

		mockFetch
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockReadyResponse,
			} as Response)
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockPurchaseResult,
			} as Response);

		await purchaseLottery({} as HttpClient, mockEnv);

		expect(sendNotification).toHaveBeenCalledWith(
			expect.objectContaining({
				type: "success",
				title: expect.stringContaining("구매 완료"),
			}),
			mockEnv,
		);
	});

	it("should include game count in notification", async () => {
		const mockAccountInfo: AccountInfo = {
			balance: 50000,
			currentRound: 1203,
		};
		(getAccountInfo as Mock).mockResolvedValue(mockAccountInfo);

		const mockReadyResponse: PurchaseReadyResponse = {
			direct_yn: "N",
			ready_ip: "INTCOM2",
			ready_time: "0",
			ready_cnt: "0",
		};

		const mockPurchaseResult: PurchaseResult = {
			loginYn: "Y",
			result: {
				resultCode: "100",
				resultMsg: "Success",
			},
		};

		mockFetch
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockReadyResponse,
			} as Response)
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockPurchaseResult,
			} as Response);

		await purchaseLottery({} as HttpClient, mockEnv);

		const notificationCall = (sendNotification as Mock).mock.calls[0];
		const message = notificationCall[0].message;
		expect(message).toContain("5게임");
	});

	it("should include total cost in notification", async () => {
		const mockAccountInfo: AccountInfo = {
			balance: 50000,
			currentRound: 1203,
		};
		(getAccountInfo as Mock).mockResolvedValue(mockAccountInfo);

		const mockReadyResponse: PurchaseReadyResponse = {
			direct_yn: "N",
			ready_ip: "INTCOM2",
			ready_time: "0",
			ready_cnt: "0",
		};

		const mockPurchaseResult: PurchaseResult = {
			loginYn: "Y",
			result: {
				resultCode: "100",
				resultMsg: "Success",
			},
		};

		mockFetch
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockReadyResponse,
			} as Response)
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockPurchaseResult,
			} as Response);

		await purchaseLottery({} as HttpClient, mockEnv);

		const notificationCall = (sendNotification as Mock).mock.calls[0];
		const message = notificationCall[0].message;
		expect(message).toContain("5,000원");
	});

	it("should include lottery round number in notification", async () => {
		const mockAccountInfo: AccountInfo = {
			balance: 50000,
			currentRound: 1203,
		};
		(getAccountInfo as Mock).mockResolvedValue(mockAccountInfo);

		const mockReadyResponse: PurchaseReadyResponse = {
			direct_yn: "N",
			ready_ip: "INTCOM2",
			ready_time: "0",
			ready_cnt: "0",
		};

		const mockPurchaseResult: PurchaseResult = {
			loginYn: "Y",
			result: {
				resultCode: "100",
				resultMsg: "Success",
			},
		};

		mockFetch
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockReadyResponse,
			} as Response)
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockPurchaseResult,
			} as Response);

		await purchaseLottery({} as HttpClient, mockEnv);

		const notificationCall = (sendNotification as Mock).mock.calls[0];
		const message = notificationCall[0].message;
		expect(message).toContain("1203회");
	});
});

describe("Lottery Purchase - TEST-PURCHASE-005: Handle purchase failures", () => {
	let mockEnv: PurchaseEnv;
	let mockFetch: Mock;

	beforeEach(() => {
		mockEnv = {
			DHLOTTERY_USER_ID: "testuser",
			DHLOTTERY_USER_PW: "testpass",
			TELEGRAM_BOT_TOKEN: "test-token",
			TELEGRAM_CHAT_ID: "test-chat",
		};

		mockFetch = vi.fn();
		global.fetch = mockFetch;
		vi.clearAllMocks();
	});

	it("should handle network errors during ready endpoint call", async () => {
		const mockAccountInfo: AccountInfo = {
			balance: 50000,
			currentRound: 1203,
		};
		(getAccountInfo as Mock).mockResolvedValue(mockAccountInfo);

		mockFetch.mockRejectedValueOnce(new Error("Network error"));

		const result = await purchaseLottery({} as HttpClient, mockEnv);

		expect(result.success).toBe(false);
		if (!result.success) {
			expect(result.error).toContain("Network error");
		}
	});

	it("should handle network errors during purchase execution", async () => {
		const mockAccountInfo: AccountInfo = {
			balance: 50000,
			currentRound: 1203,
		};
		(getAccountInfo as Mock).mockResolvedValue(mockAccountInfo);

		const mockReadyResponse: PurchaseReadyResponse = {
			direct_yn: "N",
			ready_ip: "INTCOM2",
			ready_time: "0",
			ready_cnt: "0",
		};

		mockFetch
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockReadyResponse,
			} as Response)
			.mockRejectedValueOnce(new Error("Network error"));

		const result = await purchaseLottery({} as HttpClient, mockEnv);

		expect(result.success).toBe(false);
		if (!result.success) {
			expect(result.error).toContain("Network error");
		}
	});

	it("should handle purchase limit exceeded error", async () => {
		const mockAccountInfo: AccountInfo = {
			balance: 50000,
			currentRound: 1203,
		};
		(getAccountInfo as Mock).mockResolvedValue(mockAccountInfo);

		const mockReadyResponse: PurchaseReadyResponse = {
			direct_yn: "N",
			ready_ip: "INTCOM2",
			ready_time: "0",
			ready_cnt: "0",
		};

		const mockPurchaseResult: PurchaseResult = {
			loginYn: "Y",
			result: {
				resultCode: "-7",
				resultMsg: "[온라인복권 주간 구매한도] 초과되었습니다.",
			},
		};

		mockFetch
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockReadyResponse,
			} as Response)
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockPurchaseResult,
			} as Response);

		const result = await purchaseLottery({} as HttpClient, mockEnv);

		expect(result.success).toBe(false);
		if (!result.success) {
			expect(result.code).toBe("-7");
			expect(result.error).toContain("구매한도");
		}
	});

	it("should send error notification via Telegram on failure", async () => {
		const mockAccountInfo: AccountInfo = {
			balance: 50000,
			currentRound: 1203,
		};
		(getAccountInfo as Mock).mockResolvedValue(mockAccountInfo);

		mockFetch.mockRejectedValueOnce(new Error("Network error"));

		await purchaseLottery({} as HttpClient, mockEnv);

		expect(sendNotification).toHaveBeenCalledWith(
			expect.objectContaining({
				type: "error",
				title: expect.stringContaining("구매 실패"),
			}),
			mockEnv,
		);
	});

	it("should not make partial purchases (atomic transaction)", async () => {
		const mockAccountInfo: AccountInfo = {
			balance: 50000,
			currentRound: 1203,
		};
		(getAccountInfo as Mock).mockResolvedValue(mockAccountInfo);

		const mockReadyResponse: PurchaseReadyResponse = {
			direct_yn: "N",
			ready_ip: "INTCOM2",
			ready_time: "0",
			ready_cnt: "0",
		};

		mockFetch
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockReadyResponse,
			} as Response)
			.mockRejectedValueOnce(new Error("Network error"));

		const result = await purchaseLottery({} as HttpClient, mockEnv);

		// Purchase should fail completely, not partially
		expect(result.success).toBe(false);
		// Only 2 fetch calls (ready + failed execution attempt)
		expect(mockFetch).toHaveBeenCalledTimes(2);
	});
});
