import { getAccountInfo } from '../src/dhlottery/account';
import { DHLotteryClient } from '../src/dhlottery/client';
import { USER_AGENT } from '../src/constants';
import { getNextSaturdayKst } from '../src/utils/date';

const BASE_URL = 'https://ol.dhlottery.co.kr/olotto/game';
const GAME_PAGE_URL = `${BASE_URL}/game645.do`;
const AJAX_HEADERS = {
  Origin: 'https://ol.dhlottery.co.kr',
  Referer: GAME_PAGE_URL,
  'X-Requested-With': 'XMLHttpRequest',
} as const;

function formatDateWithSlashes(date: string): string {
  return date.replace(/-/g, '/');
}

function addYearsAndDays(date: string, years: number, days: number): string {
  const [year, month, day] = date.split('-').map((part) => Number(part));
  const base = new Date(Date.UTC(year, month - 1, day));
  base.setUTCFullYear(base.getUTCFullYear() + years);
  base.setUTCDate(base.getUTCDate() + days);
  const resultYear = base.getUTCFullYear();
  const resultMonth = String(base.getUTCMonth() + 1).padStart(2, '0');
  const resultDay = String(base.getUTCDate()).padStart(2, '0');
  return `${resultYear}-${resultMonth}-${resultDay}`;
}

async function main() {
  const client = new DHLotteryClient();

  console.log('=== Login ===');
  await client.login();

  console.log('\n=== Get Account Info ===');
  const accountInfo = await getAccountInfo((client as any).client);
  console.log('Round:', accountInfo.currentRound);
  console.log('Balance:', accountInfo.balance);

  console.log('\n=== Prepare Purchase ===');
  const readyResponse = await (client as any).client.fetch(`${BASE_URL}/egovUserReadySocket.json`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json; charset=UTF-8',
      'User-Agent': USER_AGENT,
      ...AJAX_HEADERS,
    },
  });
  console.log('Ready Status:', readyResponse.status);
  const readyText = await readyResponse.text('utf-8');
  console.log('Ready Response:', readyText.substring(0, 500));

  let readyData: { ready_ip: string } | null = null;
  try {
    readyData = JSON.parse(readyText);
  } catch {
    console.log('Ready response is not JSON!');
    return;
  }

  if (!readyData?.ready_ip) {
    console.log('Ready response missing ready_ip');
    return;
  }

  console.log('\n=== Execute Purchase ===');
  const drawDate = getNextSaturdayKst();
  const payLimitDate = addYearsAndDays(drawDate, 1, 1);
  const games = ['A', 'B', 'C', 'D', 'E'].map((alpabet) => ({
    genType: '0',
    arrGameChoiceNum: null,
    alpabet,
  }));

  const params = new URLSearchParams({
    round: accountInfo.currentRound.toString(),
    direct: readyData.ready_ip,
    nBuyAmount: '5000',
    param: JSON.stringify(games),
    gameCnt: '5',
    saleMdaDcd: '10',
    ROUND_DRAW_DATE: formatDateWithSlashes(drawDate),
    WAMT_PAY_TLMT_END_DT: formatDateWithSlashes(payLimitDate),
  });

  const buyResponse = await (client as any).client.fetch(`${BASE_URL}/execBuy.do`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded; charset=UTF-8',
      'User-Agent': USER_AGENT,
      ...AJAX_HEADERS,
    },
    body: params.toString(),
  });

  console.log('Buy Status:', buyResponse.status);
  const buyText = await buyResponse.text('utf-8');
  console.log('Buy Response:', buyText.substring(0, 1000));
}

main().catch(console.error);
