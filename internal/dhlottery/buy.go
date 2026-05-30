package dhlottery

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kadragon/dhlottery-worker/internal/constants"
	"github.com/kadragon/dhlottery-worker/internal/datekst"
	"github.com/kadragon/dhlottery-worker/internal/dherr"
	"github.com/kadragon/dhlottery-worker/internal/format"
	"github.com/kadragon/dhlottery-worker/internal/httpclient"
	"github.com/kadragon/dhlottery-worker/internal/logger"
	"github.com/kadragon/dhlottery-worker/internal/notify"
)

const (
	buyBaseURL    = "https://ol.dhlottery.co.kr/olotto/game"
	gamePageURL   = buyBaseURL + "/game645.do"
	titleBuyFail  = "로또 구매 실패"
	detailErrCode = "오류코드"
)

func buyAjaxHeaders(contentType string) map[string]string {
	return map[string]string{
		constants.HeaderContentType:    contentType,
		constants.HeaderUserAgent:      constants.UserAgent,
		"Origin":                       "https://ol.dhlottery.co.kr",
		constants.HeaderReferer:        gamePageURL,
		constants.HeaderXRequestedWith: constants.HeaderXRequestedWithValue,
	}
}

func preparePurchase(client *httpclient.Client) (PurchaseReadyResponse, error) {
	var ready PurchaseReadyResponse
	resp, err := client.Fetch(buyBaseURL+"/egovUserReadySocket.json", httpclient.RequestOptions{
		Method:  http.MethodPost,
		Headers: buyAjaxHeaders("application/json; charset=UTF-8"),
	})
	if err != nil {
		return ready, err
	}
	if resp.Status != 200 {
		return ready, dherr.NewPurchase("Purchase ready failed: "+strconv.Itoa(resp.Status), "PURCHASE_READY_FAILED")
	}
	if err := resp.JSON(&ready); err != nil {
		return ready, err
	}
	return ready, nil
}

func executePurchase(client *httpclient.Client, ready PurchaseReadyResponse, roundNumber int) (purchaseResult, error) {
	var result purchaseResult

	drawDate := datekst.NextSaturdayKst(time.Now())
	payLimitDate := datekst.AddYearsAndDays(drawDate, 1, 1)

	games := make([]gameSelection, 0, constants.GamesPerPurchase)
	for i := 0; i < constants.GamesPerPurchase; i++ {
		games = append(games, gameSelection{GenType: "0", ArrGameChoiceNum: nil, Alpabet: "ABCDEFGHIJKLMNOPQRSTUVWXYZ"[i : i+1]})
	}
	gamesJSON, err := json.Marshal(games)
	if err != nil {
		return result, err
	}

	params := url.Values{}
	params.Set("round", strconv.Itoa(roundNumber))
	params.Set("direct", ready.ReadyIP)
	params.Set("nBuyAmount", strconv.Itoa(constants.TotalPurchaseCost))
	params.Set("param", string(gamesJSON))
	params.Set("gameCnt", strconv.Itoa(constants.GamesPerPurchase))
	params.Set("saleMdaDcd", "10")
	params.Set("ROUND_DRAW_DATE", formatDateWithSlashes(drawDate))
	params.Set("WAMT_PAY_TLMT_END_DT", formatDateWithSlashes(payLimitDate))

	logger.Debug("Purchase parameters", logger.Fields{
		logger.FieldModule: "buy", "round": roundNumber, "direct": ready.ReadyIP,
		"nBuyAmount": constants.TotalPurchaseCost, "gameCnt": constants.GamesPerPurchase,
	})

	resp, err := client.Fetch(buyBaseURL+"/execBuy.do", httpclient.RequestOptions{
		Method:  http.MethodPost,
		Headers: buyAjaxHeaders("application/x-www-form-urlencoded; charset=UTF-8"),
		Body:    params.Encode(),
	})
	if err != nil {
		return result, err
	}
	if resp.Status != 200 {
		return result, dherr.NewPurchase("Purchase execution failed: "+strconv.Itoa(resp.Status), "PURCHASE_EXECUTION_FAILED")
	}
	if err := resp.JSON(&result); err != nil {
		return result, err
	}
	return result, nil
}

func formatDateWithSlashes(date string) string {
	return strings.ReplaceAll(date, "-", "/")
}

// purchaseLottery buys 5 auto-pick games. It never returns an error; failures
// are reported via the returned outcome and a notification.
func purchaseLottery(client *httpclient.Client, collector *notify.Collector) PurchaseOutcome {
	info, err := getAccountInfo(client)
	if err != nil {
		return purchaseFailure(err.Error(), collector)
	}
	roundNumber := info.CurrentRound

	ready, err := preparePurchase(client)
	if err != nil {
		return purchaseFailure(err.Error(), collector)
	}

	result, err := executePurchase(client, ready, roundNumber)
	if err != nil {
		return purchaseFailure(err.Error(), collector)
	}

	if result.Result.ResultCode == constants.PurchaseSuccessCode {
		notify.Notify(notify.Payload{
			Type:    notify.Success,
			Title:   "로또 구매 완료",
			Message: fmt.Sprintf("%d회 로또 구매를 완료했습니다.", roundNumber),
			Details: []notify.KV{
				{Key: "회차", Value: fmt.Sprintf("%d회", roundNumber)},
				{Key: "결제금액", Value: format.KoreanNumber(constants.TotalPurchaseCost) + "원"},
				{Key: "잔액", Value: format.Currency(info.Balance - constants.TotalPurchaseCost)},
			},
		}, collector)

		return PurchaseOutcome{
			Success:      true,
			RoundNumber:  roundNumber,
			GameCount:    constants.GamesPerPurchase,
			TotalAmount:  constants.TotalPurchaseCost,
			PurchaseDate: time.Now().UTC().Format(time.RFC3339),
			Message:      result.Result.ResultMsg,
		}
	}

	notify.Notify(notify.Payload{
		Type:    notify.Error,
		Title:   titleBuyFail,
		Message: result.Result.ResultMsg,
		Details: []notify.KV{{Key: detailErrCode, Value: result.Result.ResultCode}},
	}, collector)

	return PurchaseOutcome{
		Success: false,
		Error:   result.Result.ResultMsg,
		Code:    result.Result.ResultCode,
	}
}

func purchaseFailure(message string, collector *notify.Collector) PurchaseOutcome {
	notify.Notify(notify.Payload{
		Type:    notify.Error,
		Title:   titleBuyFail,
		Message: "구매 중 오류가 발생했습니다: " + message,
	}, collector)
	return PurchaseOutcome{Success: false, Error: message}
}
