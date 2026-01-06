import { getAccountInfo } from '../src/dhlottery/account';
import { DHLotteryClient } from '../src/dhlottery/client';

const BASE_URL = 'https://ol.dhlottery.co.kr/olotto/game';

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
      'User-Agent': 'Mozilla/5.0',
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
  });

  const buyResponse = await (client as any).client.fetch(`${BASE_URL}/execBuy.do`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded; charset=UTF-8',
      'User-Agent': 'Mozilla/5.0',
    },
    body: params.toString(),
  });

  console.log('Buy Status:', buyResponse.status);
  const buyText = await buyResponse.text('utf-8');
  console.log('Buy Response:', buyText.substring(0, 1000));
}

main().catch(console.error);
