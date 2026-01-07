/**
 * Formatting utilities
 */

export function formatKoreanNumber(amount: number): string {
  return amount.toLocaleString('ko-KR');
}

export function formatCurrency(amount: number): string {
  return `${formatKoreanNumber(amount)}원`;
}
