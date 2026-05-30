package dhlottery

import (
	"net/http"
	"strconv"

	"github.com/kadragon/dhlottery-worker/internal/constants"
	"github.com/kadragon/dhlottery-worker/internal/format"
	"github.com/kadragon/dhlottery-worker/internal/httpclient"
	"github.com/kadragon/dhlottery-worker/internal/logger"
	"github.com/kadragon/dhlottery-worker/internal/notify"
)

// K-Bank virtual account charge initialization URL (access only, no payment).
const chargeInitURLBase = "https://www.dhlottery.co.kr/kbank.do?method=kbankProcess&PayMethod=VBANK&VBankAccountName=%EB%8F%99%ED%96%89%EB%B3%B5%EA%B6%8C&LicenseKey=&VBankExpDate=&GoodsAmt="

var chargeInitURL = chargeInitURLBase + strconv.Itoa(constants.ChargeAmount)

// initializeChargePage accesses the charge page without executing a payment.
func initializeChargePage(client *httpclient.Client) bool {
	resp, err := client.Fetch(chargeInitURL, httpclient.RequestOptions{
		Method:  http.MethodGet,
		Headers: map[string]string{constants.HeaderUserAgent: constants.UserAgent},
	})
	if err != nil {
		logger.Error("Error initializing charge page", logger.Fields{
			logger.FieldEvent: "charge_init_error", logger.FieldError: err.Error(),
		})
		return false
	}
	if resp.Status != 200 {
		logger.Error("Failed to initialize charge page", logger.Fields{
			logger.FieldEvent: "charge_init_failed", logger.FieldStatus: resp.Status,
		})
		return false
	}
	return true
}

// checkDeposit returns true if the balance meets requiredAmount. Otherwise it
// initializes the charge page and notifies, returning false. It returns an
// error only when account info cannot be retrieved (fail-safe).
func checkDeposit(client *httpclient.Client, requiredAmount int, collector *notify.Collector) (bool, error) {
	info, err := getAccountInfo(client)
	if err != nil {
		return false, err
	}

	if info.Balance >= requiredAmount {
		return true, nil
	}

	logger.Debug("Insufficient balance detected", logger.Fields{
		logger.FieldEvent: "insufficient_balance", "balance": info.Balance, "required": requiredAmount,
	})

	if !initializeChargePage(client) {
		notify.Notify(notify.Payload{
			Type:    notify.Error,
			Title:   "충전 초기화 실패",
			Message: "충전 페이지 초기화에 실패했습니다. 수동으로 입금해주세요.",
			Details: []notify.KV{
				{Key: "currentBalance", Value: format.Currency(info.Balance)},
				{Key: "minimumRequired", Value: format.Currency(requiredAmount)},
			},
		}, collector)
		return false, nil
	}

	notify.Notify(notify.Payload{
		Type:    notify.Warning,
		Title:   "예치금 부족",
		Message: "잔액이 부족하여 로또 구매를 진행할 수 없습니다. 입금 후 다음 스케줄에서 재시도됩니다.",
		Details: []notify.KV{
			{Key: "currentBalance", Value: format.Currency(info.Balance)},
			{Key: "minimumRequired", Value: format.Currency(requiredAmount)},
			{Key: "chargeAmount", Value: format.Currency(constants.ChargeAmount)},
		},
	}, collector)
	return false, nil
}
