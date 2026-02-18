/**
 * Pension 720+ next-week reserve module
 */

import { PENSION_RESERVE_COST, USER_AGENT } from '../constants';
import { sendNotification } from '../notify/telegram';
import type {
  ElAddMyReserveResponse,
  ElCheckMyReserveResponse,
  ElDepositResponse,
  ElEncryptedResponse,
  ElRoundRemainTimeResponse,
  HttpClient,
  PensionReserveFailure,
  PensionReserveOutcome,
  PensionReserveSkipped,
  PensionReserveSuccess,
} from '../types';
import { DHLotteryError } from '../utils/errors';
import { formatKoreanNumber } from '../utils/format';
import { logger } from '../utils/logger';
import { decryptElQ, encryptElQ } from './pension-crypto';

const EL_BASE_URL = 'https://el.dhlottery.co.kr';
const TOTAL_GAME_URL = `${EL_BASE_URL}/game/TotalGame.jsp?LottoId=LP72`;
const RESERVE_PAGE_URL = `${EL_BASE_URL}/game/pension720/reserveGame.jsp`;
const ROUND_REMAIN_TIME_URL = `${EL_BASE_URL}/roundRemainTime.do`;
const CHECK_DEPOSIT_URL = `${EL_BASE_URL}/checkDeposit.do`;
const CHECK_MY_RESERVE_URL = `${EL_BASE_URL}/checkMyReserve.do`;
const ADD_MY_RESERVE_URL = `${EL_BASE_URL}/addMyReserve.do`;
// Empty form serialized from the reservation page's frmAuto; accepted as-is by
// roundRemainTime.do and checkDeposit.do for session-scoped queries.
const FRM_AUTO_SERIALIZED = 'ROUND=&SEL_NO=&BUY_CNT=&AUTO_SEL_SET=&SEL_CLASS=&ACCS_TYPE=01';
const TICKET_COUNT = 5;

function createAjaxHeaders(referer: string): Record<string, string> {
  return {
    'Content-Type': 'application/x-www-form-urlencoded; charset=UTF-8',
    'User-Agent': USER_AGENT,
    Origin: EL_BASE_URL,
    Referer: referer,
    'X-Requested-With': 'XMLHttpRequest',
  };
}

function getSessionId(client: HttpClient): string {
  // el.dhlottery.co.kr uses JSESSIONID; fall back to DHJSESSIONID (main site)
  // in case the EL bootstrap sets its own session under a different key.
  const sessionId = client.cookies.JSESSIONID || client.cookies.DHJSESSIONID;
  if (!sessionId) {
    throw new DHLotteryError('Missing session cookie for pension reserve', 'PENSION_AUTH_MISSING');
  }
  logger.debug('Using session cookie for EL encryption', {
    cookieKey: client.cookies.JSESSIONID ? 'JSESSIONID' : 'DHJSESSIONID',
  });
  return sessionId;
}

function addDaysToYmd(dateStr: string, days: number): string {
  const normalized = dateStr.includes('-')
    ? dateStr
    : `${dateStr.slice(0, 4)}-${dateStr.slice(4, 6)}-${dateStr.slice(6, 8)}`;
  const [year, month, day] = normalized.split('-').map((part) => Number(part));
  // Date.UTC month is 0-indexed (0 = January), so subtract 1 from the parsed month.
  const utcDate = new Date(Date.UTC(year, month - 1, day));
  utcDate.setUTCDate(utcDate.getUTCDate() + days);
  const nextYear = utcDate.getUTCFullYear();
  const nextMonth = String(utcDate.getUTCMonth() + 1).padStart(2, '0');
  const nextDay = String(utcDate.getUTCDate()).padStart(2, '0');
  return `${nextYear}-${nextMonth}-${nextDay}`;
}

function toDotDate(ymd: string): string {
  return ymd.replace(/-/g, '.');
}

function buildReserveFormPayload(params: {
  currentRound: number;
  nextRound: number;
  nextDrawDateDot: string;
  deposit: number;
  winDate: string;
}): string {
  return new URLSearchParams({
    ROUND: String(params.currentRound),
    reserveJo: '0',
    repeatRoundCnt: '1',
    totalBuyAmt: String(PENSION_RESERVE_COST),
    totalBuyCnt: String(TICKET_COUNT),
    moneyBalance: '0/', // trailing slash is the site's serialized format (not a typo)
    couponBalance: '0/', // trailing slash is the site's serialized format (not a typo)
    nextRound: String(params.nextRound),
    repeatClass: '5',
    roundBuyCnt: '1',
    curdeposit: String(params.deposit),
    curpay: String(PENSION_RESERVE_COST),
    winDate: params.winDate,
    WORKING_FLAG: 'false',
    repeatRound: '1',
    repeatRoundHidden: params.nextDrawDateDot,
  }).toString();
}

async function bootstrapElSession(client: HttpClient): Promise<void> {
  const totalGameResponse = await client.fetch(TOTAL_GAME_URL, {
    headers: {
      'User-Agent': USER_AGENT,
    },
  });

  if (totalGameResponse.status !== 200) {
    throw new DHLotteryError(
      `Failed to load EL total game page: HTTP ${totalGameResponse.status}`,
      'PENSION_BOOTSTRAP_FAILED'
    );
  }

  const reservePageResponse = await client.fetch(RESERVE_PAGE_URL, {
    headers: {
      'User-Agent': USER_AGENT,
      Referer: TOTAL_GAME_URL,
    },
  });

  if (reservePageResponse.status !== 200) {
    throw new DHLotteryError(
      `Failed to load EL reserve page: HTTP ${reservePageResponse.status}`,
      'PENSION_BOOTSTRAP_FAILED'
    );
  }
}

async function fetchRoundRemainTime(client: HttpClient): Promise<{
  currentRound: number;
  nextRound: number;
  nextDrawDateDot: string;
}> {
  const response = await client.fetch(ROUND_REMAIN_TIME_URL, {
    method: 'POST',
    headers: createAjaxHeaders(RESERVE_PAGE_URL),
    body: FRM_AUTO_SERIALIZED,
  });

  if (response.status !== 200) {
    throw new DHLotteryError(
      `Failed to fetch round remain time: HTTP ${response.status}`,
      'PENSION_ROUND_FETCH_FAILED'
    );
  }

  const data = (await response.json()) as ElRoundRemainTimeResponse;

  if (data.resultCode !== '100') {
    throw new DHLotteryError(
      `Round remain time API failed: ${data.resultCode} ${data.resultMsg}`,
      data.resultCode
    );
  }

  const currentRound = Number(data.ROUND);
  if (!Number.isInteger(currentRound) || currentRound <= 0) {
    throw new DHLotteryError(`Invalid round value: ${data.ROUND}`, 'PENSION_INVALID_ROUND');
  }

  const nextDrawDate = addDaysToYmd(data.DRAW_DATE, 7);
  return {
    currentRound,
    nextRound: currentRound + 1, // pension draws are weekly and sequential
    nextDrawDateDot: toDotDate(nextDrawDate),
  };
}

async function postEncrypted<T>(
  client: HttpClient,
  url: string,
  plainForm: string,
  referer: string
): Promise<T> {
  const sessionId = getSessionId(client);
  const encryptedQ = encryptElQ(plainForm, sessionId);

  const response = await client.fetch(url, {
    method: 'POST',
    headers: createAjaxHeaders(referer),
    body: new URLSearchParams({ q: encryptedQ }).toString(),
  });

  if (response.status !== 200) {
    throw new DHLotteryError(
      `EL API request failed: HTTP ${response.status}`,
      'PENSION_API_FAILED'
    );
  }

  const encrypted = (await response.json()) as ElEncryptedResponse;
  if (!encrypted.q) {
    throw new DHLotteryError(
      'EL API response is missing q payload',
      'PENSION_API_INVALID_RESPONSE'
    );
  }

  const decrypted = decryptElQ(encrypted.q, sessionId);
  return JSON.parse(decrypted) as T;
}

function createFailure(error: string, code?: string, targetRound?: number): PensionReserveFailure {
  return {
    status: 'failure',
    success: false,
    skipped: false,
    targetRound,
    error,
    code,
  };
}

export async function reservePensionNextWeek(client: HttpClient): Promise<PensionReserveOutcome> {
  let targetRound: number | undefined;

  try {
    await bootstrapElSession(client);

    const roundInfo = await fetchRoundRemainTime(client);
    targetRound = roundInfo.nextRound;

    const depositData = await postEncrypted<ElDepositResponse>(
      client,
      CHECK_DEPOSIT_URL,
      FRM_AUTO_SERIALIZED,
      RESERVE_PAGE_URL
    );

    if (depositData.resultCode !== '100') {
      const failure = createFailure(depositData.resultMsg, depositData.resultCode, targetRound);
      await sendNotification({
        type: 'error',
        title: 'Pension Reserve Failed',
        message: `연금복권 예치금 조회에 실패했습니다: ${depositData.resultMsg}`,
        details: {
          오류코드: depositData.resultCode,
          대상회차: targetRound ? `${targetRound}회` : undefined,
        },
      });
      return failure;
    }

    const deposit = Number(depositData.deposit);
    if (!Number.isFinite(deposit)) {
      const failure = createFailure('Invalid deposit value from EL API', 'PENSION_INVALID_DEPOSIT');
      await sendNotification({
        type: 'error',
        title: 'Pension Reserve Failed',
        message: '연금복권 예치금 값을 파싱하지 못했습니다.',
      });
      return failure;
    }

    if (deposit < PENSION_RESERVE_COST) {
      const failure = createFailure(
        'Insufficient balance for pension reserve',
        'PENSION_INSUFFICIENT_DEPOSIT',
        targetRound
      );
      await sendNotification({
        type: 'warning',
        title: 'Pension Reserve Skipped',
        message: '연금복권 예약에 필요한 예치금이 부족하여 예약을 건너뜁니다.',
        details: {
          대상회차: `${targetRound}회`,
          필요금액: `${formatKoreanNumber(PENSION_RESERVE_COST)}원`,
          보유예치금: `${formatKoreanNumber(deposit)}원`,
        },
      });
      return failure;
    }

    const duplicatePayload = buildReserveFormPayload({
      currentRound: roundInfo.currentRound,
      nextRound: roundInfo.nextRound,
      nextDrawDateDot: roundInfo.nextDrawDateDot,
      deposit,
      winDate: '',
    });

    const duplicateData = await postEncrypted<ElCheckMyReserveResponse>(
      client,
      CHECK_MY_RESERVE_URL,
      duplicatePayload,
      RESERVE_PAGE_URL
    );

    if (duplicateData.resultCode !== '100') {
      const failure = createFailure(duplicateData.resultMsg, duplicateData.resultCode, targetRound);
      await sendNotification({
        type: 'error',
        title: 'Pension Reserve Failed',
        message: `연금복권 중복 예약 확인에 실패했습니다: ${duplicateData.resultMsg}`,
        details: {
          오류코드: duplicateData.resultCode,
          대상회차: `${targetRound}회`,
        },
      });
      return failure;
    }

    const duplicates = (duplicateData.doubleRound || [])
      .filter((item) => Number(item.doubleRound) === targetRound)
      .map((item) => `${item.doubleRound}회 ${item.doubleCnt}매`);

    if (duplicates.length > 0) {
      const skipped: PensionReserveSkipped = {
        status: 'skipped',
        success: true,
        skipped: true,
        targetRound,
        totalAmount: PENSION_RESERVE_COST,
        ticketCount: TICKET_COUNT,
        message: 'Duplicate reserve detected',
        duplicateRounds: duplicates,
      };

      await sendNotification({
        type: 'warning',
        title: 'Pension Reserve Skipped',
        message: `대상 회차(${targetRound}회)가 이미 예약되어 예약을 건너뜁니다.`,
        details: {
          중복회차: duplicates.join(', '),
        },
      });
      return skipped;
    }

    const reservePayload = buildReserveFormPayload({
      currentRound: roundInfo.currentRound,
      nextRound: roundInfo.nextRound,
      nextDrawDateDot: roundInfo.nextDrawDateDot,
      deposit,
      winDate: roundInfo.nextDrawDateDot,
    });

    const reserveData = await postEncrypted<ElAddMyReserveResponse>(
      client,
      ADD_MY_RESERVE_URL,
      reservePayload,
      RESERVE_PAGE_URL
    );

    if (reserveData.resultCode !== '100') {
      const failure = createFailure(reserveData.resultMsg, reserveData.resultCode, targetRound);
      await sendNotification({
        type: 'error',
        title: 'Pension Reserve Failed',
        message: `연금복권 예약 요청에 실패했습니다: ${reserveData.resultMsg}`,
        details: {
          오류코드: reserveData.resultCode,
          대상회차: `${targetRound}회`,
        },
      });
      return failure;
    }

    const success: PensionReserveSuccess = {
      status: 'success',
      success: true,
      skipped: false,
      targetRound,
      totalAmount: PENSION_RESERVE_COST,
      ticketCount: TICKET_COUNT,
      message: reserveData.resultMsg,
      reserveOrderNo: reserveData.reserveOrderNo,
      reserveOrderDate: reserveData.reserveOrderDate,
    };

    await sendNotification({
      type: 'success',
      title: 'Pension Reserve Completed',
      message: `${targetRound}회 연금복권720+ 예약(모든조, 1매씩)을 완료했습니다.`,
      details: {
        대상회차: `${targetRound}회`,
        예약금액: `${formatKoreanNumber(PENSION_RESERVE_COST)}원`,
        예약수량: `${TICKET_COUNT}매`,
        예약번호: reserveData.reserveOrderNo,
      },
    });

    return success;
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    // DHLotteryError carries a structured code from the API or our own error constants;
    // generic runtime errors (TypeError, SyntaxError, etc.) get a fallback code.
    const errorCode =
      error instanceof DHLotteryError ? error.code : 'PENSION_UNEXPECTED_ERROR';
    await sendNotification({
      type: 'error',
      title: 'Pension Reserve Failed',
      message: `연금복권 예약 중 오류가 발생했습니다: ${errorMessage}`,
      details: {
        오류코드: errorCode,
        대상회차: targetRound ? `${targetRound}회` : undefined,
      },
    });
    return createFailure(errorMessage, errorCode, targetRound);
  }
}
