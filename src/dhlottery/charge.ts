/**
 * Deposit Check and Charge Initialization Module
 *
 * Trace:
 *   spec_id: SPEC-DEPOSIT-001
 *   task_id: TASK-004, TASK-010, TASK-011
 */

import { CHARGE_AMOUNT, MIN_DEPOSIT_AMOUNT } from '../constants';
import { sendNotification } from '../notify/telegram';
import type { DepositEnv, HttpClient } from '../types';
import { getAccountInfo } from './account';

/**
 * K-Bank virtual account charge initialization URL
 * Discovered via Chrome MCP verification on 2025-12-16
 * Amount: 50,000 KRW
 */
const CHARGE_INIT_URL = `https://www.dhlottery.co.kr/kbank.do?method=kbankProcess&PayMethod=VBANK&VBankAccountName=%EB%8F%99%ED%96%89%EB%B3%B5%EA%B6%8C&LicenseKey=&VBankExpDate=&GoodsAmt=${CHARGE_AMOUNT}`;

/**
 * Format number with thousands separator
 */
function formatCurrency(amount: number): string {
  return `${amount.toLocaleString('ko-KR')}원`;
}

/**
 * Initialize charge page (access but do not execute payment)
 */
async function initializeChargePage(client: HttpClient): Promise<boolean> {
  try {
    const response = await client.fetch(CHARGE_INIT_URL, {
      method: 'GET',
    });

    if (response.status !== 200) {
      console.error(`Failed to initialize charge page: HTTP ${response.status}`);
      return false;
    }

    return true;
  } catch (error) {
    console.error('Error initializing charge page:', error);
    return false;
  }
}

/**
 * Check deposit balance and initialize charge if needed
 *
 * @param client - HTTP client with authenticated session
 * @param env - Environment containing Telegram credentials
 * @returns true if purchase can proceed, false if charge is needed
 * @throws Error if account info cannot be retrieved (fail-safe)
 */
export async function checkDeposit(client: HttpClient, env: DepositEnv): Promise<boolean> {
  // Fetch current account information
  const accountInfo = await getAccountInfo(client);

  // Check if balance is sufficient
  if (accountInfo.balance >= MIN_DEPOSIT_AMOUNT) {
    // Balance is sufficient - proceed with purchase
    return true;
  }

  // Balance is insufficient - initialize charge and notify
  console.log(`Insufficient balance: ${accountInfo.balance} < ${MIN_DEPOSIT_AMOUNT}`);

  // Initialize charge page
  const chargeSuccess = await initializeChargePage(client);

  if (!chargeSuccess) {
    // Charge initialization failed - send error notification
    await sendNotification(
      {
        type: 'error',
        title: 'Charge Initialization Failed',
        message: '충전 페이지 초기화에 실패했습니다. 수동으로 입금해주세요.',
        details: {
          currentBalance: formatCurrency(accountInfo.balance),
          minimumRequired: formatCurrency(MIN_DEPOSIT_AMOUNT),
        },
      },
      env
    );

    return false;
  }

  // Send warning notification about insufficient balance
  await sendNotification(
    {
      type: 'warning',
      title: 'Insufficient Balance',
      message:
        '잔액이 부족하여 로또 구매를 진행할 수 없습니다. 입금 후 다음 스케줄에서 재시도됩니다.',
      details: {
        currentBalance: formatCurrency(accountInfo.balance),
        minimumRequired: formatCurrency(MIN_DEPOSIT_AMOUNT),
        chargeAmount: formatCurrency(CHARGE_AMOUNT),
      },
    },
    env
  );

  // Block purchase
  return false;
}
