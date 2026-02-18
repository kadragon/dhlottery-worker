import { beforeEach, describe, expect, it, vi } from 'vitest';
import { PENSION_RESERVE_COST } from '../constants';
import type { HttpClient } from '../types';
import { DHLotteryError } from '../utils/errors';
import { reservePensionNextWeek } from './pension-reserve';

vi.mock('../notify/telegram', () => ({
  sendNotification: vi.fn().mockResolvedValue(undefined),
}));

vi.mock('./pension-crypto', () => ({
  encryptElQ: vi.fn((plain: string) => `enc:${plain}`),
  decryptElQ: vi.fn((encrypted: string) =>
    encrypted.startsWith('enc:') ? encrypted.slice(4) : encrypted
  ),
}));

const { sendNotification } = await import('../notify/telegram');

function createMockResponse(status: number, jsonData: unknown) {
  return {
    status,
    statusText: status === 200 ? 'OK' : 'ERROR',
    headers: new Headers(),
    text: async () => JSON.stringify(jsonData),
    json: async <T>() => jsonData as T,
  };
}

describe('pension reserve', () => {
  let mockClient: HttpClient;
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchMock = vi.fn();
    mockClient = {
      cookies: {
        JSESSIONID: '12345678901234567890123456789012.tail.of.session==',
      },
      fetch: fetchMock,
      getCookieHeader: vi.fn(),
      clearCookies: vi.fn(),
    } as unknown as HttpClient;
    vi.clearAllMocks();
  });

  it('should reserve next week ticket and send success notification', async () => {
    fetchMock
      .mockResolvedValueOnce(createMockResponse(200, {}))
      .mockResolvedValueOnce(createMockResponse(200, {}))
      .mockResolvedValueOnce(
        createMockResponse(200, { resultCode: '100', resultMsg: 'ok', ROUND: '303', DRAW_DATE: '2026-02-19' })
      )
      .mockResolvedValueOnce(
        createMockResponse(200, { q: 'enc:{"resultCode":"100","resultMsg":"조회 성공","deposit":"10000"}' })
      )
      .mockResolvedValueOnce(
        createMockResponse(200, { q: 'enc:{"resultCode":"100","resultMsg":"예약 조회 성공","doubleRound":[]}' })
      )
      .mockResolvedValueOnce(
        createMockResponse(
          200,
          {
            q: 'enc:{"resultCode":"100","resultMsg":"예약 성공","reserveOrderNo":"A-1","reserveOrderDate":"2026-02-18 17:00:00"}',
          }
        )
      );

    const result = await reservePensionNextWeek(mockClient);

    expect(result).toMatchObject({
      status: 'success',
      success: true,
      skipped: false,
      targetRound: 304,
      totalAmount: PENSION_RESERVE_COST,
    });
    expect(fetchMock).toHaveBeenCalledTimes(6);
    expect(fetchMock.mock.calls[0][0]).toContain('/game/TotalGame.jsp?LottoId=LP72');
    expect(fetchMock.mock.calls[1][0]).toContain('/game/pension720/reserveGame.jsp');
    expect(fetchMock.mock.calls[2][0]).toContain('/roundRemainTime.do');
    expect(fetchMock.mock.calls[3][0]).toContain('/checkDeposit.do');
    expect(fetchMock.mock.calls[4][0]).toContain('/checkMyReserve.do');
    expect(fetchMock.mock.calls[5][0]).toContain('/addMyReserve.do');

    const duplicateBody = fetchMock.mock.calls[4][1]?.body as string;
    const duplicateQ = new URLSearchParams(duplicateBody).get('q');
    expect(duplicateQ).toContain('repeatRoundCnt=1');
    expect(duplicateQ).toContain('nextRound=304');
    expect(duplicateQ).toContain('winDate=');

    const addBody = fetchMock.mock.calls[5][1]?.body as string;
    const addQ = new URLSearchParams(addBody).get('q');
    expect(addQ).toContain('winDate=2026.02.26');

    expect(sendNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'success',
        title: 'Pension Reserve Completed',
      })
    );
  });

  it('should skip when duplicate reserve exists for target round', async () => {
    fetchMock
      .mockResolvedValueOnce(createMockResponse(200, {}))
      .mockResolvedValueOnce(createMockResponse(200, {}))
      .mockResolvedValueOnce(
        createMockResponse(200, { resultCode: '100', resultMsg: 'ok', ROUND: '303', DRAW_DATE: '2026-02-19' })
      )
      .mockResolvedValueOnce(
        createMockResponse(200, { q: 'enc:{"resultCode":"100","resultMsg":"조회 성공","deposit":"10000"}' })
      )
      .mockResolvedValueOnce(
        createMockResponse(
          200,
          { q: 'enc:{"resultCode":"100","resultMsg":"예약 조회 성공","doubleRound":[{"doubleRound":"304","doubleCnt":"5"}]}' }
        )
      );

    const result = await reservePensionNextWeek(mockClient);

    expect(result).toMatchObject({
      status: 'skipped',
      success: true,
      skipped: true,
      targetRound: 304,
    });
    expect(fetchMock).toHaveBeenCalledTimes(5);
    expect(sendNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'warning',
        title: 'Pension Reserve Skipped',
      })
    );
  });

  it('should return failure and warning when deposit is insufficient', async () => {
    fetchMock
      .mockResolvedValueOnce(createMockResponse(200, {}))
      .mockResolvedValueOnce(createMockResponse(200, {}))
      .mockResolvedValueOnce(
        createMockResponse(200, { resultCode: '100', resultMsg: 'ok', ROUND: '303', DRAW_DATE: '2026-02-19' })
      )
      .mockResolvedValueOnce(
        createMockResponse(200, { q: 'enc:{"resultCode":"100","resultMsg":"조회 성공","deposit":"3000"}' })
      );

    const result = await reservePensionNextWeek(mockClient);

    expect(result).toMatchObject({
      status: 'failure',
      success: false,
      code: 'PENSION_INSUFFICIENT_DEPOSIT',
      targetRound: 304,
    });
    expect(fetchMock).toHaveBeenCalledTimes(4);
    expect(sendNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'warning',
        title: 'Pension Reserve Skipped',
      })
    );
  });

  it('should return failure on E001 auth error from duplicate check', async () => {
    fetchMock
      .mockResolvedValueOnce(createMockResponse(200, {}))
      .mockResolvedValueOnce(createMockResponse(200, {}))
      .mockResolvedValueOnce(
        createMockResponse(200, { resultCode: '100', resultMsg: 'ok', ROUND: '303', DRAW_DATE: '2026-02-19' })
      )
      .mockResolvedValueOnce(
        createMockResponse(200, { q: 'enc:{"resultCode":"100","resultMsg":"조회 성공","deposit":"10000"}' })
      )
      .mockResolvedValueOnce(
        createMockResponse(200, { q: 'enc:{"resultCode":"E001","resultMsg":"로그인후 이용하시기 바랍니다.","doubleRound":[]}' })
      );

    const result = await reservePensionNextWeek(mockClient);

    expect(result).toMatchObject({
      status: 'failure',
      success: false,
      code: 'E001',
      targetRound: 304,
    });
    expect(sendNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'error',
        title: 'Pension Reserve Failed',
      })
    );
  });

  it('should include DHLotteryError code in failure when bootstrap throws', async () => {
    fetchMock.mockRejectedValueOnce(
      new DHLotteryError('EL session failed', 'PENSION_BOOTSTRAP_FAILED')
    );

    const result = await reservePensionNextWeek(mockClient);

    expect(result).toMatchObject({
      status: 'failure',
      success: false,
      code: 'PENSION_BOOTSTRAP_FAILED',
    });
    expect(sendNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'error',
        details: expect.objectContaining({ 오류코드: 'PENSION_BOOTSTRAP_FAILED' }),
      })
    );
  });

  it('should use PENSION_UNEXPECTED_ERROR code when a generic error is thrown', async () => {
    fetchMock.mockRejectedValueOnce(new TypeError('fetch failed'));

    const result = await reservePensionNextWeek(mockClient);

    expect(result).toMatchObject({
      status: 'failure',
      success: false,
      code: 'PENSION_UNEXPECTED_ERROR',
    });
    expect(sendNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'error',
        details: expect.objectContaining({ 오류코드: 'PENSION_UNEXPECTED_ERROR' }),
      })
    );
  });
});
