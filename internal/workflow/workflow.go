// Package workflow orchestrates the end-to-end DHLottery run.
//
// It is non-throwing by design: non-critical failures are collected and
// reported as a single notification rather than aborting the run.
package workflow

import (
	"time"

	"github.com/kadragon/dhlottery-worker/internal/constants"
	"github.com/kadragon/dhlottery-worker/internal/dhlottery"
	"github.com/kadragon/dhlottery-worker/internal/env"
	"github.com/kadragon/dhlottery-worker/internal/format"
	"github.com/kadragon/dhlottery-worker/internal/notify"
)

// Client is the surface RunWorkflow needs. *dhlottery.Client satisfies it.
type Client interface {
	Login() error
	CheckDeposit(requiredAmount int) (bool, error)
	ReservePensionNextWeek() dhlottery.PensionReserveOutcome
	Buy() dhlottery.PurchaseOutcome
	CheckWinning(now time.Time) []dhlottery.WinningResult
	AggregateLedger(startDate string, now time.Time) dhlottery.LedgerSummary
	Collector() *notify.Collector
}

// SendCombined is the delivery seam (overridable in tests).
var SendCombined = notify.SendCombinedNotification

func workflowError(err error) notify.Payload {
	return notify.Payload{
		Type:    notify.Error,
		Title:   "워크플로 오류",
		Message: "워크플로우 실행 중 오류가 발생했습니다: " + err.Error(),
	}
}

// RunWorkflow runs the full pipeline once and returns false only if the final
// Telegram delivery fails after all retries; true otherwise.
func RunWorkflow(now time.Time, client Client) bool {
	collector := client.Collector()

	if err := client.Login(); err != nil {
		collector.Add(workflowError(err))
	} else {
		canPurchase, err := client.CheckDeposit(constants.WeeklyCombinedRequiredBalance)
		if err != nil {
			collector.Add(workflowError(err))
			canPurchase = false
		}

		var weeklyPurchase int
		if canPurchase {
			pension := client.ReservePensionNextWeek()
			purchase := client.Buy()
			weeklyPurchase = pension.TotalAmount + purchase.TotalAmount
		}

		wins := client.CheckWinning(now)
		summary := client.AggregateLedger(resolveLedgerStartDate(), now)

		collector.Add(buildSettlementPayload(weeklyPurchase, sumWins(wins), summary))
	}

	if !collector.IsEmpty() {
		return SendCombined(collector.Payloads())
	}
	return true
}

// resolveLedgerStartDate returns LEDGER_START_DATE (YYYYMMDD) when set, else
// the lifetime default. Env access stays behind internal/env (Golden Rule #1).
func resolveLedgerStartDate() string {
	if v, err := env.Get("LEDGER_START_DATE"); err == nil {
		return v
	}
	return constants.DefaultLedgerStartDate
}

// sumWins totals the prize amounts of the given wins (this week's winnings).
func sumWins(wins []dhlottery.WinningResult) int {
	total := 0
	for _, w := range wins {
		total += w.PrizeAmount
	}
	return total
}

// signedCurrency formats a net amount with an explicit leading "+" when
// positive. Negative amounts already render with "-" via format.Currency.
func signedCurrency(net int) string {
	if net > 0 {
		return "+" + format.Currency(net)
	}
	return format.Currency(net)
}

// buildSettlementPayload assembles the weekly 주간 결산 summary: this week's
// purchase/winning plus lifetime cumulative totals and the signed net.
func buildSettlementPayload(weeklyPurchase, weeklyWinning int, s dhlottery.LedgerSummary) notify.Payload {
	net := s.CumulativeWinning - s.CumulativePurchase
	return notify.Payload{
		Type:  notify.Success,
		Title: "주간 결산",
		Details: []notify.KV{
			{Key: "이번 주 구매", Value: format.Currency(weeklyPurchase)},
			{Key: "이번 주 당첨", Value: format.Currency(weeklyWinning)},
			{Key: "누적 구매", Value: format.Currency(s.CumulativePurchase)},
			{Key: "누적 당첨", Value: format.Currency(s.CumulativeWinning)},
			{Key: "결산", Value: signedCurrency(net)},
		},
	}
}
