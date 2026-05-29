// Package workflow orchestrates the end-to-end DHLottery run.
//
// It is non-throwing by design: non-critical failures are collected and
// reported as a single notification rather than aborting the run.
package workflow

import (
	"time"

	"github.com/kadragon/dhlottery-worker/internal/constants"
	"github.com/kadragon/dhlottery-worker/internal/dhlottery"
	"github.com/kadragon/dhlottery-worker/internal/notify"
)

// Client is the surface RunWorkflow needs. *dhlottery.Client satisfies it.
type Client interface {
	Login() error
	CheckDeposit(requiredAmount int) (bool, error)
	ReservePensionNextWeek() dhlottery.PensionReserveOutcome
	Buy() dhlottery.PurchaseOutcome
	CheckWinning(now time.Time) []dhlottery.WinningResult
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

		if canPurchase {
			client.ReservePensionNextWeek()
			client.Buy()
		}

		client.CheckWinning(now)
	}

	if !collector.IsEmpty() {
		return SendCombined(collector.Payloads())
	}
	return true
}
